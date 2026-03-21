package sync

// SyncService интерфейс синхронизации
type SyncService interface {
	Sync(userID int64) error
}
