package clientcore

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/hydra13/gophkeeper/internal/models"
	"github.com/hydra13/gophkeeper/pkg/apiclient"
	"github.com/hydra13/gophkeeper/pkg/cache"
)

const (
	defaultClientName = "cli-client"
	defaultClientType = "cli"
)

// ClientCore координирует транспорт и локальный кеш клиента.
type ClientCore struct {
	transport apiclient.Transport
	store     cache.Store
	deviceID  string
}

var localIDSeq atomic.Int64

// Config задаёт параметры инициализации ClientCore.
type Config struct {
	DeviceID string
}

// New создаёт клиентское ядро поверх транспорта и локального кеша.
func New(transport apiclient.Transport, store cache.Store, cfg Config) *ClientCore {
	return &ClientCore{
		transport: transport,
		store:     store,
		deviceID:  cfg.DeviceID,
	}
}

func (c *ClientCore) setAuth(data cache.AuthData) error {
	return c.store.Auth().Set(data)
}

func (c *ClientCore) setLastRevision(rev int64) error {
	return c.store.Sync().SetLastRevision(rev)
}

func (c *ClientCore) enqueuePending(op cache.PendingOp) error {
	return c.store.Pending().Enqueue(op)
}

func (c *ClientCore) saveTransfer(transfer cache.Transfer) error {
	return c.store.Transfers().Save(transfer)
}

// Register регистрирует пользователя и сохраняет локальное состояние сессии.
func (c *ClientCore) Register(ctx context.Context, email, password string) error {
	userID, err := c.transport.Register(ctx, email, password)
	if err != nil {
		return fmt.Errorf("register: %w", err)
	}

	if err := c.setAuth(cache.AuthData{
		UserID:   userID,
		Email:    email,
		DeviceID: c.deviceID,
	}); err != nil {
		return fmt.Errorf("cache auth after register: %w", err)
	}

	return c.store.Flush()
}

// Login выполняет вход с параметрами клиента по умолчанию.
func (c *ClientCore) Login(ctx context.Context, email, password string) error {
	return c.LoginWithClient(ctx, email, password, defaultClientName, defaultClientType)
}

// LoginWithClient выполняет вход с явными параметрами клиента.
func (c *ClientCore) LoginWithClient(ctx context.Context, email, password, clientName, clientType string) error {
	accessToken, refreshToken, err := c.transport.Login(ctx, email, password, c.deviceID, clientName, clientType)
	if err != nil {
		return fmt.Errorf("login: %w", err)
	}

	if err := c.setAuth(cache.AuthData{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Email:        email,
		DeviceID:     c.deviceID,
	}); err != nil {
		return fmt.Errorf("cache auth after login: %w", err)
	}
	c.transport.SetAccessToken(accessToken)

	return c.store.Flush()
}

// Logout завершает сессию и очищает локальный кеш.
func (c *ClientCore) Logout(ctx context.Context) error {
	if err := c.transport.Logout(ctx); err != nil {
		return fmt.Errorf("logout: %w", err)
	}

	c.store.Auth().Clear()
	c.store.Records().Clear()
	c.store.Pending().Clear()
	c.store.Transfers().Clear()
	if err := c.setLastRevision(0); err != nil {
		return fmt.Errorf("reset sync revision: %w", err)
	}

	return c.store.Flush()
}

// ListRecords возвращает записи из кеша и при возможности обновляет их с сервера.
func (c *ClientCore) ListRecords(ctx context.Context, recordType models.RecordType) ([]models.Record, error) {
	if c.isOnline() {
		if err := c.syncFromServer(ctx); err == nil {
			_ = c.store.Flush()
		}
	}

	all := c.store.Records().GetAll()
	if c.isOnline() && len(all) == 0 && c.store.Pending().Len() == 0 {
		records, err := c.fetchRecordsSnapshot(ctx, recordType)
		if err != nil {
			return nil, err
		}
		return records, nil
	}

	if recordType == "" {
		return all, nil
	}

	var filtered []models.Record
	for _, r := range all {
		if r.Type == recordType && !r.IsDeleted() {
			filtered = append(filtered, r)
		}
	}
	return filtered, nil
}

func (c *ClientCore) fetchRecordsSnapshot(ctx context.Context, recordType models.RecordType) ([]models.Record, error) {
	includeDeleted := recordType == ""
	records, err := c.transport.ListRecords(ctx, recordType, includeDeleted)
	if err != nil {
		return nil, fmt.Errorf("list records: %w", err)
	}

	var maxRevision int64
	for i := range records {
		rec := records[i]
		if rec.Revision > maxRevision {
			maxRevision = rec.Revision
		}
		c.store.Records().Put(&rec)
	}
	if maxRevision > c.store.Sync().Get().LastRevision {
		if err := c.setLastRevision(maxRevision); err != nil {
			return nil, fmt.Errorf("cache sync revision: %w", err)
		}
	}
	if err := c.store.Flush(); err != nil {
		return nil, fmt.Errorf("flush records snapshot: %w", err)
	}

	if recordType == "" {
		return c.store.Records().GetAll(), nil
	}

	var filtered []models.Record
	for _, rec := range c.store.Records().GetAll() {
		if rec.Type == recordType && !rec.IsDeleted() {
			filtered = append(filtered, rec)
		}
	}
	return filtered, nil
}

// GetRecord возвращает запись из кеша или запрашивает её с сервера.
func (c *ClientCore) GetRecord(ctx context.Context, id int64) (*models.Record, error) {
	if rec, ok := c.store.Records().Get(id); ok {
		return rec, nil
	}

	if c.isOnline() {
		rec, err := c.transport.GetRecord(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("get record: %w", err)
		}
		c.store.Records().Put(rec)
		_ = c.store.Flush()
		return rec, nil
	}

	return nil, models.ErrRecordNotFound
}

// SaveRecord создаёт или обновляет запись и при офлайне ставит изменение в очередь.
func (c *ClientCore) SaveRecord(ctx context.Context, record *models.Record) (*models.Record, error) {
	record.DeviceID = c.deviceID

	if c.isOnline() {
		var result *models.Record
		var err error

		if record.ID == 0 {
			result, err = c.transport.CreateRecord(ctx, record)
		} else {
			result, err = c.transport.UpdateRecord(ctx, record)
		}

		if err != nil {
			return nil, fmt.Errorf("save record: %w", err)
		}

		c.store.Records().Put(result)
		_ = c.store.Flush()
		return result, nil
	}

	if record.ID == 0 {
		record.ID = nextLocalID()
	}
	c.store.Records().Put(record)

	opType := cache.OperationCreate
	if record.ID > 0 {
		opType = cache.OperationUpdate
	}

	if err := c.enqueuePending(cache.PendingOp{
		RecordID:     record.ID,
		Operation:    opType,
		Record:       record,
		BaseRevision: record.Revision,
		CreatedAt:    time.Now().Unix(),
	}); err != nil {
		return nil, fmt.Errorf("enqueue pending record save: %w", err)
	}
	if err := c.store.Flush(); err != nil {
		return nil, fmt.Errorf("flush pending record save: %w", err)
	}

	return record, nil
}

// DeleteRecord удаляет запись сразу или откладывает удаление до синхронизации.
func (c *ClientCore) DeleteRecord(ctx context.Context, id int64) error {
	if c.isOnline() {
		if err := c.transport.DeleteRecord(ctx, id, c.deviceID); err != nil {
			return fmt.Errorf("delete record: %w", err)
		}

		c.store.Records().Delete(id)
		_ = c.store.Flush()
		return nil
	}

	if rec, ok := c.store.Records().Get(id); ok {
		if err := rec.SoftDelete(); err != nil {
			return fmt.Errorf("soft delete cached record: %w", err)
		}
		c.store.Records().Put(rec)

		if err := c.enqueuePending(cache.PendingOp{
			RecordID:     id,
			Operation:    cache.OperationDelete,
			Record:       rec,
			BaseRevision: rec.Revision,
			CreatedAt:    time.Now().Unix(),
		}); err != nil {
			return fmt.Errorf("enqueue pending record delete: %w", err)
		}
		if err := c.store.Flush(); err != nil {
			return fmt.Errorf("flush pending record delete: %w", err)
		}
	}

	return nil
}

func (c *ClientCore) SyncNow(ctx context.Context) error {
	if !c.isOnline() {
		return fmt.Errorf("offline: cannot sync")
	}

	if err := c.flushPending(ctx); err != nil {
		return fmt.Errorf("flush pending: %w", err)
	}

	if err := c.flushPendingTransfers(ctx); err != nil {
		return fmt.Errorf("flush pending transfers: %w", err)
	}

	if err := c.syncFromServer(ctx); err != nil {
		return fmt.Errorf("sync from server: %w", err)
	}

	return c.store.Flush()
}

func (c *ClientCore) flushPending(ctx context.Context) error {
	ops, err := c.store.Pending().DequeueAll()
	if err != nil {
		return err
	}
	if len(ops) == 0 {
		return nil
	}

	for _, op := range ops {
		switch op.Operation {
		case cache.OperationCreate:
			if op.Record != nil {
				localID := op.Record.ID
				result, err := c.transport.CreateRecord(ctx, op.Record)
				if err != nil {
					if enqueueErr := c.enqueuePending(op); enqueueErr != nil {
						return fmt.Errorf("requeue create record %d: %w", op.RecordID, enqueueErr)
					}
					return fmt.Errorf("push create record %d: %w", op.RecordID, err)
				}
				c.rebindOfflineRecord(localID, result, ops)
				c.store.Records().Put(result)
			}
		case cache.OperationUpdate:
			if op.Record != nil {
				result, err := c.transport.UpdateRecord(ctx, op.Record)
				if err != nil {
					if enqueueErr := c.enqueuePending(op); enqueueErr != nil {
						return fmt.Errorf("requeue update record %d: %w", op.RecordID, enqueueErr)
					}
					return fmt.Errorf("push update record %d: %w", op.RecordID, err)
				}
				c.store.Records().Put(result)
			}
		case cache.OperationDelete:
			if err := c.transport.DeleteRecord(ctx, op.RecordID, c.deviceID); err != nil {
				if enqueueErr := c.enqueuePending(op); enqueueErr != nil {
					return fmt.Errorf("requeue delete record %d: %w", op.RecordID, enqueueErr)
				}
				return fmt.Errorf("push delete record %d: %w", op.RecordID, err)
			}
			c.store.Records().Delete(op.RecordID)
		}
	}

	return nil
}

func (c *ClientCore) syncFromServer(ctx context.Context) error {
	lastRev := c.store.Sync().Get().LastRevision

	result, err := c.transport.Pull(ctx, lastRev, c.deviceID, 100)
	if err != nil {
		return err
	}

	for i := range result.Records {
		c.store.Records().Put(&result.Records[i])
	}

	if result.NextRevision > lastRev {
		if err := c.setLastRevision(result.NextRevision); err != nil {
			return fmt.Errorf("cache sync revision: %w", err)
		}
	}

	return nil
}

func (c *ClientCore) isOnline() bool {
	authData, ok := c.store.Auth().Get()
	return ok && authData.AccessToken != ""
}

func (c *ClientCore) RestoreAuth() bool {
	authData, ok := c.store.Auth().Get()
	if !ok || authData.AccessToken == "" {
		return false
	}
	c.transport.SetAccessToken(authData.AccessToken)
	return true
}

func (c *ClientCore) IsAuthenticated() bool {
	authData, ok := c.store.Auth().Get()
	return ok && authData.AccessToken != ""
}

func (c *ClientCore) CurrentAuth() (cache.AuthData, bool) {
	authData, ok := c.store.Auth().Get()
	if !ok || authData == nil {
		return cache.AuthData{}, false
	}
	return *authData, true
}

func (c *ClientCore) UploadBinary(ctx context.Context, recordID int64, data []byte, chunkSize int64) error {
	totalSize := int64(len(data))
	totalChunks := totalSize / chunkSize
	if totalSize%chunkSize != 0 {
		totalChunks++
	}

	if !c.isOnline() {
		transferID := nextLocalID()
		if existing, ok := c.store.Transfers().GetByRecord(recordID); ok && existing.Type == cache.TransferUpload {
			transferID = existing.ID
		}
		if err := c.saveTransfer(cache.Transfer{
			ID:           transferID,
			Type:         cache.TransferUpload,
			RecordID:     recordID,
			TotalChunks:  totalChunks,
			CompletedIdx: -1,
			Status:       cache.TransferStatusPaused,
			ChunkSize:    chunkSize,
			TotalSize:    totalSize,
			Data:         append([]byte(nil), data...),
		}); err != nil {
			return fmt.Errorf("cache offline upload transfer: %w", err)
		}
		_ = c.store.Flush()
		return nil
	}

	var transfer *cache.Transfer
	if tr, ok := c.store.Transfers().GetByRecord(recordID); ok && (tr.Status == cache.TransferStatusActive || tr.Status == cache.TransferStatusPaused) {
		transfer = &tr
	}

	var uploadID int64
	var startChunk int64
	var err error

	if transfer != nil {
		uploadID = transfer.SessionID

		if uploadID <= 0 {
			uploadID, err = c.transport.CreateUploadSession(ctx, recordID, totalChunks, chunkSize, totalSize, 1)
			if err != nil {
				return fmt.Errorf("create upload session: %w", err)
			}
			startChunk = 0
		} else {
			status, err := c.transport.GetUploadStatus(ctx, uploadID)
			if err != nil {
				uploadID, err = c.transport.CreateUploadSession(ctx, recordID, totalChunks, chunkSize, totalSize, 1)
				if err != nil {
					return fmt.Errorf("create upload session: %w", err)
				}
				startChunk = 0
			} else {
				if len(status.MissingChunks) > 0 {
					startChunk = status.MissingChunks[0]
				} else {
					startChunk = status.ReceivedChunks
				}
			}
		}
	} else {
		uploadID, err = c.transport.CreateUploadSession(ctx, recordID, totalChunks, chunkSize, totalSize, 1)
		if err != nil {
			return fmt.Errorf("create upload session: %w", err)
		}

		transfer = &cache.Transfer{
			ID:          uploadID,
			Type:        cache.TransferUpload,
			RecordID:    recordID,
			SessionID:   uploadID,
			TotalChunks: totalChunks,
			Status:      cache.TransferStatusActive,
			ChunkSize:   chunkSize,
			TotalSize:   totalSize,
			Data:        append([]byte(nil), data...),
		}
		if err := c.saveTransfer(*transfer); err != nil {
			return fmt.Errorf("cache new upload transfer: %w", err)
		}
	}

	if transfer != nil && transfer.SessionID != uploadID {
		c.store.Transfers().Delete(transfer.ID)
		transfer.ID = uploadID
		transfer.SessionID = uploadID
		transfer.Status = cache.TransferStatusActive
		transfer.TotalChunks = totalChunks
		transfer.ChunkSize = chunkSize
		transfer.TotalSize = totalSize
		transfer.Data = append([]byte(nil), data...)
		if err := c.saveTransfer(*transfer); err != nil {
			return fmt.Errorf("cache rebound upload transfer: %w", err)
		}
	}

	for i := startChunk; i < totalChunks; i++ {
		start := i * chunkSize
		end := start + chunkSize
		if end > totalSize {
			end = totalSize
		}

		chunk := data[start:end]
		if err := c.transport.UploadChunk(ctx, uploadID, i, chunk); err != nil {
			if saveErr := c.saveTransfer(cache.Transfer{
				ID:           uploadID,
				Type:         cache.TransferUpload,
				RecordID:     recordID,
				SessionID:    uploadID,
				TotalChunks:  totalChunks,
				CompletedIdx: i - 1,
				Status:       cache.TransferStatusPaused,
				ChunkSize:    chunkSize,
				TotalSize:    totalSize,
				Data:         append([]byte(nil), data...),
			}); saveErr != nil {
				return fmt.Errorf("cache paused upload transfer: %w", saveErr)
			}
			_ = c.store.Flush()
			return fmt.Errorf("upload chunk %d: %w", i, err)
		}
	}

	if err := c.saveTransfer(cache.Transfer{
		ID:           uploadID,
		Type:         cache.TransferUpload,
		RecordID:     recordID,
		SessionID:    uploadID,
		TotalChunks:  totalChunks,
		CompletedIdx: totalChunks - 1,
		Status:       cache.TransferStatusCompleted,
		ChunkSize:    chunkSize,
		TotalSize:    totalSize,
	}); err != nil {
		return fmt.Errorf("cache completed upload transfer: %w", err)
	}
	c.store.Transfers().Delete(uploadID)
	_ = c.store.Flush()

	return nil
}

func (c *ClientCore) flushPendingTransfers(ctx context.Context) error {
	for _, transfer := range c.store.Transfers().ListPending() {
		if transfer.Type != cache.TransferUpload || len(transfer.Data) == 0 {
			continue
		}
		if transfer.RecordID <= 0 {
			continue
		}
		if err := c.UploadBinary(ctx, transfer.RecordID, transfer.Data, transfer.ChunkSize); err != nil {
			return fmt.Errorf("upload transfer for record %d: %w", transfer.RecordID, err)
		}
	}
	return nil
}

func (c *ClientCore) rebindOfflineRecord(localID int64, result *models.Record, ops []cache.PendingOp) {
	if localID <= 0 {
		c.store.Records().Delete(localID)
	}
	if transfer, ok := c.store.Transfers().GetByRecord(localID); ok {
		c.store.Transfers().Delete(transfer.ID)
		transfer.RecordID = result.ID
		if err := c.saveTransfer(transfer); err != nil {
			return
		}
	}
	for i := range ops {
		if ops[i].RecordID == localID {
			ops[i].RecordID = result.ID
		}
		if ops[i].Record != nil && ops[i].Record.ID == localID {
			ops[i].Record.ID = result.ID
		}
	}
}

func nextLocalID() int64 {
	return -localIDSeq.Add(1)
}

func (c *ClientCore) DownloadBinary(ctx context.Context, recordID int64, chunkSize int64) ([]byte, error) {
	if !c.isOnline() {
		return nil, fmt.Errorf("offline: cannot download")
	}

	var downloadID int64
	var totalChunks int64
	var startChunk int64

	if tr, ok := c.store.Transfers().GetByRecord(recordID); ok && tr.Type == cache.TransferDownload && (tr.Status == cache.TransferStatusActive || tr.Status == cache.TransferStatusPaused) {
		downloadID = tr.SessionID
		totalChunks = tr.TotalChunks
		startChunk = tr.CompletedIdx + 1
	} else {
		var err error
		downloadID, totalChunks, err = c.transport.CreateDownloadSession(ctx, recordID)
		if err != nil {
			return nil, fmt.Errorf("create download session: %w", err)
		}
		startChunk = 0
	}

	var data []byte

	for i := startChunk; i < totalChunks; i++ {
		chunk, err := c.transport.DownloadChunk(ctx, downloadID, i)
		if err != nil {
			if saveErr := c.saveTransfer(cache.Transfer{
				ID:           downloadID,
				Type:         cache.TransferDownload,
				RecordID:     recordID,
				SessionID:    downloadID,
				TotalChunks:  totalChunks,
				CompletedIdx: i - 1,
				Status:       cache.TransferStatusPaused,
				ChunkSize:    chunkSize,
			}); saveErr != nil {
				return nil, fmt.Errorf("cache paused download transfer: %w", saveErr)
			}
			_ = c.store.Flush()
			return nil, fmt.Errorf("download chunk %d: %w", i, err)
		}

		data = append(data, chunk...)

		if err := c.transport.ConfirmChunk(ctx, downloadID, i); err != nil {
			if saveErr := c.saveTransfer(cache.Transfer{
				ID:           downloadID,
				Type:         cache.TransferDownload,
				RecordID:     recordID,
				SessionID:    downloadID,
				TotalChunks:  totalChunks,
				CompletedIdx: i,
				Status:       cache.TransferStatusPaused,
				ChunkSize:    chunkSize,
			}); saveErr != nil {
				return nil, fmt.Errorf("cache confirmed download transfer: %w", saveErr)
			}
			_ = c.store.Flush()
			return nil, fmt.Errorf("confirm chunk %d: %w", i, err)
		}
	}

	c.store.Transfers().Delete(downloadID)
	_ = c.store.Flush()

	return data, nil
}
