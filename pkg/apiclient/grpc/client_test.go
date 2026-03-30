package grpc

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hydra13/gophkeeper/internal/models"
	"github.com/hydra13/gophkeeper/internal/rpc/pbv1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func generateTestCert() (certPEM, keyPEM []byte, err error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{Organization: []string{"Test"}},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour),
		IsCA:         true,
		KeyUsage:     x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return nil, nil, err
	}

	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, nil, err
	}
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	return certPEM, keyPEM, nil
}

// --- domainToProtoRecordType / protoToDomainRecordType ---

func TestDomainToProtoRecordType(t *testing.T) {
	tests := []struct {
		name string
		rt   models.RecordType
		want pbv1.RecordType
	}{
		{name: "login", rt: models.RecordTypeLogin, want: pbv1.RecordType_RECORD_TYPE_LOGIN},
		{name: "text", rt: models.RecordTypeText, want: pbv1.RecordType_RECORD_TYPE_TEXT},
		{name: "binary", rt: models.RecordTypeBinary, want: pbv1.RecordType_RECORD_TYPE_BINARY},
		{name: "card", rt: models.RecordTypeCard, want: pbv1.RecordType_RECORD_TYPE_CARD},
		{name: "unknown returns unspecified", rt: models.RecordType("unknown"), want: pbv1.RecordType_RECORD_TYPE_UNSPECIFIED},
		{name: "empty returns unspecified", rt: models.RecordType(""), want: pbv1.RecordType_RECORD_TYPE_UNSPECIFIED},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := domainToProtoRecordType(tt.rt)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestProtoToDomainRecordType(t *testing.T) {
	tests := []struct {
		name string
		rt   pbv1.RecordType
		want models.RecordType
	}{
		{name: "login", rt: pbv1.RecordType_RECORD_TYPE_LOGIN, want: models.RecordTypeLogin},
		{name: "text", rt: pbv1.RecordType_RECORD_TYPE_TEXT, want: models.RecordTypeText},
		{name: "binary", rt: pbv1.RecordType_RECORD_TYPE_BINARY, want: models.RecordTypeBinary},
		{name: "card", rt: pbv1.RecordType_RECORD_TYPE_CARD, want: models.RecordTypeCard},
		{name: "unspecified returns empty", rt: pbv1.RecordType_RECORD_TYPE_UNSPECIFIED, want: models.RecordType("")},
		{name: "unknown value returns empty", rt: pbv1.RecordType(99), want: models.RecordType("")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := protoToDomainRecordType(tt.rt)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRoundTripRecordType(t *testing.T) {
	types := []models.RecordType{
		models.RecordTypeLogin,
		models.RecordTypeText,
		models.RecordTypeBinary,
		models.RecordTypeCard,
	}
	for _, rt := range types {
		t.Run(string(rt), func(t *testing.T) {
			pb := domainToProtoRecordType(rt)
			got := protoToDomainRecordType(pb)
			assert.Equal(t, rt, got)
		})
	}
}

// --- domainToProtoRecord / protoToDomainRecord ---

func TestDomainToProtoRecord_Nil(t *testing.T) {
	result := domainToProtoRecord(nil)
	assert.Nil(t, result)
}

func TestProtoToDomainRecord_Nil(t *testing.T) {
	result := protoToDomainRecord(nil)
	assert.Nil(t, result)
}

func TestDomainToProtoRecord_LoginPayload(t *testing.T) {
	now := time.Now().Truncate(time.Millisecond)
	deletedAt := now.Add(-1 * time.Hour)

	record := &models.Record{
		ID:             1,
		UserID:         10,
		Type:           models.RecordTypeLogin,
		Name:           "My Login",
		Metadata:       `{"tag":"work"}`,
		Revision:       5,
		DeviceID:       "dev-1",
		KeyVersion:     2,
		PayloadVersion: 1,
		CreatedAt:      now,
		UpdatedAt:      now,
		DeletedAt:      &deletedAt,
		Payload: models.LoginPayload{
			Login:    "user@example.com",
			Password: "secret123",
		},
	}

	pb := domainToProtoRecord(record)
	require.NotNil(t, pb)

	assert.Equal(t, int64(1), pb.Id)
	assert.Equal(t, int64(10), pb.UserId)
	assert.Equal(t, pbv1.RecordType_RECORD_TYPE_LOGIN, pb.Type)
	assert.Equal(t, "My Login", pb.Name)
	assert.Equal(t, `{"tag":"work"}`, pb.Metadata)
	assert.Equal(t, int64(5), pb.Revision)
	assert.Equal(t, "dev-1", pb.DeviceId)
	assert.Equal(t, int64(2), pb.KeyVersion)
	assert.Equal(t, int64(1), pb.PayloadVersion)
	require.NotNil(t, pb.DeletedAt)
	require.NotNil(t, pb.CreatedAt)
	require.NotNil(t, pb.UpdatedAt)

	login := pb.GetLogin()
	require.NotNil(t, login)
	assert.Equal(t, "user@example.com", login.Login)
	assert.Equal(t, "secret123", login.Password)
}

func TestDomainToProtoRecord_TextPayload(t *testing.T) {
	record := &models.Record{
		ID:   2,
		Type: models.RecordTypeText,
		Name: "Notes",
		Payload: models.TextPayload{
			Content: "some text data",
		},
	}

	pb := domainToProtoRecord(record)
	require.NotNil(t, pb)

	text := pb.GetText()
	require.NotNil(t, text)
	assert.Equal(t, "some text data", text.Content)
}

func TestDomainToProtoRecord_BinaryPayload(t *testing.T) {
	record := &models.Record{
		ID:   3,
		Type: models.RecordTypeBinary,
		Name: "File",
		Payload: models.BinaryPayload{
			Data: []byte("binary-data"),
		},
	}

	pb := domainToProtoRecord(record)
	require.NotNil(t, pb)

	bin := pb.GetBinary()
	require.NotNil(t, bin)
}

func TestDomainToProtoRecord_CardPayload(t *testing.T) {
	record := &models.Record{
		ID:   4,
		Type: models.RecordTypeCard,
		Name: "Visa",
		Payload: models.CardPayload{
			Number:     "4111111111111111",
			HolderName: "John Doe",
			ExpiryDate: "12/30",
			CVV:        "123",
		},
	}

	pb := domainToProtoRecord(record)
	require.NotNil(t, pb)

	card := pb.GetCard()
	require.NotNil(t, card)
	assert.Equal(t, "4111111111111111", card.Number)
	assert.Equal(t, "John Doe", card.HolderName)
	assert.Equal(t, "12/30", card.ExpiryDate)
	assert.Equal(t, "123", card.Cvv)
}

func TestDomainToProtoRecord_NoPayload(t *testing.T) {
	record := &models.Record{
		ID:   5,
		Type: models.RecordTypeLogin,
		Name: "Empty",
	}

	pb := domainToProtoRecord(record)
	require.NotNil(t, pb)
	assert.Nil(t, pb.Payload)
}

func TestDomainToProtoRecord_ZeroTimestamps(t *testing.T) {
	record := &models.Record{
		ID:        6,
		Type:      models.RecordTypeText,
		Name:      "No timestamps",
		CreatedAt: time.Time{},
		UpdatedAt: time.Time{},
	}

	pb := domainToProtoRecord(record)
	require.NotNil(t, pb)
	assert.Nil(t, pb.CreatedAt)
	assert.Nil(t, pb.UpdatedAt)
	assert.Nil(t, pb.DeletedAt)
}

func TestProtoToDomainRecord_LoginPayload(t *testing.T) {
	now := time.Now().Truncate(time.Millisecond)
	deletedAt := now.Add(-1 * time.Hour)

	pb := &pbv1.Record{
		Id:             1,
		UserId:         10,
		Type:           pbv1.RecordType_RECORD_TYPE_LOGIN,
		Name:           "My Login",
		Metadata:       `{"tag":"work"}`,
		Revision:       5,
		DeviceId:       "dev-1",
		KeyVersion:     2,
		PayloadVersion: 1,
		DeletedAt:      timestamppb.New(deletedAt),
		CreatedAt:      timestamppb.New(now),
		UpdatedAt:      timestamppb.New(now),
		Payload: &pbv1.Record_Login{
			Login: &pbv1.LoginPayload{Login: "user@example.com", Password: "secret123"},
		},
	}

	r := protoToDomainRecord(pb)
	require.NotNil(t, r)

	assert.Equal(t, int64(1), r.ID)
	assert.Equal(t, int64(10), r.UserID)
	assert.Equal(t, models.RecordTypeLogin, r.Type)
	assert.Equal(t, "My Login", r.Name)
	assert.Equal(t, `{"tag":"work"}`, r.Metadata)
	assert.Equal(t, int64(5), r.Revision)
	assert.Equal(t, "dev-1", r.DeviceID)
	assert.Equal(t, int64(2), r.KeyVersion)
	assert.Equal(t, int64(1), r.PayloadVersion)
	require.NotNil(t, r.DeletedAt)
	assert.False(t, r.CreatedAt.IsZero())
	assert.False(t, r.UpdatedAt.IsZero())

	payload, ok := r.Payload.(models.LoginPayload)
	require.True(t, ok)
	assert.Equal(t, "user@example.com", payload.Login)
	assert.Equal(t, "secret123", payload.Password)
}

func TestProtoToDomainRecord_TextPayload(t *testing.T) {
	pb := &pbv1.Record{
		Id:   2,
		Type: pbv1.RecordType_RECORD_TYPE_TEXT,
		Name: "Notes",
		Payload: &pbv1.Record_Text{
			Text: &pbv1.TextPayload{Content: "some text"},
		},
	}

	r := protoToDomainRecord(pb)
	require.NotNil(t, r)

	payload, ok := r.Payload.(models.TextPayload)
	require.True(t, ok)
	assert.Equal(t, "some text", payload.Content)
}

func TestProtoToDomainRecord_BinaryPayload(t *testing.T) {
	pb := &pbv1.Record{
		Id:   3,
		Type: pbv1.RecordType_RECORD_TYPE_BINARY,
		Name: "File",
		Payload: &pbv1.Record_Binary{
			Binary: &pbv1.BinaryPayload{},
		},
	}

	r := protoToDomainRecord(pb)
	require.NotNil(t, r)

	_, ok := r.Payload.(models.BinaryPayload)
	require.True(t, ok)
}

func TestProtoToDomainRecord_CardPayload(t *testing.T) {
	pb := &pbv1.Record{
		Id:   4,
		Type: pbv1.RecordType_RECORD_TYPE_CARD,
		Name: "Visa",
		Payload: &pbv1.Record_Card{
			Card: &pbv1.CardPayload{
				Number:     "4111111111111111",
				HolderName: "John Doe",
				ExpiryDate: "12/30",
				Cvv:        "123",
			},
		},
	}

	r := protoToDomainRecord(pb)
	require.NotNil(t, r)

	payload, ok := r.Payload.(models.CardPayload)
	require.True(t, ok)
	assert.Equal(t, "4111111111111111", payload.Number)
	assert.Equal(t, "John Doe", payload.HolderName)
	assert.Equal(t, "12/30", payload.ExpiryDate)
	assert.Equal(t, "123", payload.CVV)
}

func TestProtoToDomainRecord_NoPayload(t *testing.T) {
	pb := &pbv1.Record{
		Id:   5,
		Type: pbv1.RecordType_RECORD_TYPE_LOGIN,
		Name: "Empty",
	}

	r := protoToDomainRecord(pb)
	require.NotNil(t, r)
	assert.Nil(t, r.Payload)
}

func TestProtoToDomainRecord_NilTimestamps(t *testing.T) {
	pb := &pbv1.Record{
		Id:        6,
		Name:      "No timestamps",
		DeletedAt: nil,
		CreatedAt: nil,
		UpdatedAt: nil,
	}

	r := protoToDomainRecord(pb)
	require.NotNil(t, r)
	assert.Nil(t, r.DeletedAt)
	assert.True(t, r.CreatedAt.IsZero())
	assert.True(t, r.UpdatedAt.IsZero())
}

// --- Round trip domain -> proto -> domain ---

func TestRoundTripRecord_Login(t *testing.T) {
	now := time.Now().Truncate(time.Millisecond)
	deletedAt := now.Add(-1 * time.Hour)

	original := &models.Record{
		ID:             100,
		UserID:         200,
		Type:           models.RecordTypeLogin,
		Name:           "Round Trip Login",
		Metadata:       "meta",
		Revision:       7,
		DeviceID:       "device-roundtrip",
		KeyVersion:     3,
		PayloadVersion: 1,
		CreatedAt:      now,
		UpdatedAt:      now,
		DeletedAt:      &deletedAt,
		Payload: models.LoginPayload{
			Login:    "rt@example.com",
			Password: "rtpass",
		},
	}

	pb := domainToProtoRecord(original)
	result := protoToDomainRecord(pb)
	require.NotNil(t, result)

	assert.Equal(t, original.ID, result.ID)
	assert.Equal(t, original.UserID, result.UserID)
	assert.Equal(t, original.Type, result.Type)
	assert.Equal(t, original.Name, result.Name)
	assert.Equal(t, original.Metadata, result.Metadata)
	assert.Equal(t, original.Revision, result.Revision)
	assert.Equal(t, original.DeviceID, result.DeviceID)
	assert.Equal(t, original.KeyVersion, result.KeyVersion)
	assert.Equal(t, original.PayloadVersion, result.PayloadVersion)
	require.NotNil(t, result.DeletedAt)
	assert.True(t, original.CreatedAt.Equal(result.CreatedAt))
	assert.True(t, original.UpdatedAt.Equal(result.UpdatedAt))

	payload, ok := result.Payload.(models.LoginPayload)
	require.True(t, ok)
	assert.Equal(t, "rt@example.com", payload.Login)
	assert.Equal(t, "rtpass", payload.Password)
}

func TestRoundTripRecord_Card(t *testing.T) {
	original := &models.Record{
		ID:   300,
		Type: models.RecordTypeCard,
		Name: "Round Trip Card",
		Payload: models.CardPayload{
			Number:     "5500000000000004",
			HolderName: "Jane Smith",
			ExpiryDate: "06/28",
			CVV:        "456",
		},
	}

	pb := domainToProtoRecord(original)
	result := protoToDomainRecord(pb)
	require.NotNil(t, result)

	payload, ok := result.Payload.(models.CardPayload)
	require.True(t, ok)
	assert.Equal(t, "5500000000000004", payload.Number)
	assert.Equal(t, "Jane Smith", payload.HolderName)
	assert.Equal(t, "06/28", payload.ExpiryDate)
	assert.Equal(t, "456", payload.CVV)
}

// --- domainToProtoPayload ---

func TestDomainToProtoPayload_NilPayload(t *testing.T) {
	record := &models.Record{Payload: nil}
	result := domainToProtoPayload(record)
	assert.Nil(t, result)
}

func TestDomainToProtoPayload_Login(t *testing.T) {
	record := &models.Record{
		Payload: models.LoginPayload{Login: "a@b.c", Password: "pw"},
	}

	result := domainToProtoPayload(record)
	require.NotNil(t, result)

	login, ok := result.(*pbv1.CreateRecordRequest_Login)
	require.True(t, ok)
	assert.Equal(t, "a@b.c", login.Login.Login)
	assert.Equal(t, "pw", login.Login.Password)
}

func TestDomainToProtoPayload_Text(t *testing.T) {
	record := &models.Record{
		Payload: models.TextPayload{Content: "hello"},
	}

	result := domainToProtoPayload(record)
	require.NotNil(t, result)

	text, ok := result.(*pbv1.CreateRecordRequest_Text)
	require.True(t, ok)
	assert.Equal(t, "hello", text.Text.Content)
}

func TestDomainToProtoPayload_Binary(t *testing.T) {
	record := &models.Record{
		Payload: models.BinaryPayload{Data: []byte{1, 2, 3}},
	}

	result := domainToProtoPayload(record)
	require.NotNil(t, result)

	_, ok := result.(*pbv1.CreateRecordRequest_Binary)
	require.True(t, ok)
}

func TestDomainToProtoPayload_Card(t *testing.T) {
	record := &models.Record{
		Payload: models.CardPayload{
			Number:     "4111",
			HolderName: "HN",
			ExpiryDate: "01/25",
			CVV:        "999",
		},
	}

	result := domainToProtoPayload(record)
	require.NotNil(t, result)

	card, ok := result.(*pbv1.CreateRecordRequest_Card)
	require.True(t, ok)
	assert.Equal(t, "4111", card.Card.Number)
	assert.Equal(t, "HN", card.Card.HolderName)
	assert.Equal(t, "01/25", card.Card.ExpiryDate)
	assert.Equal(t, "999", card.Card.Cvv)
}

// --- domainToProtoPayloadForUpdate ---

func TestDomainToProtoPayloadForUpdate_NilPayload(t *testing.T) {
	record := &models.Record{Payload: nil}
	result := domainToProtoPayloadForUpdate(record)
	assert.Nil(t, result)
}

func TestDomainToProtoPayloadForUpdate_Login(t *testing.T) {
	record := &models.Record{
		Payload: models.LoginPayload{Login: "u@d.e", Password: "upd"},
	}

	result := domainToProtoPayloadForUpdate(record)
	require.NotNil(t, result)

	login, ok := result.(*pbv1.UpdateRecordRequest_Login)
	require.True(t, ok)
	assert.Equal(t, "u@d.e", login.Login.Login)
	assert.Equal(t, "upd", login.Login.Password)
}

func TestDomainToProtoPayloadForUpdate_Text(t *testing.T) {
	record := &models.Record{
		Payload: models.TextPayload{Content: "updated text"},
	}

	result := domainToProtoPayloadForUpdate(record)
	require.NotNil(t, result)

	text, ok := result.(*pbv1.UpdateRecordRequest_Text)
	require.True(t, ok)
	assert.Equal(t, "updated text", text.Text.Content)
}

func TestDomainToProtoPayloadForUpdate_Binary(t *testing.T) {
	record := &models.Record{
		Payload: models.BinaryPayload{},
	}

	result := domainToProtoPayloadForUpdate(record)
	require.NotNil(t, result)

	_, ok := result.(*pbv1.UpdateRecordRequest_Binary)
	require.True(t, ok)
}

func TestDomainToProtoPayloadForUpdate_Card(t *testing.T) {
	record := &models.Record{
		Payload: models.CardPayload{
			Number:     "1234",
			HolderName: "updHN",
			ExpiryDate: "11/29",
			CVV:        "321",
		},
	}

	result := domainToProtoPayloadForUpdate(record)
	require.NotNil(t, result)

	card, ok := result.(*pbv1.UpdateRecordRequest_Card)
	require.True(t, ok)
	assert.Equal(t, "1234", card.Card.Number)
	assert.Equal(t, "updHN", card.Card.HolderName)
	assert.Equal(t, "11/29", card.Card.ExpiryDate)
	assert.Equal(t, "321", card.Card.Cvv)
}

// --- protoToConflictInfo ---

func TestProtoToConflictInfo_Basic(t *testing.T) {
	pb := &pbv1.SyncConflict{
		Id:             42,
		RecordId:       100,
		LocalRevision:  3,
		ServerRevision: 5,
		Resolved:       false,
	}

	info := protoToConflictInfo(pb)
	assert.Equal(t, int64(42), info.ID)
	assert.Equal(t, int64(100), info.RecordID)
	assert.Equal(t, int64(3), info.LocalRevision)
	assert.Equal(t, int64(5), info.ServerRevision)
	assert.False(t, info.Resolved)
	assert.Nil(t, info.LocalRecord)
	assert.Nil(t, info.ServerRecord)
}

func TestProtoToConflictInfo_WithRecords(t *testing.T) {
	pb := &pbv1.SyncConflict{
		Id:             1,
		RecordId:       200,
		LocalRevision:  10,
		ServerRevision: 12,
		Resolved:       true,
		LocalRecord: &pbv1.Record{
			Id:   200,
			Name: "Local Version",
			Type: pbv1.RecordType_RECORD_TYPE_TEXT,
			Payload: &pbv1.Record_Text{
				Text: &pbv1.TextPayload{Content: "local"},
			},
		},
		ServerRecord: &pbv1.Record{
			Id:   200,
			Name: "Server Version",
			Type: pbv1.RecordType_RECORD_TYPE_TEXT,
			Payload: &pbv1.Record_Text{
				Text: &pbv1.TextPayload{Content: "server"},
			},
		},
	}

	info := protoToConflictInfo(pb)
	assert.Equal(t, int64(1), info.ID)
	assert.Equal(t, int64(200), info.RecordID)
	assert.True(t, info.Resolved)

	require.NotNil(t, info.LocalRecord)
	assert.Equal(t, "Local Version", info.LocalRecord.Name)
	localPayload, ok := info.LocalRecord.Payload.(models.TextPayload)
	require.True(t, ok)
	assert.Equal(t, "local", localPayload.Content)

	require.NotNil(t, info.ServerRecord)
	assert.Equal(t, "Server Version", info.ServerRecord.Name)
	serverPayload, ok := info.ServerRecord.Payload.(models.TextPayload)
	require.True(t, ok)
	assert.Equal(t, "server", serverPayload.Content)
}

func TestProtoToConflictInfo_NilRecords(t *testing.T) {
	pb := &pbv1.SyncConflict{
		Id:             99,
		RecordId:       500,
		LocalRevision:  1,
		ServerRevision: 2,
		Resolved:       false,
		LocalRecord:    nil,
		ServerRecord:   nil,
	}

	info := protoToConflictInfo(pb)
	assert.Nil(t, info.LocalRecord)
	assert.Nil(t, info.ServerRecord)
}

// --- TLS credentials tests ---

func TestTLSCredentialsFromFile_ValidCert(t *testing.T) {
	// Use the dev cert from the repo
	certPath := filepath.Join("..", "..", "..", "configs", "certs", "dev.crt")
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		t.Skip("dev.crt not found, skipping TLS test")
	}

	creds, err := tlsCredentialsFromFile(certPath)
	require.NoError(t, err)
	require.NotNil(t, creds)
}

func TestTLSCredentialsFromFile_NonExistentFile(t *testing.T) {
	_, err := tlsCredentialsFromFile("/nonexistent/path/cert.pem")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read CA cert")
}

func TestTLSCredentialsFromFile_InvalidCertContent(t *testing.T) {
	// Create a temp file with invalid cert content
	tmpDir := t.TempDir()
	certPath := filepath.Join(tmpDir, "bad-cert.crt")
	err := os.WriteFile(certPath, []byte("not a valid certificate"), 0644)
	require.NoError(t, err)

	_, err = tlsCredentialsFromFile(certPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no certificates were parsed")
}

func TestTLSCredentialsFromFile_SelfGeneratedCert(t *testing.T) {
	// Generate a self-signed cert for testing
	tmpDir := t.TempDir()
	certPath := filepath.Join(tmpDir, "test-ca.crt")

	certPEM, keyPEM, err := generateTestCert()
	require.NoError(t, err)
	_ = keyPEM
	err = os.WriteFile(certPath, certPEM, 0644)
	require.NoError(t, err)

	creds, err := tlsCredentialsFromFile(certPath)
	require.NoError(t, err)
	require.NotNil(t, creds)
}

func TestNewClient_TLSConfigError(t *testing.T) {
	ctx := context.Background()

	_, err := NewClient(ctx, Config{
		Address:     "localhost:9090",
		TLSCertFile: "/nonexistent/cert.pem",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "TLS configuration error")
}

func TestNewClient_InsecureNoCert(t *testing.T) {
	ctx := context.Background()

	// Without TLSCertFile, client should use insecure (for testing only)
	client, err := NewClient(ctx, Config{
		Address: "localhost:9090",
	})
	require.NoError(t, err)
	require.NotNil(t, client)
	client.Close()
}
