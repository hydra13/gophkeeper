package clientui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hydra13/gophkeeper/internal/models"
)

func TestParseRecordType(t *testing.T) {
	got, err := ParseRecordType("LOGIN")
	if err != nil {
		t.Fatalf("ParseRecordType(): %v", err)
	}
	if got != models.RecordTypeLogin {
		t.Fatalf("ParseRecordType() = %q, want %q", got, models.RecordTypeLogin)
	}
}

func TestBuildPayloadCard(t *testing.T) {
	payload, err := BuildPayload(models.RecordTypeCard, PayloadFields{
		Number: "4111111111111111",
		Holder: "Ivan Ivanov",
		Expiry: "12/30",
		CVV:    "123",
	})
	if err != nil {
		t.Fatalf("BuildPayload(): %v", err)
	}

	card, ok := payload.(models.CardPayload)
	if !ok {
		t.Fatalf("payload type = %T, want models.CardPayload", payload)
	}
	if card.HolderName != "Ivan Ivanov" {
		t.Fatalf("card.HolderName = %q, want %q", card.HolderName, "Ivan Ivanov")
	}
}

func TestFormatRecordShortTruncatesMetadata(t *testing.T) {
	record := models.Record{
		ID:       42,
		Type:     models.RecordTypeText,
		Name:     "note",
		Revision: 7,
		Metadata: "this is a very long metadata line that should be truncated for list view",
	}

	got := FormatRecordShort(record)
	if !strings.Contains(got, "rev=7") {
		t.Fatalf("FormatRecordShort() = %q, want revision marker", got)
	}
	if !strings.Contains(got, "...") {
		t.Fatalf("FormatRecordShort() = %q, want truncated metadata", got)
	}
}

func TestReadWriteBinaryFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "payload.bin")
	want := []byte("secret")

	if err := WriteBinaryFile(path, want); err != nil {
		t.Fatalf("WriteBinaryFile(): %v", err)
	}

	got, err := ReadBinaryFile(path)
	if err != nil {
		t.Fatalf("ReadBinaryFile(): %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("ReadBinaryFile() = %q, want %q", string(got), string(want))
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(): %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("file perm = %v, want 0600", info.Mode().Perm())
	}
}
