package reencrypt

import (
	"context"
	"time"

	"github.com/hydra13/gophkeeper/internal/models"
	"github.com/hydra13/gophkeeper/internal/repositories"
	cryptosvc "github.com/hydra13/gophkeeper/internal/services/crypto"
	"github.com/hydra13/gophkeeper/internal/services/keys"
)

const (
	defaultBatchSize = 100
	defaultInterval  = 30 * time.Second
)

// Repository описывает минимальные операции для re-encryption.
type Repository interface {
	ListRecordsForReencrypt(activeVersion int64, limit int) ([]models.Record, error)
	UpdateRecord(record *models.Record) error
	ListPayloads(recordID int64) ([]models.StoredPayload, error)
	UpdatePayloadSize(recordID int64, version int64, size int64) error
}

// Job фоновая ротация и перешифрование данных.
type Job struct {
	repo      Repository
	blob      repositories.BlobStorage
	crypto    cryptosvc.CryptoService
	keys      *keys.Manager
	batchSize int
	interval  time.Duration
	stopCh    chan struct{}
	doneCh    chan struct{}
	enabled   bool
}

// Option описывает настройку job.
type Option func(*Job)

// WithDeps задаёт зависимости и включает job.
func WithDeps(repo Repository, blob repositories.BlobStorage, crypto cryptosvc.CryptoService, keyManager *keys.Manager) Option {
	return func(j *Job) {
		j.repo = repo
		j.blob = blob
		j.crypto = crypto
		j.keys = keyManager
		j.enabled = true
	}
}

// WithBatchSize задаёт размер батча.
func WithBatchSize(size int) Option {
	return func(j *Job) {
		j.batchSize = size
	}
}

// WithInterval задаёт интервал запуска.
func WithInterval(interval time.Duration) Option {
	return func(j *Job) {
		j.interval = interval
	}
}

// New создаёт job.
func New(opts ...Option) *Job {
	job := &Job{
		batchSize: defaultBatchSize,
		interval:  defaultInterval,
		stopCh:    make(chan struct{}),
		doneCh:    make(chan struct{}),
	}
	for _, opt := range opts {
		opt(job)
	}
	return job
}

// Start запускает job.
func (j *Job) Start(ctx context.Context) error {
	if !j.enabled {
		return nil
	}
	go func() {
		defer close(j.doneCh)
		ticker := time.NewTicker(j.interval)
		defer ticker.Stop()
		_ = j.runOnce(ctx)
		for {
			select {
			case <-ctx.Done():
				return
			case <-j.stopCh:
				return
			case <-ticker.C:
				_ = j.runOnce(ctx)
			}
		}
	}()
	return nil
}

// Stop останавливает job.
func (j *Job) Stop(ctx context.Context) error {
	if !j.enabled {
		return nil
	}
	close(j.stopCh)
	select {
	case <-j.doneCh:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (j *Job) runOnce(ctx context.Context) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	active, err := j.keys.EnsureActive()
	if err != nil {
		return err
	}
	activeVersion := active.Version

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		records, err := j.repo.ListRecordsForReencrypt(activeVersion, j.batchSize)
		if err != nil {
			return err
		}
		if len(records) == 0 {
			return nil
		}
		for i := range records {
			record := records[i]
			if record.KeyVersion == activeVersion {
				continue
			}
			if record.Type == models.RecordTypeBinary {
				if err := j.reencryptBinary(&record, activeVersion); err != nil {
					return err
				}
			}
			record.KeyVersion = activeVersion
			if err := j.repo.UpdateRecord(&record); err != nil {
				return err
			}
		}
	}
}

func (j *Job) reencryptBinary(record *models.Record, newVersion int64) error {
	oldVersion := record.KeyVersion
	payloads, err := j.repo.ListPayloads(record.ID)
	if err != nil {
		return err
	}
	for _, payload := range payloads {
		data, err := j.blob.Read(payload.StoragePath)
		if err != nil {
			return err
		}
		data, err = decryptMaybeLegacy(j.crypto, data, oldVersion)
		if err != nil {
			return err
		}
		encrypted, err := j.crypto.Encrypt(data, newVersion)
		if err != nil {
			return err
		}
		if err := j.blob.Save(payload.StoragePath, encrypted); err != nil {
			return err
		}
		if err := j.repo.UpdatePayloadSize(payload.RecordID, payload.Version, int64(len(encrypted))); err != nil {
			return err
		}
	}
	return nil
}

func decryptMaybeLegacy(crypto cryptosvc.CryptoService, data []byte, keyVersion int64) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}
	if !cryptosvc.HasEncryptedPrefix(data) {
		return data, nil
	}
	return crypto.Decrypt(data, keyVersion)
}
