package database

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/hydra13/gophkeeper/internal/models"
	"github.com/hydra13/gophkeeper/internal/repositories"
	cryptosvc "github.com/hydra13/gophkeeper/internal/services/crypto"
)

const (
	chunkPathTemplate   = "uploads/%d/%d.chunk"
	payloadPathTemplate = "payloads/%d/%d.bin"
)

// Repository реализация PostgreSQL хранилища.
type Repository struct {
	db     *sql.DB
	blob   repositories.BlobStorage
	crypto cryptosvc.CryptoService
}

// New создаёт PostgreSQL репозиторий.
func New(db *sql.DB, blob repositories.BlobStorage) (*Repository, error) {
	if db == nil {
		return nil, errors.New("db instance is required")
	}
	if blob == nil {
		return nil, errors.New("blob storage is required")
	}
	return &Repository{db: db, blob: blob}, nil
}

// SetCrypto задаёт сервис шифрования для repository.
func (r *Repository) SetCrypto(service cryptosvc.CryptoService) {
	r.crypto = service
}

func (r *Repository) ensureCrypto() (cryptosvc.CryptoService, error) {
	if r.crypto == nil {
		return nil, errors.New("crypto service is required")
	}
	return r.crypto, nil
}

// CreateUser сохраняет нового пользователя.
func (r *Repository) CreateUser(user *models.User) error {
	if user == nil {
		return errors.New("user is nil")
	}
	row := r.db.QueryRowContext(context.Background(), `
		INSERT INTO users (email, password_hash, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`, user.Email, user.PasswordHash)
	if err := row.Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt); err != nil {
		if isUniqueViolation(err) {
			return models.ErrEmailAlreadyExists
		}
		return err
	}
	return nil
}

// GetUserByEmail возвращает пользователя по email.
func (r *Repository) GetUserByEmail(email string) (*models.User, error) {
	row := r.db.QueryRowContext(context.Background(), `
		SELECT id, email, password_hash, created_at, updated_at
		FROM users
		WHERE email = $1
	`, email)
	return scanUser(row)
}

// GetUserByID возвращает пользователя по ID.
func (r *Repository) GetUserByID(id int64) (*models.User, error) {
	row := r.db.QueryRowContext(context.Background(), `
		SELECT id, email, password_hash, created_at, updated_at
		FROM users
		WHERE id = $1
	`, id)
	return scanUser(row)
}

// CreateRecord сохраняет запись секрета.
func (r *Repository) CreateRecord(record *models.Record) error {
	if record == nil {
		return errors.New("record is nil")
	}
	crypto, err := r.ensureCrypto()
	if err != nil {
		return err
	}
	payload, payloadData, err := splitRecordPayload(record)
	if err != nil {
		return err
	}
	if len(payload) > 0 {
		payload, err = encryptJSONPayload(crypto, payload, record.KeyVersion)
		if err != nil {
			return err
		}
	}
	if len(payloadData) > 0 {
		payloadData, err = crypto.Encrypt(payloadData, record.KeyVersion)
		if err != nil {
			return err
		}
	}

	tx, err := r.db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	row := tx.QueryRowContext(context.Background(), `
		INSERT INTO records (
			user_id, type, name, metadata, payload, revision, deleted_at,
			device_id, key_version, payload_version, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW())
		RETURNING id, created_at, updated_at, deleted_at
	`,
		record.UserID,
		string(record.Type),
		record.Name,
		record.Metadata,
		payload,
		record.Revision,
		record.DeletedAt,
		record.DeviceID,
		record.KeyVersion,
		record.PayloadVersion,
	)
	var deletedAt sql.NullTime
	if scanErr := row.Scan(&record.ID, &record.CreatedAt, &record.UpdatedAt, &deletedAt); scanErr != nil {
		_ = tx.Rollback()
		return scanErr
	}
	if deletedAt.Valid {
		record.DeletedAt = &deletedAt.Time
	}

	if len(payloadData) > 0 {
		storagePath := fmt.Sprintf(payloadPathTemplate, record.ID, record.PayloadVersion)
		if saveErr := r.blob.Save(storagePath, payloadData); saveErr != nil {
			_ = tx.Rollback()
			return saveErr
		}
		_, err = tx.ExecContext(context.Background(), `
			INSERT INTO payloads (record_id, version, storage_path, size, created_at)
			VALUES ($1, $2, $3, $4, NOW())
		`, record.ID, record.PayloadVersion, storagePath, int64(len(payloadData)))
		if err != nil {
			_ = tx.Rollback()
			_ = r.blob.Delete(storagePath)
			return err
		}
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

// GetRecord возвращает запись по ID.
func (r *Repository) GetRecord(id int64) (*models.Record, error) {
	row := r.db.QueryRowContext(context.Background(), `
		SELECT id, user_id, type, name, metadata, payload, revision, deleted_at,
		       device_id, key_version, payload_version, created_at, updated_at
		FROM records
		WHERE id = $1
	`, id)
	return r.scanRecord(row)
}

// ListRecords возвращает список записей пользователя с опциональной фильтрацией.
// Если recordType не пустой — фильтрует по типу.
// Если includeDeleted — включает soft-deleted записи.
func (r *Repository) ListRecords(userID int64, recordType models.RecordType, includeDeleted bool) ([]models.Record, error) {
	if _, err := r.ensureCrypto(); err != nil {
		return nil, err
	}

	query := `
		SELECT id, user_id, type, name, metadata, payload, revision, deleted_at,
		       device_id, key_version, payload_version, created_at, updated_at
		FROM records
		WHERE user_id = $1`
	args := []interface{}{userID}
	argIdx := 2

	if !includeDeleted {
		query += fmt.Sprintf(" AND deleted_at IS NULL")
	}
	if recordType != "" {
		query += fmt.Sprintf(" AND type = $%d", argIdx)
		args = append(args, string(recordType))
		argIdx++
	}
	query += " ORDER BY updated_at DESC"

	rows, err := r.db.QueryContext(context.Background(), query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []models.Record
	for rows.Next() {
		record, scanErr := r.scanRecord(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		records = append(records, *record)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return records, nil
}

// ListRecordsForReencrypt возвращает записи, зашифрованные неактуальным ключом.
func (r *Repository) ListRecordsForReencrypt(activeVersion int64, limit int) ([]models.Record, error) {
	if _, err := r.ensureCrypto(); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.QueryContext(context.Background(), `
		SELECT id, user_id, type, name, metadata, payload, revision, deleted_at,
		       device_id, key_version, payload_version, created_at, updated_at
		FROM records
		WHERE key_version <> $1 AND deleted_at IS NULL
		ORDER BY id ASC
		LIMIT $2
	`, activeVersion, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []models.Record
	for rows.Next() {
		record, scanErr := r.scanRecord(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		records = append(records, *record)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return records, nil
}

// UpdateRecord обновляет существующую запись.
func (r *Repository) UpdateRecord(record *models.Record) error {
	if record == nil {
		return errors.New("record is nil")
	}
	crypto, err := r.ensureCrypto()
	if err != nil {
		return err
	}
	payload, payloadData, err := splitRecordPayload(record)
	if err != nil {
		return err
	}
	if len(payload) > 0 {
		payload, err = encryptJSONPayload(crypto, payload, record.KeyVersion)
		if err != nil {
			return err
		}
	}
	if len(payloadData) > 0 {
		payloadData, err = crypto.Encrypt(payloadData, record.KeyVersion)
		if err != nil {
			return err
		}
	}

	tx, err := r.db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	result, err := tx.ExecContext(context.Background(), `
		UPDATE records
		SET type = $2,
		    name = $3,
		    metadata = $4,
		    payload = $5,
		    revision = $6,
		    deleted_at = $7,
		    device_id = $8,
		    key_version = $9,
		    payload_version = $10,
		    updated_at = NOW()
		WHERE id = $1
	`, record.ID,
		string(record.Type),
		record.Name,
		record.Metadata,
		payload,
		record.Revision,
		record.DeletedAt,
		record.DeviceID,
		record.KeyVersion,
		record.PayloadVersion,
	)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	if affected == 0 {
		_ = tx.Rollback()
		return models.ErrRecordNotFound
	}

	if len(payloadData) > 0 {
		storagePath := fmt.Sprintf(payloadPathTemplate, record.ID, record.PayloadVersion)
		if saveErr := r.blob.Save(storagePath, payloadData); saveErr != nil {
			_ = tx.Rollback()
			return saveErr
		}
		_, err = tx.ExecContext(context.Background(), `
			INSERT INTO payloads (record_id, version, storage_path, size, created_at)
			VALUES ($1, $2, $3, $4, NOW())
			ON CONFLICT (record_id, version)
			DO UPDATE SET storage_path = EXCLUDED.storage_path, size = EXCLUDED.size
		`, record.ID, record.PayloadVersion, storagePath, int64(len(payloadData)))
		if err != nil {
			_ = tx.Rollback()
			_ = r.blob.Delete(storagePath)
			return err
		}
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

// ListPayloads возвращает сохранённые payloads записи.
func (r *Repository) ListPayloads(recordID int64) ([]models.StoredPayload, error) {
	rows, err := r.db.QueryContext(context.Background(), `
		SELECT record_id, version, storage_path, size
		FROM payloads
		WHERE record_id = $1
		ORDER BY version ASC
	`, recordID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var payloads []models.StoredPayload
	for rows.Next() {
		var payload models.StoredPayload
		if err := rows.Scan(&payload.RecordID, &payload.Version, &payload.StoragePath, &payload.Size); err != nil {
			return nil, err
		}
		payloads = append(payloads, payload)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return payloads, nil
}

// UpdatePayloadSize обновляет размер сохранённого payload.
func (r *Repository) UpdatePayloadSize(recordID int64, version int64, size int64) error {
	result, err := r.db.ExecContext(context.Background(), `
		UPDATE payloads
		SET size = $3
		WHERE record_id = $1 AND version = $2
	`, recordID, version, size)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return errors.New("payload not found")
	}
	return nil
}

// DeleteRecord выполняет soft delete записи.
func (r *Repository) DeleteRecord(id int64) error {
	result, err := r.db.ExecContext(context.Background(), `
		UPDATE records
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`, id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected > 0 {
		return nil
	}

	row := r.db.QueryRowContext(context.Background(), `SELECT deleted_at FROM records WHERE id = $1`, id)
	var deletedAt sql.NullTime
	if err := row.Scan(&deletedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.ErrRecordNotFound
		}
		return err
	}
	if deletedAt.Valid {
		return models.ErrAlreadyDeleted
	}
	return nil
}

// GetRevisions возвращает ревизии после указанной.
func (r *Repository) GetRevisions(userID int64, sinceRevision int64) ([]models.RecordRevision, error) {
	rows, err := r.db.QueryContext(context.Background(), `
		SELECT id, record_id, user_id, revision, device_id
		FROM record_revisions
		WHERE user_id = $1 AND revision > $2
		ORDER BY revision ASC
	`, userID, sinceRevision)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var revisions []models.RecordRevision
	for rows.Next() {
		var rev models.RecordRevision
		if err := rows.Scan(&rev.ID, &rev.RecordID, &rev.UserID, &rev.Revision, &rev.DeviceID); err != nil {
			return nil, err
		}
		revisions = append(revisions, rev)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return revisions, nil
}

// CreateRevision добавляет ревизию записи.
func (r *Repository) CreateRevision(rev *models.RecordRevision) error {
	if rev == nil {
		return errors.New("revision is nil")
	}
	_, err := r.db.ExecContext(context.Background(), `
		INSERT INTO record_revisions (record_id, user_id, revision, device_id, created_at)
		VALUES ($1, $2, $3, $4, NOW())
	`, rev.RecordID, rev.UserID, rev.Revision, rev.DeviceID)
	if err != nil {
		if isUniqueViolation(err) {
			return models.ErrRevisionConflict
		}
		return err
	}
	return nil
}

// GetMaxRevision возвращает максимальную ревизию пользователя (0 если ревизий нет).
func (r *Repository) GetMaxRevision(userID int64) (int64, error) {
	var rev int64
	row := r.db.QueryRowContext(context.Background(), `
		SELECT COALESCE(MAX(revision), 0) FROM record_revisions WHERE user_id = $1
	`, userID)
	if err := row.Scan(&rev); err != nil {
		return 0, err
	}
	return rev, nil
}

// GetConflicts возвращает нерешённые конфликты пользователя.
func (r *Repository) GetConflicts(userID int64) ([]models.SyncConflict, error) {
	rows, err := r.db.QueryContext(context.Background(), `
		SELECT id, user_id, record_id, local_revision, server_revision, resolved, resolution, local_record, server_record
		FROM sync_conflicts
		WHERE user_id = $1 AND resolved = FALSE
		ORDER BY id ASC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conflicts []models.SyncConflict
	for rows.Next() {
		var conflict models.SyncConflict
		var resolution sql.NullString
		var localJSON, serverJSON []byte
		if err := rows.Scan(
			&conflict.ID,
			&conflict.UserID,
			&conflict.RecordID,
			&conflict.LocalRevision,
			&conflict.ServerRevision,
			&conflict.Resolved,
			&resolution,
			&localJSON,
			&serverJSON,
		); err != nil {
			return nil, err
		}
		if resolution.Valid {
			conflict.Resolution = resolution.String
		}
		if len(localJSON) > 0 {
			conflict.LocalRecord, err = unmarshalConflictRecord(localJSON)
			if err != nil {
				return nil, fmt.Errorf("unmarshal local record for conflict %d: %w", conflict.ID, err)
			}
		}
		if len(serverJSON) > 0 {
			conflict.ServerRecord, err = unmarshalConflictRecord(serverJSON)
			if err != nil {
				return nil, fmt.Errorf("unmarshal server record for conflict %d: %w", conflict.ID, err)
			}
		}
		conflicts = append(conflicts, conflict)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return conflicts, nil
}

// CreateConflict сохраняет конфликт синхронизации.
func (r *Repository) CreateConflict(conflict *models.SyncConflict) error {
	if conflict == nil {
		return errors.New("conflict is nil")
	}
	localJSON, err := marshalConflictRecord(conflict.LocalRecord)
	if err != nil {
		return fmt.Errorf("marshal local record: %w", err)
	}
	serverJSON, err := marshalConflictRecord(conflict.ServerRecord)
	if err != nil {
		return fmt.Errorf("marshal server record: %w", err)
	}
	_, err = r.db.ExecContext(context.Background(), `
		INSERT INTO sync_conflicts (
			user_id, record_id, local_revision, server_revision, resolved, resolution, local_record, server_record, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
	`,
		conflict.UserID,
		conflict.RecordID,
		conflict.LocalRevision,
		conflict.ServerRevision,
		conflict.Resolved,
		nullIfEmpty(conflict.Resolution),
		localJSON,
		serverJSON,
	)
	return err
}

// ResolveConflict помечает конфликт разрешённым.
func (r *Repository) ResolveConflict(conflictID int64, resolution string) error {
	result, err := r.db.ExecContext(context.Background(), `
		UPDATE sync_conflicts
		SET resolved = TRUE, resolution = $2, updated_at = NOW()
		WHERE id = $1
	`, conflictID, resolution)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return models.ErrConflictAlreadyResolved
	}
	return nil
}

// CreateSession сохраняет пользовательскую сессию.
func (r *Repository) CreateSession(session *models.Session) error {
	if session == nil {
		return errors.New("session is nil")
	}
	row := r.db.QueryRowContext(context.Background(), `
		INSERT INTO sessions (
			user_id, device_id, device_name, client_type, refresh_token,
			last_seen_at, expires_at, revoked_at, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		RETURNING id, created_at
	`,
		session.UserID,
		session.DeviceID,
		session.DeviceName,
		session.ClientType,
		session.RefreshToken,
		session.LastSeenAt,
		session.ExpiresAt,
		session.RevokedAt,
	)
	return row.Scan(&session.ID, &session.CreatedAt)
}

// GetSession возвращает сессию по ID.
func (r *Repository) GetSession(id int64) (*models.Session, error) {
	row := r.db.QueryRowContext(context.Background(), `
		SELECT id, user_id, device_id, device_name, client_type, refresh_token,
		       last_seen_at, expires_at, revoked_at, created_at
		FROM sessions
		WHERE id = $1
	`, id)
	return scanSession(row)
}

// GetSessionByRefreshToken возвращает сессию по refresh-токену.
func (r *Repository) GetSessionByRefreshToken(token string) (*models.Session, error) {
	row := r.db.QueryRowContext(context.Background(), `
		SELECT id, user_id, device_id, device_name, client_type, refresh_token,
		       last_seen_at, expires_at, revoked_at, created_at
		FROM sessions
		WHERE refresh_token = $1
	`, token)
	return scanSession(row)
}

// RevokeSession отзывает конкретную сессию.
func (r *Repository) RevokeSession(id int64) error {
	_, err := r.db.ExecContext(context.Background(), `
		UPDATE sessions SET revoked_at = NOW() WHERE id = $1 AND revoked_at IS NULL
	`, id)
	return err
}

// RevokeSessionsByUser отзывает все активные сессии пользователя.
func (r *Repository) RevokeSessionsByUser(userID int64) error {
	_, err := r.db.ExecContext(context.Background(), `
		UPDATE sessions SET revoked_at = NOW()
		WHERE user_id = $1 AND revoked_at IS NULL
	`, userID)
	return err
}

// UpdateLastSeenAt обновляет время последней активности.
func (r *Repository) UpdateLastSeenAt(id int64) error {
	_, err := r.db.ExecContext(context.Background(), `
		UPDATE sessions SET last_seen_at = NOW() WHERE id = $1
	`, id)
	return err
}

// CreateUploadSession создаёт новую upload-сессию.
func (r *Repository) CreateUploadSession(session *models.UploadSession) error {
	if session == nil {
		return errors.New("upload session is nil")
	}
	if session.Status == "" {
		session.Status = models.UploadStatusPending
	}
	row := r.db.QueryRowContext(context.Background(), `
		INSERT INTO upload_sessions (
			record_id, user_id, status, total_chunks, received_chunks,
			chunk_size, total_size, key_version, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
		RETURNING id
	`,
		session.RecordID,
		session.UserID,
		session.Status,
		session.TotalChunks,
		session.ReceivedChunks,
		session.ChunkSize,
		session.TotalSize,
		session.KeyVersion,
	)
	return row.Scan(&session.ID)
}

// GetUploadSession возвращает upload-сессию по ID.
func (r *Repository) GetUploadSession(id int64) (*models.UploadSession, error) {
	row := r.db.QueryRowContext(context.Background(), `
		SELECT id, record_id, user_id, status, total_chunks, received_chunks, chunk_size, total_size, key_version
		FROM upload_sessions
		WHERE id = $1
	`, id)
	var session models.UploadSession
	if err := row.Scan(
		&session.ID,
		&session.RecordID,
		&session.UserID,
		&session.Status,
		&session.TotalChunks,
		&session.ReceivedChunks,
		&session.ChunkSize,
		&session.TotalSize,
		&session.KeyVersion,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, models.ErrUploadNotFound
		}
		return nil, err
	}

	chunksRows, err := r.db.QueryContext(context.Background(), `
		SELECT chunk_index FROM upload_chunks WHERE upload_id = $1
	`, id)
	if err != nil {
		return nil, err
	}
	defer chunksRows.Close()

	session.ReceivedChunkSet = make(map[int64]bool)
	for chunksRows.Next() {
		var idx int64
		if err := chunksRows.Scan(&idx); err != nil {
			return nil, err
		}
		session.ReceivedChunkSet[idx] = true
	}
	if err := chunksRows.Err(); err != nil {
		return nil, err
	}

	return &session, nil
}

// GetCompletedUploadByRecordID возвращает завершённую upload-сессию по recordID.
func (r *Repository) GetCompletedUploadByRecordID(recordID int64) (*models.UploadSession, error) {
	row := r.db.QueryRowContext(context.Background(), `
		SELECT id, record_id, user_id, status, total_chunks, received_chunks, chunk_size, total_size, key_version
		FROM upload_sessions
		WHERE record_id = $1 AND status = 'completed'
		ORDER BY updated_at DESC
		LIMIT 1
	`, recordID)
	var session models.UploadSession
	if err := row.Scan(
		&session.ID,
		&session.RecordID,
		&session.UserID,
		&session.Status,
		&session.TotalChunks,
		&session.ReceivedChunks,
		&session.ChunkSize,
		&session.TotalSize,
		&session.KeyVersion,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, models.ErrUploadNotFound
		}
		return nil, err
	}

	chunksRows, err := r.db.QueryContext(context.Background(), `
		SELECT chunk_index FROM upload_chunks WHERE upload_id = $1
	`, session.ID)
	if err != nil {
		return nil, err
	}
	defer chunksRows.Close()

	session.ReceivedChunkSet = make(map[int64]bool)
	for chunksRows.Next() {
		var idx int64
		if err := chunksRows.Scan(&idx); err != nil {
			return nil, err
		}
		session.ReceivedChunkSet[idx] = true
	}
	if err := chunksRows.Err(); err != nil {
		return nil, err
	}

	return &session, nil
}

// UpdateUploadSession обновляет состояние upload-сессии.
func (r *Repository) UpdateUploadSession(session *models.UploadSession) error {
	if session == nil {
		return errors.New("upload session is nil")
	}
	result, err := r.db.ExecContext(context.Background(), `
		UPDATE upload_sessions
		SET status = $2,
		    total_chunks = $3,
		    received_chunks = $4,
		    chunk_size = $5,
		    total_size = $6,
		    key_version = $7,
		    updated_at = NOW()
		WHERE id = $1
	`,
		session.ID,
		session.Status,
		session.TotalChunks,
		session.ReceivedChunks,
		session.ChunkSize,
		session.TotalSize,
		session.KeyVersion,
	)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return models.ErrUploadNotFound
	}
	return nil
}

// SaveChunk сохраняет чанк и обновляет состояние upload-сессии атомарно.
func (r *Repository) SaveChunk(chunk *models.Chunk) error {
	if chunk == nil {
		return errors.New("chunk is nil")
	}
	crypto, err := r.ensureCrypto()
	if err != nil {
		return err
	}
	tx, err := r.db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var status models.UploadStatus
	var totalChunks int64
	var receivedChunks int64
	var keyVersion int64
	row := tx.QueryRowContext(context.Background(), `
		SELECT status, total_chunks, received_chunks, key_version
		FROM upload_sessions
		WHERE id = $1
		FOR UPDATE
	`, chunk.UploadID)
	if scanErr := row.Scan(&status, &totalChunks, &receivedChunks, &keyVersion); scanErr != nil {
		if errors.Is(scanErr, sql.ErrNoRows) {
			return models.ErrUploadNotFound
		}
		return scanErr
	}
	if status == models.UploadStatusCompleted {
		return models.ErrUploadCompleted
	}
	if status == models.UploadStatusAborted {
		return models.ErrUploadAborted
	}
	if status != models.UploadStatusPending {
		return models.ErrUploadNotPending
	}
	if chunk.ChunkIndex < 0 || chunk.ChunkIndex >= totalChunks {
		return models.ErrChunkOutOfRange
	}

	var exists bool
	if err := tx.QueryRowContext(context.Background(), `
		SELECT EXISTS (
			SELECT 1 FROM upload_chunks WHERE upload_id = $1 AND chunk_index = $2
		)
	`, chunk.UploadID, chunk.ChunkIndex).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return models.ErrDuplicateChunk
	}
	chunkData, err := crypto.Encrypt(chunk.Data, keyVersion)
	if err != nil {
		return err
	}
	storagePath := fmt.Sprintf(chunkPathTemplate, chunk.UploadID, chunk.ChunkIndex)
	if err := r.blob.Save(storagePath, chunkData); err != nil {
		return err
	}

	_, err = tx.ExecContext(context.Background(), `
		INSERT INTO upload_chunks (upload_id, chunk_index, size, storage_path, created_at)
		VALUES ($1, $2, $3, $4, NOW())
	`, chunk.UploadID, chunk.ChunkIndex, int64(len(chunkData)), storagePath)
	if err != nil {
		_ = r.blob.Delete(storagePath)
		return err
	}

	receivedChunks++
	newStatus := status
	if receivedChunks >= totalChunks {
		newStatus = models.UploadStatusCompleted
	}

	_, err = tx.ExecContext(context.Background(), `
		UPDATE upload_sessions
		SET received_chunks = $2, status = $3, updated_at = NOW()
		WHERE id = $1
	`, chunk.UploadID, receivedChunks, newStatus)
	if err != nil {
		_ = r.blob.Delete(storagePath)
		return err
	}

	if err = tx.Commit(); err != nil {
		_ = r.blob.Delete(storagePath)
		return err
	}
	return nil
}

// GetChunks возвращает все чанки для upload-сессии.
func (r *Repository) GetChunks(uploadID int64) ([]models.Chunk, error) {
	crypto, err := r.ensureCrypto()
	if err != nil {
		return nil, err
	}
	var keyVersion int64
	keyRow := r.db.QueryRowContext(context.Background(), `
		SELECT key_version
		FROM upload_sessions
		WHERE id = $1
	`, uploadID)
	if err := keyRow.Scan(&keyVersion); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, models.ErrUploadNotFound
		}
		return nil, err
	}
	rows, err := r.db.QueryContext(context.Background(), `
		SELECT chunk_index, storage_path
		FROM upload_chunks
		WHERE upload_id = $1
		ORDER BY chunk_index ASC
	`, uploadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chunks []models.Chunk
	for rows.Next() {
		var idx int64
		var storagePath string
		if err := rows.Scan(&idx, &storagePath); err != nil {
			return nil, err
		}
		data, err := r.blob.Read(storagePath)
		if err != nil {
			return nil, err
		}
		data, err = decryptMaybeLegacy(crypto, data, keyVersion)
		if err != nil {
			return nil, err
		}
		chunks = append(chunks, models.Chunk{
			UploadID:   uploadID,
			ChunkIndex: idx,
			Data:       data,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(chunks) == 0 {
		row := r.db.QueryRowContext(context.Background(), `SELECT 1 FROM upload_sessions WHERE id = $1`, uploadID)
		var exists int
		if err := row.Scan(&exists); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, models.ErrUploadNotFound
			}
			return nil, err
		}
	}
	return chunks, nil
}

// CreateKeyVersion сохраняет новую версию ключа.
func (r *Repository) CreateKeyVersion(kv *models.KeyVersion) error {
	if kv == nil {
		return errors.New("key version is nil")
	}
	row := r.db.QueryRowContext(context.Background(), `
		INSERT INTO key_versions (version, status, encrypted_key, key_nonce, created_at, deprecated_at, retired_at)
		VALUES ($1, $2, $3, $4, NOW(), $5, $6)
		RETURNING id, created_at
	`, kv.Version, kv.Status, kv.EncryptedKey, kv.KeyNonce, kv.DeprecatedAt, kv.RetiredAt)
	return row.Scan(&kv.ID, &kv.CreatedAt)
}

// GetKeyVersion возвращает конкретную версию ключа.
func (r *Repository) GetKeyVersion(version int64) (*models.KeyVersion, error) {
	row := r.db.QueryRowContext(context.Background(), `
		SELECT id, version, status, encrypted_key, key_nonce, created_at, deprecated_at, retired_at
		FROM key_versions
		WHERE version = $1
	`, version)
	return scanKeyVersion(row)
}

// GetActiveKeyVersion возвращает активную версию ключа.
func (r *Repository) GetActiveKeyVersion() (*models.KeyVersion, error) {
	row := r.db.QueryRowContext(context.Background(), `
		SELECT id, version, status, encrypted_key, key_nonce, created_at, deprecated_at, retired_at
		FROM key_versions
		WHERE status = 'active'
		ORDER BY version DESC
		LIMIT 1
	`)
	return scanKeyVersion(row)
}

// ListKeyVersions возвращает список версий ключей.
func (r *Repository) ListKeyVersions() ([]models.KeyVersion, error) {
	rows, err := r.db.QueryContext(context.Background(), `
		SELECT id, version, status, encrypted_key, key_nonce, created_at, deprecated_at, retired_at
		FROM key_versions
		ORDER BY version ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []models.KeyVersion
	for rows.Next() {
		kv, err := scanKeyVersion(rows)
		if err != nil {
			return nil, err
		}
		versions = append(versions, *kv)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return versions, nil
}

// UpdateKeyVersion обновляет статус версии ключа.
func (r *Repository) UpdateKeyVersion(kv *models.KeyVersion) error {
	if kv == nil {
		return errors.New("key version is nil")
	}
	result, err := r.db.ExecContext(context.Background(), `
		UPDATE key_versions
		SET status = $2, deprecated_at = $3, retired_at = $4
		WHERE version = $1
	`, kv.Version, kv.Status, kv.DeprecatedAt, kv.RetiredAt)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return models.ErrUnknownKeyVersion
	}
	return nil
}

func scanUser(row interface{ Scan(dest ...any) error }) (*models.User, error) {
	var user models.User
	if err := row.Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, models.ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *Repository) scanRecord(row interface{ Scan(dest ...any) error }) (*models.Record, error) {
	var record models.Record
	var recordType string
	var payloadRaw []byte
	var deletedAt sql.NullTime
	if err := row.Scan(
		&record.ID,
		&record.UserID,
		&recordType,
		&record.Name,
		&record.Metadata,
		&payloadRaw,
		&record.Revision,
		&deletedAt,
		&record.DeviceID,
		&record.KeyVersion,
		&record.PayloadVersion,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, models.ErrRecordNotFound
		}
		return nil, err
	}
	crypto, err := r.ensureCrypto()
	if err != nil {
		return nil, err
	}
	record.Type = models.RecordType(recordType)
	if deletedAt.Valid {
		record.DeletedAt = &deletedAt.Time
	}
	decodedPayload, err := decodeStoredPayload(crypto, payloadRaw, record.KeyVersion)
	if err != nil {
		return nil, err
	}
	payload, err := decodePayload(record.Type, decodedPayload)
	if err != nil {
		return nil, err
	}
	record.Payload = payload
	return &record, nil
}

func scanSession(row interface{ Scan(dest ...any) error }) (*models.Session, error) {
	var session models.Session
	var revokedAt sql.NullTime
	if err := row.Scan(
		&session.ID,
		&session.UserID,
		&session.DeviceID,
		&session.DeviceName,
		&session.ClientType,
		&session.RefreshToken,
		&session.LastSeenAt,
		&session.ExpiresAt,
		&revokedAt,
		&session.CreatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, models.ErrUnauthorized
		}
		return nil, err
	}
	if revokedAt.Valid {
		session.RevokedAt = &revokedAt.Time
	}
	return &session, nil
}

func scanKeyVersion(row interface{ Scan(dest ...any) error }) (*models.KeyVersion, error) {
	var kv models.KeyVersion
	var deprecatedAt sql.NullTime
	var retiredAt sql.NullTime
	if err := row.Scan(
		&kv.ID,
		&kv.Version,
		&kv.Status,
		&kv.EncryptedKey,
		&kv.KeyNonce,
		&kv.CreatedAt,
		&deprecatedAt,
		&retiredAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, models.ErrUnknownKeyVersion
		}
		return nil, err
	}
	if deprecatedAt.Valid {
		kv.DeprecatedAt = &deprecatedAt.Time
	}
	if retiredAt.Valid {
		kv.RetiredAt = &retiredAt.Time
	}
	return &kv, nil
}

func splitRecordPayload(record *models.Record) ([]byte, []byte, error) {
	if record.Payload == nil {
		return nil, nil, nil
	}
	switch record.Payload.(type) {
	case models.BinaryPayload, *models.BinaryPayload:
		// Binary payload lifecycle is managed by uploads layer (task_13).
		// Records CRUD stores only payload_version as a reference to the attachment.
		return nil, nil, nil
	default:
		data, err := json.Marshal(record.Payload)
		if err != nil {
			return nil, nil, err
		}
		return data, nil, nil
	}
}

func encryptJSONPayload(crypto cryptosvc.CryptoService, payload []byte, keyVersion int64) ([]byte, error) {
	if len(payload) == 0 {
		return payload, nil
	}
	enc, err := crypto.Encrypt(payload, keyVersion)
	if err != nil {
		return nil, err
	}
	encoded := base64.StdEncoding.EncodeToString(enc)
	return json.Marshal(encoded)
}

func decodeStoredPayload(crypto cryptosvc.CryptoService, raw []byte, keyVersion int64) ([]byte, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var encoded string
	if err := json.Unmarshal(raw, &encoded); err != nil {
		return raw, nil
	}
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}
	return crypto.Decrypt(decoded, keyVersion)
}

func decryptMaybeLegacy(crypto cryptosvc.CryptoService, data []byte, keyVersion int64) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}
	if !cryptosvc.HasEncryptedPrefix(data) {
		return data, nil
	}
	return crypto.Decrypt(data, keyVersion)
}

func decodePayload(recordType models.RecordType, raw []byte) (models.RecordPayload, error) {
	if len(raw) == 0 {
		if recordType == models.RecordTypeBinary {
			return models.BinaryPayload{}, nil
		}
		return nil, nil
	}
	switch recordType {
	case models.RecordTypeLogin:
		var payload models.LoginPayload
		if err := json.Unmarshal(raw, &payload); err != nil {
			return nil, err
		}
		return payload, nil
	case models.RecordTypeText:
		var payload models.TextPayload
		if err := json.Unmarshal(raw, &payload); err != nil {
			return nil, err
		}
		return payload, nil
	case models.RecordTypeBinary:
		var payload models.BinaryPayload
		if err := json.Unmarshal(raw, &payload); err != nil {
			return nil, err
		}
		return payload, nil
	case models.RecordTypeCard:
		var payload models.CardPayload
		if err := json.Unmarshal(raw, &payload); err != nil {
			return nil, err
		}
		return payload, nil
	default:
		return nil, models.ErrInvalidRecordType
	}
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}

func nullIfEmpty(value string) sql.NullString {
	if value == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value, Valid: true}
}

// conflictRecordJSON — вспомогательная структура для JSON-сериализации Record в контексте конфликта.
// Payload сериализуется как конкретный тип, а не как интерфейс.
type conflictRecordJSON struct {
	ID             int64              `json:"id"`
	UserID         int64              `json:"user_id"`
	Type           string             `json:"type"`
	Name           string             `json:"name"`
	Metadata       string             `json:"metadata"`
	Payload        json.RawMessage    `json:"payload"`
	Revision       int64              `json:"revision"`
	DeletedAt      *time.Time         `json:"deleted_at,omitempty"`
	DeviceID       string             `json:"device_id"`
	KeyVersion     int64              `json:"key_version"`
	PayloadVersion int64              `json:"payload_version"`
	CreatedAt      time.Time          `json:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at"`
}

func marshalConflictRecord(record *models.Record) ([]byte, error) {
	if record == nil {
		return nil, nil
	}
	payloadJSON, err := json.Marshal(record.Payload)
	if err != nil {
		return nil, err
	}
	return json.Marshal(conflictRecordJSON{
		ID:             record.ID,
		UserID:         record.UserID,
		Type:           string(record.Type),
		Name:           record.Name,
		Metadata:       record.Metadata,
		Payload:        payloadJSON,
		Revision:       record.Revision,
		DeletedAt:      record.DeletedAt,
		DeviceID:       record.DeviceID,
		KeyVersion:     record.KeyVersion,
		PayloadVersion: record.PayloadVersion,
		CreatedAt:      record.CreatedAt,
		UpdatedAt:      record.UpdatedAt,
	})
}

func unmarshalConflictRecord(data []byte) (*models.Record, error) {
	var cr conflictRecordJSON
	if err := json.Unmarshal(data, &cr); err != nil {
		return nil, err
	}
	record := &models.Record{
		ID:             cr.ID,
		UserID:         cr.UserID,
		Type:           models.RecordType(cr.Type),
		Name:           cr.Name,
		Metadata:       cr.Metadata,
		Revision:       cr.Revision,
		DeletedAt:      cr.DeletedAt,
		DeviceID:       cr.DeviceID,
		KeyVersion:     cr.KeyVersion,
		PayloadVersion: cr.PayloadVersion,
		CreatedAt:      cr.CreatedAt,
		UpdatedAt:      cr.UpdatedAt,
	}
	payload, err := decodePayload(models.RecordType(cr.Type), cr.Payload)
	if err != nil {
		return nil, err
	}
	record.Payload = payload
	return record, nil
}
