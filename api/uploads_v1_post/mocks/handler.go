package mocks

// UploadCreatorMock — мок UploadCreator для тестов.
type UploadCreatorMock struct {
	CreateSessionFunc func(userID, recordID, totalChunks, chunkSize, totalSize, keyVersion int64) (int64, error)
}

// CreateSession вызывает мок-реализацию CreateSessionFunc.
func (m *UploadCreatorMock) CreateSession(userID, recordID, totalChunks, chunkSize, totalSize, keyVersion int64) (int64, error) {
	return m.CreateSessionFunc(userID, recordID, totalChunks, chunkSize, totalSize, keyVersion)
}
