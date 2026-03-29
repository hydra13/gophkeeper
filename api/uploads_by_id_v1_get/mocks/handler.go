package mocks

import "github.com/hydra13/gophkeeper/api/uploads_by_id_v1_get"

// UploadStatusGetterMock — мок UploadStatusGetter для тестов.
type UploadStatusGetterMock struct {
	GetUploadStatusFunc func(uploadID int64) (*uploads_by_id_v1_get.UploadStatusResponse, error)
}

// GetUploadStatus вызывает мок-реализацию GetUploadStatusFunc.
func (m *UploadStatusGetterMock) GetUploadStatus(uploadID int64) (*uploads_by_id_v1_get.UploadStatusResponse, error) {
	return m.GetUploadStatusFunc(uploadID)
}
