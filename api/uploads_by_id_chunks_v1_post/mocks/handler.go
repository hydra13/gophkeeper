package mocks

// ChunkUploaderMock — мок ChunkUploader для тестов.
type ChunkUploaderMock struct {
	UploadChunkFunc func(uploadID, chunkIndex int64, data []byte) (receivedChunks, totalChunks int64, completed bool, missingChunks []int64, err error)
}

// UploadChunk вызывает мок-реализацию UploadChunkFunc.
func (m *ChunkUploaderMock) UploadChunk(uploadID, chunkIndex int64, data []byte) (receivedChunks, totalChunks int64, completed bool, missingChunks []int64, err error) {
	return m.UploadChunkFunc(uploadID, chunkIndex, data)
}
