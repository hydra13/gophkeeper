package backend

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hydra13/gophkeeper/pkg/clientcore"
	"github.com/hydra13/gophkeeper/pkg/clientui"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type BinaryService struct {
	core *clientcore.ClientCore

	mu         sync.RWMutex
	runtimeCtx context.Context
}

func NewBinaryService(core *clientcore.ClientCore) *BinaryService {
	return &BinaryService{core: core}
}

func (s *BinaryService) SetRuntimeContext(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.runtimeCtx = ctx
}

func (s *BinaryService) PickFileForUpload() (string, error) {
	ctx, err := s.context()
	if err != nil {
		return "", err
	}

	path, err := runtime.OpenFileDialog(ctx, runtime.OpenDialogOptions{
		Title: "Select file to upload",
	})
	if err != nil {
		return "", normalizeError(err)
	}
	return path, nil
}

func (s *BinaryService) SaveBinaryAs(recordID int64) (string, error) {
	ctx, err := s.context()
	if err != nil {
		return "", err
	}

	path, err := runtime.SaveFileDialog(ctx, runtime.SaveDialogOptions{
		Title: "Save binary file",
	})
	if err != nil {
		return "", normalizeError(err)
	}
	if path == "" {
		return "", nil
	}

	return s.DownloadBinary(recordID, path)
}

func (s *BinaryService) DownloadBinary(recordID int64, savePath string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	data, err := s.core.DownloadBinary(ctx, recordID, binaryChunkSize)
	if err != nil {
		return "", normalizeError(err)
	}

	if err := clientui.WriteBinaryFile(savePath, data); err != nil {
		return "", normalizeError(err)
	}

	return savePath, nil
}

func (s *BinaryService) context() (context.Context, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.runtimeCtx == nil {
		return nil, fmt.Errorf("desktop runtime is not ready")
	}

	return s.runtimeCtx, nil
}
