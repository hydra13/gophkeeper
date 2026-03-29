package models

import (
	"testing"
	"time"
)

func TestRecordValidate_ValidLoginRecord(t *testing.T) {
	r := &Record{
		UserID:         1,
		Type:           RecordTypeLogin,
		Name:           "My Login",
		Payload:        LoginPayload{Login: "user", Password: "pass"},
		Revision:       1,
		DeviceID:       "device-1",
		KeyVersion:     1,
		PayloadVersion: 0,
	}
	if err := r.Validate(); err != nil {
		t.Fatalf("expected valid record, got error: %v", err)
	}
}

func TestRecordValidate_ValidTextRecord(t *testing.T) {
	r := &Record{
		UserID:         1,
		Type:           RecordTypeText,
		Name:           "My Note",
		Payload:        TextPayload{Content: "some text"},
		Revision:       1,
		DeviceID:       "device-1",
		KeyVersion:     1,
		PayloadVersion: 0,
	}
	if err := r.Validate(); err != nil {
		t.Fatalf("expected valid record, got error: %v", err)
	}
}

func TestRecordValidate_ValidBinaryRecord(t *testing.T) {
	r := &Record{
		UserID:         1,
		Type:           RecordTypeBinary,
		Name:           "My File",
		Payload:        BinaryPayload{Data: []byte("binary")},
		Revision:       1,
		DeviceID:       "device-1",
		KeyVersion:     1,
		PayloadVersion: 1,
	}
	if err := r.Validate(); err != nil {
		t.Fatalf("expected valid record, got error: %v", err)
	}
}

func TestRecordValidate_ValidCardRecord(t *testing.T) {
	r := &Record{
		UserID:         1,
		Type:           RecordTypeCard,
		Name:           "My Card",
		Payload:        CardPayload{Number: "4111111111111111", CVV: "123"},
		Revision:       1,
		DeviceID:       "device-1",
		KeyVersion:     1,
		PayloadVersion: 0,
	}
	if err := r.Validate(); err != nil {
		t.Fatalf("expected valid record, got error: %v", err)
	}
}

func TestRecordValidate_InvalidType(t *testing.T) {
	r := &Record{
		UserID:     1,
		Type:       RecordType("unknown"),
		Name:       "test",
		Payload:    TextPayload{Content: "data"},
		DeviceID:   "device-1",
		KeyVersion: 1,
	}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error for invalid record type")
	}
}

func TestRecordValidate_InvalidUserID(t *testing.T) {
	r := &Record{
		UserID:     0,
		Type:       RecordTypeText,
		Name:       "test",
		Payload:    TextPayload{Content: "data"},
		DeviceID:   "device-1",
		KeyVersion: 1,
	}
	if err := r.Validate(); err != ErrInvalidUserID {
		t.Fatalf("expected ErrInvalidUserID, got: %v", err)
	}
}

func TestRecordValidate_EmptyName(t *testing.T) {
	r := &Record{
		UserID:     1,
		Type:       RecordTypeText,
		Name:       "",
		Payload:    TextPayload{Content: "data"},
		DeviceID:   "device-1",
		KeyVersion: 1,
	}
	if err := r.Validate(); err != ErrEmptyRecordName {
		t.Fatalf("expected ErrEmptyRecordName, got: %v", err)
	}
}

func TestRecordValidate_EmptyDeviceID(t *testing.T) {
	r := &Record{
		UserID:     1,
		Type:       RecordTypeText,
		Name:       "test",
		Payload:    TextPayload{Content: "data"},
		DeviceID:   "",
		KeyVersion: 1,
	}
	if err := r.Validate(); err != ErrEmptyDeviceID {
		t.Fatalf("expected ErrEmptyDeviceID, got: %v", err)
	}
}

func TestRecordValidate_InvalidKeyVersion(t *testing.T) {
	r := &Record{
		UserID:     1,
		Type:       RecordTypeText,
		Name:       "test",
		Payload:    TextPayload{Content: "data"},
		DeviceID:   "device-1",
		KeyVersion: 0,
	}
	if err := r.Validate(); err != ErrInvalidKeyVersion {
		t.Fatalf("expected ErrInvalidKeyVersion, got: %v", err)
	}
}

func TestRecordValidate_InvalidPayloadVersion(t *testing.T) {
	r := &Record{
		UserID:         1,
		Type:           RecordTypeBinary,
		Name:           "test",
		Payload:        BinaryPayload{Data: []byte("data")},
		DeviceID:       "device-1",
		KeyVersion:     1,
		PayloadVersion: 0,
	}
	if err := r.Validate(); err != ErrInvalidPayloadVersion {
		t.Fatalf("expected ErrInvalidPayloadVersion, got: %v", err)
	}
}

func TestRecordValidate_NilPayload(t *testing.T) {
	r := &Record{
		UserID:     1,
		Type:       RecordTypeText,
		Name:       "test",
		Payload:    nil,
		DeviceID:   "device-1",
		KeyVersion: 1,
	}
	if err := r.Validate(); err != ErrNilPayload {
		t.Fatalf("expected ErrNilPayload, got: %v", err)
	}
}

func TestRecordValidate_PayloadTypeMismatch(t *testing.T) {
	r := &Record{
		UserID:     1,
		Type:       RecordTypeLogin,
		Name:       "test",
		Payload:    TextPayload{Content: "data"},
		DeviceID:   "device-1",
		KeyVersion: 1,
	}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error for payload type mismatch")
	}
}

func TestRecordIsDeleted(t *testing.T) {
	now := time.Now()
	r := &Record{DeletedAt: &now}
	if !r.IsDeleted() {
		t.Fatal("expected record to be deleted")
	}

	r2 := &Record{DeletedAt: nil}
	if r2.IsDeleted() {
		t.Fatal("expected record to not be deleted")
	}
}

func TestRecordBumpRevision(t *testing.T) {
	r := &Record{Revision: 3}
	if err := r.BumpRevision(5, "device-2"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Revision != 5 {
		t.Fatalf("expected revision 5, got %d", r.Revision)
	}
	if r.DeviceID != "device-2" {
		t.Fatalf("expected device_id 'device-2', got %s", r.DeviceID)
	}
}

func TestRecordBumpRevision_NotMonotonic(t *testing.T) {
	r := &Record{Revision: 5}
	if err := r.BumpRevision(5, "device-1"); err != ErrRevisionNotMonotonic {
		t.Fatalf("expected ErrRevisionNotMonotonic, got: %v", err)
	}
	if err := r.BumpRevision(3, "device-1"); err != ErrRevisionNotMonotonic {
		t.Fatalf("expected ErrRevisionNotMonotonic, got: %v", err)
	}
}

func TestRecordBumpRevision_EmptyDeviceID(t *testing.T) {
	r := &Record{Revision: 1}
	if err := r.BumpRevision(2, ""); err != ErrEmptyDeviceID {
		t.Fatalf("expected ErrEmptyDeviceID, got: %v", err)
	}
}

func TestRecordSoftDelete(t *testing.T) {
	r := &Record{}
	if err := r.SoftDelete(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !r.IsDeleted() {
		t.Fatal("expected record to be deleted")
	}
	if r.DeletedAt == nil {
		t.Fatal("expected DeletedAt to be set")
	}
}

func TestRecordSoftDelete_AlreadyDeleted(t *testing.T) {
	r := &Record{}
	_ = r.SoftDelete()
	if err := r.SoftDelete(); err != ErrAlreadyDeleted {
		t.Fatalf("expected ErrAlreadyDeleted, got: %v", err)
	}
}

func TestRecordRestore(t *testing.T) {
	r := &Record{}
	_ = r.SoftDelete()
	if err := r.Restore(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.IsDeleted() {
		t.Fatal("expected record to be restored")
	}
	if r.DeletedAt != nil {
		t.Fatal("expected DeletedAt to be nil after restore")
	}
}

func TestRecordRestore_NotDeleted(t *testing.T) {
	r := &Record{}
	if err := r.Restore(); err != ErrNotDeleted {
		t.Fatalf("expected ErrNotDeleted, got: %v", err)
	}
}
