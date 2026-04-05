package models

// RecordRevision описывает зафиксированное изменение записи.
type RecordRevision struct {
	ID       int64
	RecordID int64
	UserID   int64
	Revision int64
	DeviceID string
}

// SyncConflict описывает конфликт синхронизации между версиями записи.
type SyncConflict struct {
	ID             int64
	UserID         int64
	RecordID       int64
	LocalRevision  int64
	ServerRevision int64
	Resolved       bool
	Resolution     string
	LocalRecord    *Record
	ServerRecord   *Record
}

// Resolve помечает конфликт разрешённым с выбранной стратегией.
func (c *SyncConflict) Resolve(resolution string) error {
	if c.Resolved {
		return ErrConflictAlreadyResolved
	}
	if resolution != ConflictResolutionLocal && resolution != ConflictResolutionServer {
		return ErrInvalidConflictResolution
	}
	c.Resolved = true
	c.Resolution = resolution
	return nil
}

// PendingChange описывает локальное изменение для отправки на сервер.
type PendingChange struct {
	Record       *Record
	Deleted      bool
	BaseRevision int64
}

// Поддерживаемые стратегии разрешения конфликта.
const (
	ConflictResolutionLocal  = "local"
	ConflictResolutionServer = "server"
)
