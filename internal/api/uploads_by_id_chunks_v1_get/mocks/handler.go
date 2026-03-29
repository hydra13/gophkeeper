package mocks

import "github.com/hydra13/gophkeeper/internal/api/uploads_by_id_chunks_v1_get"

// ChunkDownloaderMock — мок ChunkDownloader для тестов.
type ChunkDownloaderMock struct {
	DownloadChunkFunc func(uploadID, chunkIndex int64) (*uploads_by_id_chunks_v1_get.ChunkDownloadResponse, error)
}

// DownloadChunk вызывает мок-реализацию DownloadChunkFunc.
func (m *ChunkDownloaderMock) DownloadChunk(uploadID, chunkIndex int64) (*uploads_by_id_chunks_v1_get.ChunkDownloadResponse, error) {
	return m.DownloadChunkFunc(uploadID, chunkIndex)
}
