package reencrypt

import (
	"context"
	"iter"
	"time"

	"github.com/hydra13/gophkeeper/internal/models"
	"github.com/hydra13/gophkeeper/internal/option"
	"github.com/hydra13/gophkeeper/internal/repositories"
	cryptosvc "github.com/hydra13/gophkeeper/internal/services/crypto"
	"github.com/hydra13/gophkeeper/internal/services/keys"
)

const (
	defaultBatchSize = 100
	defaultInterval  = 30 * time.Second
)

// Repository описывает доступ к данным для переупаковки.
type Repository interface {
	ListRecordsForReencrypt(activeVersion int64, limit int) ([]models.Record, error)
	UpdateRecord(record *models.Record) error
	ListPayloads(recordID int64) ([]models.StoredPayload, error)
	UpdatePayloadSize(recordID int64, version int64, size int64) error
}

type recordsForReencryptSeqRepo interface {
	ListRecordsForReencryptSeq(activeVersion int64, limit int) iter.Seq2[models.Record, error]
}

// Job переупаковывает записи на актуальный ключ.
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

// Option настраивает задачу.
type Option = option.Option[Job]

// WithDeps задаёт зависимости задачи.
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

// New создаёт задачу переупаковки.
func New(opts ...Option) *Job {
	job := &Job{
		batchSize: defaultBatchSize,
		interval:  defaultInterval,
		stopCh:    make(chan struct{}),
		doneCh:    make(chan struct{}),
	}
	option.Apply(job, opts...)
	return job
}

// Start запускает фоновую задачу.
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

// Stop останавливает фоновую задачу.
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

		processed, err := j.processBatch(activeVersion)
		if err != nil {
			return err
		}
		if processed == 0 {
			return nil
		}
	}
}

func (j *Job) processBatch(activeVersion int64) (int, error) {
	if seqRepo, ok := j.repo.(recordsForReencryptSeqRepo); ok {
		return j.processBatchSeq(seqRepo.ListRecordsForReencryptSeq(activeVersion, j.batchSize), activeVersion)
	}

	records, err := j.repo.ListRecordsForReencrypt(activeVersion, j.batchSize)
	if err != nil {
		return 0, err
	}
	for i := range records {
		if err := j.processRecord(records[i], activeVersion); err != nil {
			return 0, err
		}
	}
	return len(records), nil
}

func (j *Job) processBatchSeq(records iter.Seq2[models.Record, error], activeVersion int64) (int, error) {
	processed := 0
	for record, err := range records {
		if err != nil {
			return 0, err
		}
		if err := j.processRecord(record, activeVersion); err != nil {
			return 0, err
		}
		processed++
	}
	return processed, nil
}

func (j *Job) processRecord(record models.Record, activeVersion int64) error {
	if record.KeyVersion == activeVersion {
		return nil
	}
	if record.Type == models.RecordTypeBinary {
		if err := j.reencryptBinary(&record, activeVersion); err != nil {
			return err
		}
	}
	record.KeyVersion = activeVersion
	return j.repo.UpdateRecord(&record)
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
