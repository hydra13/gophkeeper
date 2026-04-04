package models

// RecordRevision — точка изменения записи для синхронизации между устройствами.
type RecordRevision struct {
	// ID — уникальный идентификатор ревизии.
	ID int64
	// RecordID — ссылка на изменённую запись.
	RecordID int64
	// UserID — владелец записи.
	UserID int64
	// Revision — монотонно возрастающий номер ревизии записи.
	Revision int64
	// DeviceID — устройство, инициировавшее изменение.
	DeviceID string
	// CreatedAt — время создания ревизии.
	// CreatedAt time.Time
}

// SyncConflict — зафиксированный конфликт синхронизации между клиентами.
// Требует явного разрешения пользователем (автоматический мерж не допускается).
type SyncConflict struct {
	// ID — уникальный идентификатор конфликта.
	ID int64
	// UserID — владелец записи с конфликтом.
	UserID int64
	// RecordID — запись, вызвавшая конфликт.
	RecordID int64
	// LocalRevision — ревизия локальной версии клиента.
	LocalRevision int64
	// ServerRevision — ревизия серверной версии.
	ServerRevision int64
	// Resolved — флаг разрешения конфликта.
	Resolved bool
	// Resolution — выбранное разрешение: "local" (оставить локальную) или "server" (принять серверную).
	Resolution string
	// LocalRecord — полные данные локальной версии записи (может быть nil).
	LocalRecord *Record
	// ServerRecord — полные данные серверной версии записи (может быть nil).
	ServerRecord *Record
}

// Resolve разрешает конфликт с указанной стратегией.
// Возвращает ErrConflictAlreadyResolved если конфликт уже разрешён.
// Возвращает ErrInvalidConflictResolution если стратегия неизвестна.
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

// PendingChange описывает одно локальное изменение для push-синхронизации.
type PendingChange struct {
	// Record — полные данные записи.
	Record *Record
	// Deleted — признак удаления записи.
	Deleted bool
	// BaseRevision — ревизия, на основе которой было сделано изменение.
	BaseRevision int64
}

// Константы разрешения конфликтов.
const (
	ConflictResolutionLocal  = "local"
	ConflictResolutionServer = "server"
)
