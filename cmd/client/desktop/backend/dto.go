package backend

// AppInfo содержит сведения о desktop-клиенте и его окружении.
type AppInfo struct {
	AppName       string `json:"appName"`
	Version       string `json:"version"`
	ServerAddress string `json:"serverAddress"`
	CacheDir      string `json:"cacheDir"`
}

// SessionState описывает текущее состояние пользовательской сессии.
type SessionState struct {
	Authenticated bool   `json:"authenticated"`
	Email         string `json:"email"`
	DeviceID      string `json:"deviceId"`
	AppName       string `json:"appName"`
	Version       string `json:"version"`
	ServerAddress string `json:"serverAddress"`
	CacheDir      string `json:"cacheDir"`
}

// RecordPayloadDTO передает данные payload в формате, удобном для UI.
type RecordPayloadDTO struct {
	Login      string `json:"login,omitempty"`
	Password   string `json:"password,omitempty"`
	Content    string `json:"content,omitempty"`
	Number     string `json:"number,omitempty"`
	Holder     string `json:"holder,omitempty"`
	Expiry     string `json:"expiry,omitempty"`
	CVV        string `json:"cvv,omitempty"`
	BinarySize int    `json:"binarySize,omitempty"`
}

// RecordListItem описывает запись в списке записей desktop-клиента.
type RecordListItem struct {
	ID              int64            `json:"id"`
	Type            string           `json:"type"`
	Name            string           `json:"name"`
	Metadata        string           `json:"metadata"`
	MetadataPreview string           `json:"metadataPreview"`
	Revision        int64            `json:"revision"`
	PayloadVersion  int64            `json:"payloadVersion"`
	Payload         RecordPayloadDTO `json:"payload"`
}

// RecordDetails содержит полное представление записи для карточки просмотра.
type RecordDetails struct {
	ID             int64            `json:"id"`
	Type           string           `json:"type"`
	Name           string           `json:"name"`
	Metadata       string           `json:"metadata"`
	Revision       int64            `json:"revision"`
	DeviceID       string           `json:"deviceId"`
	KeyVersion     int64            `json:"keyVersion"`
	PayloadVersion int64            `json:"payloadVersion"`
	CreatedAt      string           `json:"createdAt"`
	UpdatedAt      string           `json:"updatedAt"`
	Payload        RecordPayloadDTO `json:"payload"`
}

// RecordUpsertInput описывает данные для создания или обновления записи.
type RecordUpsertInput struct {
	ID       int64  `json:"id"`
	Type     string `json:"type"`
	Name     string `json:"name"`
	Metadata string `json:"metadata"`

	Login    string `json:"login"`
	Password string `json:"password"`
	Content  string `json:"content"`
	Number   string `json:"number"`
	Holder   string `json:"holder"`
	Expiry   string `json:"expiry"`
	CVV      string `json:"cvv"`

	FilePath string `json:"filePath"`
}

// SyncResult содержит результат ручной синхронизации.
type SyncResult struct {
	Message string `json:"message"`
}
