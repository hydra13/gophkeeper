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

// ClientCore — общий клиентский слой для CLI и desktop.
// Не зависит от конкретного UI.
// Работает через Transport (apiclient) и Store (cache).
type ClientCore struct {
	transport apiclient.Transport
	store     cache.Store
	deviceID  string
}

var localIDSeq atomic.Int64

// Config — конфигурация ClientCore.
type Config struct {
	DeviceID string
}

// New создаёт ClientCore с заданным transport и cache store.
func New(transport apiclient.Transport, store cache.Store, cfg Config) *ClientCore {
	return &ClientCore{
		transport: transport,
		store:     store,
		deviceID:  cfg.DeviceID,
	}
}

// Register регистрирует нового пользователя.
func (c *ClientCore) Register(ctx context.Context, email, password string) error {
	userID, err := c.transport.Register(ctx, email, password)
	if err != nil {
		return fmt.Errorf("register: %w", err)
	}

	c.store.Auth().Set(cache.AuthData{
		UserID:   userID,
		Email:    email,
		DeviceID: c.deviceID,
	})

	return c.store.Flush()
}

// Login аутентифицирует пользователя.
func (c *ClientCore) Login(ctx context.Context, email, password string) error {
	return c.LoginWithClient(ctx, email, password, defaultClientName, defaultClientType)
}

// LoginWithClient аутентифицирует пользователя с явным client descriptor.
func (c *ClientCore) LoginWithClient(ctx context.Context, email, password, clientName, clientType string) error {
	accessToken, refreshToken, err := c.transport.Login(ctx, email, password, c.deviceID, clientName, clientType)
	if err != nil {
		return fmt.Errorf("login: %w", err)
	}

	// Получим userID из токена или из ответа — пока сохраняем что есть
	c.store.Auth().Set(cache.AuthData{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Email:        email,
		DeviceID:     c.deviceID,
	})
	c.transport.SetAccessToken(accessToken)

	return c.store.Flush()
}

// Logout отзывает текущую сессию.
func (c *ClientCore) Logout(ctx context.Context) error {
	if err := c.transport.Logout(ctx); err != nil {
		return fmt.Errorf("logout: %w", err)
	}

	c.store.Auth().Clear()
	c.store.Records().Clear()
	c.store.Pending().Clear()
	c.store.Transfers().Clear()
	c.store.Sync().SetLastRevision(0)

	return c.store.Flush()
}

// ListRecords возвращает список записей (из кеша, с фономной синхронизацией).
func (c *ClientCore) ListRecords(ctx context.Context, recordType models.RecordType) ([]models.Record, error) {
	// Попытка синхронизировать с сервером
	if c.isOnline() {
		if err := c.syncFromServer(ctx); err == nil {
			_ = c.store.Flush()
		}
	}

	// Возвращаем из кеша
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
		c.store.Sync().SetLastRevision(maxRevision)
	}
	_ = c.store.Flush()

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

// GetRecord возвращает запись по ID (из кеша или с сервера).
func (c *ClientCore) GetRecord(ctx context.Context, id int64) (*models.Record, error) {
	// Сначала из кеша
	if rec, ok := c.store.Records().Get(id); ok {
		return rec, nil
	}

	// Если онлайн — запрос к серверу
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

// SaveRecord создаёт или обновляет запись.
// Если офлайн — добавляет в pending-очередь.
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

	// Офлайн — сохраняем в кеш и добавляем в pending
	if record.ID == 0 {
		record.ID = nextLocalID()
	}
	c.store.Records().Put(record)

	opType := cache.OperationCreate
	if record.ID > 0 {
		opType = cache.OperationUpdate
	}

	c.store.Pending().Enqueue(cache.PendingOp{
		RecordID:     record.ID,
		Operation:    opType,
		Record:       record,
		BaseRevision: record.Revision,
		CreatedAt:    time.Now().Unix(),
	})
	_ = c.store.Flush()

	return record, nil
}

// DeleteRecord удаляет запись.
// Если офлайн — добавляет в pending-очередь.
func (c *ClientCore) DeleteRecord(ctx context.Context, id int64) error {
	if c.isOnline() {
		if err := c.transport.DeleteRecord(ctx, id, c.deviceID); err != nil {
			return fmt.Errorf("delete record: %w", err)
		}

		c.store.Records().Delete(id)
		_ = c.store.Flush()
		return nil
	}

	// Офлайн — помечаем в кеше и добавляем pending
	if rec, ok := c.store.Records().Get(id); ok {
		rec.SoftDelete()
		c.store.Records().Put(rec)

		c.store.Pending().Enqueue(cache.PendingOp{
			RecordID:     id,
			Operation:    cache.OperationDelete,
			Record:       rec,
			BaseRevision: rec.Revision,
			CreatedAt:    time.Now().Unix(),
		})
		_ = c.store.Flush()
	}

	return nil
}

// SyncNow выполняет полную синхронизацию: pull + push pending.
func (c *ClientCore) SyncNow(ctx context.Context) error {
	if !c.isOnline() {
		return fmt.Errorf("offline: cannot sync")
	}

	// Сначала отправляем pending-операции
	if err := c.flushPending(ctx); err != nil {
		return fmt.Errorf("flush pending: %w", err)
	}

	if err := c.flushPendingTransfers(ctx); err != nil {
		return fmt.Errorf("flush pending transfers: %w", err)
	}

	// Затем получаем изменения с сервера
	if err := c.syncFromServer(ctx); err != nil {
		return fmt.Errorf("sync from server: %w", err)
	}

	return c.store.Flush()
}

// flushPending отправляет все pending-операции на сервер.
func (c *ClientCore) flushPending(ctx context.Context) error {
	ops, err := c.store.Pending().DequeueAll()
	if err != nil {
		return err
	}
	if len(ops) == 0 {
		return nil
	}

	// Отправляем каждую операцию
	for _, op := range ops {
		switch op.Operation {
		case cache.OperationCreate:
			if op.Record != nil {
				localID := op.Record.ID
				result, err := c.transport.CreateRecord(ctx, op.Record)
				if err != nil {
					// Возвращаем в очередь при ошибке
					c.store.Pending().Enqueue(op)
					return fmt.Errorf("push create record %d: %w", op.RecordID, err)
				}
				c.rebindOfflineRecord(localID, result, ops)
				c.store.Records().Put(result)
			}
		case cache.OperationUpdate:
			if op.Record != nil {
				result, err := c.transport.UpdateRecord(ctx, op.Record)
				if err != nil {
					c.store.Pending().Enqueue(op)
					return fmt.Errorf("push update record %d: %w", op.RecordID, err)
				}
				c.store.Records().Put(result)
			}
		case cache.OperationDelete:
			if err := c.transport.DeleteRecord(ctx, op.RecordID, c.deviceID); err != nil {
				c.store.Pending().Enqueue(op)
				return fmt.Errorf("push delete record %d: %w", op.RecordID, err)
			}
			c.store.Records().Delete(op.RecordID)
		}
	}

	return nil
}

// syncFromServer получает изменения с сервера и обновляет кеш.
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
		c.store.Sync().SetLastRevision(result.NextRevision)
	}

	return nil
}

// isOnline проверяет, авторизован ли пользователь (есть токен).
func (c *ClientCore) isOnline() bool {
	authData, ok := c.store.Auth().Get()
	return ok && authData.AccessToken != ""
}

// RestoreAuth восстанавливает access token из кеша.
// Вызывать при старте приложения.
func (c *ClientCore) RestoreAuth() bool {
	authData, ok := c.store.Auth().Get()
	if !ok || authData.AccessToken == "" {
		return false
	}
	c.transport.SetAccessToken(authData.AccessToken)
	return true
}

// IsAuthenticated проверяет наличие кешированных токенов.
func (c *ClientCore) IsAuthenticated() bool {
	authData, ok := c.store.Auth().Get()
	return ok && authData.AccessToken != ""
}

// CurrentAuth возвращает копию текущего состояния аутентификации из локального кеша.
func (c *ClientCore) CurrentAuth() (cache.AuthData, bool) {
	authData, ok := c.store.Auth().Get()
	if !ok || authData == nil {
		return cache.AuthData{}, false
	}
	return *authData, true
}

// UploadBinary загружает бинарные данные на сервер через chunk upload.
// Поддерживает resume: если upload уже был начат, продолжает с последнего чанка.
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
		c.store.Transfers().Save(cache.Transfer{
			ID:           transferID,
			Type:         cache.TransferUpload,
			RecordID:     recordID,
			TotalChunks:  totalChunks,
			CompletedIdx: -1,
			Status:       cache.TransferStatusPaused,
			ChunkSize:    chunkSize,
			TotalSize:    totalSize,
			Data:         append([]byte(nil), data...),
		})
		_ = c.store.Flush()
		return nil
	}

	// Проверим, есть ли незавершённый upload (active или paused)
	var transfer *cache.Transfer
	if tr, ok := c.store.Transfers().GetByRecord(recordID); ok && (tr.Status == cache.TransferStatusActive || tr.Status == cache.TransferStatusPaused) {
		transfer = &tr
	}

	var uploadID int64
	var startChunk int64
	var err error

	if transfer != nil {
		// Resume
		uploadID = transfer.SessionID
		startChunk = transfer.CompletedIdx + 1

		if uploadID <= 0 {
			uploadID, err = c.transport.CreateUploadSession(ctx, recordID, totalChunks, chunkSize, totalSize, 1)
			if err != nil {
				return fmt.Errorf("create upload session: %w", err)
			}
			startChunk = 0
		} else {
			status, err := c.transport.GetUploadStatus(ctx, uploadID)
			if err != nil {
				// Upload больше не валиден — создаём новый
				uploadID, err = c.transport.CreateUploadSession(ctx, recordID, totalChunks, chunkSize, totalSize, 1)
				if err != nil {
					return fmt.Errorf("create upload session: %w", err)
				}
				startChunk = 0
			} else {
				// Проверим, какие чанки уже загружены
				if len(status.MissingChunks) > 0 {
					startChunk = status.MissingChunks[0]
				} else {
					startChunk = status.ReceivedChunks
				}
			}
		}
	} else {
		// Новый upload
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
		c.store.Transfers().Save(*transfer)
	}

	// Если transfer существовал офлайн/локально — переведём его на server session id.
	if transfer != nil && transfer.SessionID != uploadID {
		c.store.Transfers().Delete(transfer.ID)
		transfer.ID = uploadID
		transfer.SessionID = uploadID
		transfer.Status = cache.TransferStatusActive
		transfer.TotalChunks = totalChunks
		transfer.ChunkSize = chunkSize
		transfer.TotalSize = totalSize
		transfer.Data = append([]byte(nil), data...)
		c.store.Transfers().Save(*transfer)
	}

	// Загружаем чанки
	for i := startChunk; i < totalChunks; i++ {
		start := i * chunkSize
		end := start + chunkSize
		if end > totalSize {
			end = totalSize
		}

		chunk := data[start:end]
		if err := c.transport.UploadChunk(ctx, uploadID, i, chunk); err != nil {
			// Сохраняем прогресс
			c.store.Transfers().Save(cache.Transfer{
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
			})
			_ = c.store.Flush()
			return fmt.Errorf("upload chunk %d: %w", i, err)
		}
	}

	// Upload завершён
	c.store.Transfers().Save(cache.Transfer{
		ID:           uploadID,
		Type:         cache.TransferUpload,
		RecordID:     recordID,
		SessionID:    uploadID,
		TotalChunks:  totalChunks,
		CompletedIdx: totalChunks - 1,
		Status:       cache.TransferStatusCompleted,
		ChunkSize:    chunkSize,
		TotalSize:    totalSize,
	})
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
		c.store.Transfers().Save(transfer)
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

// DownloadBinary скачивает бинарные данные с сервера через chunk download.
// Поддерживает resume: если download уже был начат, продолжает с неподтверждённого чанка.
func (c *ClientCore) DownloadBinary(ctx context.Context, recordID int64, chunkSize int64) ([]byte, error) {
	if !c.isOnline() {
		return nil, fmt.Errorf("offline: cannot download")
	}

	// Проверим, есть ли незавершённый download
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
			// Сохраняем прогресс
			c.store.Transfers().Save(cache.Transfer{
				ID:           downloadID,
				Type:         cache.TransferDownload,
				RecordID:     recordID,
				SessionID:    downloadID,
				TotalChunks:  totalChunks,
				CompletedIdx: i - 1,
				Status:       cache.TransferStatusPaused,
				ChunkSize:    chunkSize,
			})
			_ = c.store.Flush()
			return nil, fmt.Errorf("download chunk %d: %w", i, err)
		}

		data = append(data, chunk...)

		// Подтверждаем получение чанка
		if err := c.transport.ConfirmChunk(ctx, downloadID, i); err != nil {
			c.store.Transfers().Save(cache.Transfer{
				ID:           downloadID,
				Type:         cache.TransferDownload,
				RecordID:     recordID,
				SessionID:    downloadID,
				TotalChunks:  totalChunks,
				CompletedIdx: i,
				Status:       cache.TransferStatusPaused,
				ChunkSize:    chunkSize,
			})
			_ = c.store.Flush()
			return nil, fmt.Errorf("confirm chunk %d: %w", i, err)
		}
	}

	// Download завершён
	c.store.Transfers().Delete(downloadID)
	_ = c.store.Flush()

	return data, nil
}
