package backend

type AppInfo struct {
	AppName       string `json:"appName"`
	Version       string `json:"version"`
	ServerAddress string `json:"serverAddress"`
	CacheDir      string `json:"cacheDir"`
}

type SessionState struct {
	Authenticated bool   `json:"authenticated"`
	Email         string `json:"email"`
	DeviceID      string `json:"deviceId"`
	AppName       string `json:"appName"`
	Version       string `json:"version"`
	ServerAddress string `json:"serverAddress"`
	CacheDir      string `json:"cacheDir"`
}

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

type RecordListItem struct {
	ID              int64            `json:"id"`
	Type            string           `json:"type"`
	Name            string           `json:"name"`
	Metadata        string           `json:"metadata"`
	MetadataPreview string           `json:"metadataPreview"`
	Revision        int64            `json:"revision"`
	Deleted         bool             `json:"deleted"`
	PayloadVersion  int64            `json:"payloadVersion"`
	Payload         RecordPayloadDTO `json:"payload"`
}

type RecordDetails struct {
	ID             int64            `json:"id"`
	Type           string           `json:"type"`
	Name           string           `json:"name"`
	Metadata       string           `json:"metadata"`
	Revision       int64            `json:"revision"`
	Deleted        bool             `json:"deleted"`
	DeviceID       string           `json:"deviceId"`
	KeyVersion     int64            `json:"keyVersion"`
	PayloadVersion int64            `json:"payloadVersion"`
	CreatedAt      string           `json:"createdAt"`
	UpdatedAt      string           `json:"updatedAt"`
	Payload        RecordPayloadDTO `json:"payload"`
}

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

type SyncResult struct {
	Message string `json:"message"`
}
