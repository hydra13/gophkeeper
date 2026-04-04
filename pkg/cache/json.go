package cache

import (
	"encoding/json"
	"fmt"

	"github.com/hydra13/gophkeeper/internal/models"
)

type jsonRecord struct {
	ID             int64             `json:"id"`
	UserID         int64             `json:"user_id"`
	Type           models.RecordType `json:"type"`
	Name           string            `json:"name"`
	Metadata       string            `json:"metadata"`
	Payload        json.RawMessage   `json:"payload"`
	Revision       int64             `json:"revision"`
	DeletedAt      *int64            `json:"deleted_at,omitempty"`
	DeviceID       string            `json:"device_id"`
	KeyVersion     int64             `json:"key_version"`
	PayloadVersion int64             `json:"payload_version"`
	CreatedAt      int64             `json:"created_at"`
	UpdatedAt      int64             `json:"updated_at"`
}

func recordToJSON(r models.Record) (jsonRecord, error) {
	var payloadRaw json.RawMessage
	var err error
	if r.Payload != nil {
		payloadRaw, err = json.Marshal(r.Payload)
		if err != nil {
			return jsonRecord{}, fmt.Errorf("marshal payload: %w", err)
		}
	}

	var deletedAt *int64
	if r.DeletedAt != nil {
		unix := r.DeletedAt.Unix()
		deletedAt = &unix
	}

	return jsonRecord{
		ID:             r.ID,
		UserID:         r.UserID,
		Type:           r.Type,
		Name:           r.Name,
		Metadata:       r.Metadata,
		Payload:        payloadRaw,
		Revision:       r.Revision,
		DeletedAt:      deletedAt,
		DeviceID:       r.DeviceID,
		KeyVersion:     r.KeyVersion,
		PayloadVersion: r.PayloadVersion,
		CreatedAt:      r.CreatedAt.Unix(),
		UpdatedAt:      r.UpdatedAt.Unix(),
	}, nil
}

func jsonToRecord(jr jsonRecord) (models.Record, error) {
	r := models.Record{
		ID:             jr.ID,
		UserID:         jr.UserID,
		Type:           jr.Type,
		Name:           jr.Name,
		Metadata:       jr.Metadata,
		Revision:       jr.Revision,
		DeviceID:       jr.DeviceID,
		KeyVersion:     jr.KeyVersion,
		PayloadVersion: jr.PayloadVersion,
	}

	if jr.DeletedAt != nil {
		r.DeletedAt = newTimeFromUnix(*jr.DeletedAt)
	}
	if jr.CreatedAt != 0 {
		r.CreatedAt = timeFromUnix(jr.CreatedAt)
	}
	if jr.UpdatedAt != 0 {
		r.UpdatedAt = timeFromUnix(jr.UpdatedAt)
	}

	if len(jr.Payload) > 0 {
		switch jr.Type {
		case models.RecordTypeLogin:
			var p models.LoginPayload
			if err := json.Unmarshal(jr.Payload, &p); err != nil {
				return r, fmt.Errorf("unmarshal login payload: %w", err)
			}
			r.Payload = p
		case models.RecordTypeText:
			var p models.TextPayload
			if err := json.Unmarshal(jr.Payload, &p); err != nil {
				return r, fmt.Errorf("unmarshal text payload: %w", err)
			}
			r.Payload = p
		case models.RecordTypeBinary:
			var p models.BinaryPayload
			if err := json.Unmarshal(jr.Payload, &p); err != nil {
				return r, fmt.Errorf("unmarshal binary payload: %w", err)
			}
			r.Payload = p
		case models.RecordTypeCard:
			var p models.CardPayload
			if err := json.Unmarshal(jr.Payload, &p); err != nil {
				return r, fmt.Errorf("unmarshal card payload: %w", err)
			}
			r.Payload = p
		}
	}

	return r, nil
}

type jsonPendingOp struct {
	ID           int64           `json:"id"`
	RecordID     int64           `json:"record_id"`
	Operation    OperationType   `json:"operation"`
	Record       json.RawMessage `json:"record"`
	BaseRevision int64           `json:"base_revision"`
	CreatedAt    int64           `json:"created_at"`
}

func pendingOpToJSON(op PendingOp) (jsonPendingOp, error) {
	var recRaw json.RawMessage
	if op.Record != nil {
		jr, err := recordToJSON(*op.Record)
		if err != nil {
			return jsonPendingOp{}, err
		}
		recRaw, err = json.Marshal(jr)
		if err != nil {
			return jsonPendingOp{}, err
		}
	}
	return jsonPendingOp{
		ID:           op.ID,
		RecordID:     op.RecordID,
		Operation:    op.Operation,
		Record:       recRaw,
		BaseRevision: op.BaseRevision,
		CreatedAt:    op.CreatedAt,
	}, nil
}

func jsonToPendingOp(jop jsonPendingOp) (PendingOp, error) {
	op := PendingOp{
		ID:           jop.ID,
		RecordID:     jop.RecordID,
		Operation:    jop.Operation,
		BaseRevision: jop.BaseRevision,
		CreatedAt:    jop.CreatedAt,
	}
	if len(jop.Record) > 0 {
		var jr jsonRecord
		if err := json.Unmarshal(jop.Record, &jr); err != nil {
			return op, err
		}
		rec, err := jsonToRecord(jr)
		if err != nil {
			return op, err
		}
		op.Record = &rec
	}
	return op, nil
}
