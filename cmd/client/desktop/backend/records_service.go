package backend

import (
	"context"
	"fmt"
	"time"

	"github.com/hydra13/gophkeeper/internal/models"
	"github.com/hydra13/gophkeeper/pkg/clientcore"
	"github.com/hydra13/gophkeeper/pkg/clientui"
)

const binaryChunkSize int64 = 64 * 1024

type RecordsService struct {
	core *clientcore.ClientCore
}

func NewRecordsService(core *clientcore.ClientCore) *RecordsService {
	return &RecordsService{core: core}
}

func (s *RecordsService) ListRecords(filter string) ([]RecordListItem, error) {
	recordType, err := parseFilter(filter)
	if err != nil {
		return nil, normalizeError(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	records, err := s.core.ListRecords(ctx, recordType)
	if err != nil {
		return nil, normalizeError(err)
	}

	items := make([]RecordListItem, 0, len(records))
	for _, rec := range records {
		items = append(items, toListItem(rec))
	}
	return items, nil
}

func (s *RecordsService) GetRecord(id int64) (*RecordDetails, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	record, err := s.core.GetRecord(ctx, id)
	if err != nil {
		return nil, normalizeError(err)
	}

	return toRecordDetails(record), nil
}

func (s *RecordsService) CreateRecord(input RecordUpsertInput) (*RecordDetails, error) {
	input.ID = 0
	return s.saveRecord(input)
}

func (s *RecordsService) UpdateRecord(input RecordUpsertInput) (*RecordDetails, error) {
	if input.ID <= 0 {
		return nil, fmt.Errorf("record id is required")
	}
	return s.saveRecord(input)
}

func (s *RecordsService) DeleteRecord(id int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return normalizeError(s.core.DeleteRecord(ctx, id))
}

func (s *RecordsService) SyncNow() (SyncResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err := s.core.SyncNow(ctx); err != nil {
		return SyncResult{}, normalizeError(err)
	}

	return SyncResult{Message: "sync completed"}, nil
}

func (s *RecordsService) saveRecord(input RecordUpsertInput) (*RecordDetails, error) {
	recordType, err := clientui.ParseRecordType(input.Type)
	if err != nil {
		return nil, normalizeError(err)
	}

	fields := clientui.PayloadFields{
		Login:    input.Login,
		Password: input.Password,
		Content:  input.Content,
		Number:   input.Number,
		Holder:   input.Holder,
		Expiry:   input.Expiry,
		CVV:      input.CVV,
	}

	payload, err := clientui.BuildPayload(recordType, fields)
	if err != nil {
		return nil, normalizeError(err)
	}

	var fileData []byte
	if recordType == models.RecordTypeBinary && input.FilePath != "" {
		fileData, err = clientui.ReadBinaryFile(input.FilePath)
		if err != nil {
			return nil, normalizeError(err)
		}
	}
	if recordType == models.RecordTypeBinary && input.ID == 0 && len(fileData) == 0 {
		return nil, fmt.Errorf("file path is required for binary records")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	recordToSave, err := s.buildRecordForSave(ctx, input, recordType, payload, len(fileData) > 0)
	if err != nil {
		return nil, normalizeError(err)
	}

	result, err := s.core.SaveRecord(ctx, recordToSave)
	if err != nil {
		return nil, normalizeError(err)
	}

	if recordType == models.RecordTypeBinary && len(fileData) > 0 {
		if err := s.core.UploadBinary(ctx, result.ID, fileData, binaryChunkSize); err != nil {
			return nil, normalizeError(fmt.Errorf("upload binary: %w", err))
		}
	}

	return toRecordDetails(result), nil
}

func (s *RecordsService) buildRecordForSave(ctx context.Context, input RecordUpsertInput, recordType models.RecordType, payload models.RecordPayload, hasBinaryFile bool) (*models.Record, error) {
	if input.ID <= 0 {
		record := &models.Record{
			Type:     recordType,
			Name:     input.Name,
			Metadata: input.Metadata,
			Payload:  payload,
		}
		if recordType == models.RecordTypeBinary {
			record.PayloadVersion = 1
		}
		return record, nil
	}

	existing, err := s.core.GetRecord(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	updated := *existing
	updated.Name = input.Name
	updated.Metadata = input.Metadata

	if recordType != models.RecordTypeBinary {
		updated.Payload = payload
		return &updated, nil
	}

	if hasBinaryFile {
		if updated.PayloadVersion <= 0 {
			updated.PayloadVersion = 1
		} else {
			updated.PayloadVersion++
		}
	}

	return &updated, nil
}

func parseFilter(filter string) (models.RecordType, error) {
	if filter == "" || filter == "all" {
		return "", nil
	}
	return clientui.ParseRecordType(filter)
}
