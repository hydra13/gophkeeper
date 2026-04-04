package clientui

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/hydra13/gophkeeper/internal/models"
)

type PayloadFields struct {
	Login    string
	Password string
	Content  string
	Number   string
	Holder   string
	Expiry   string
	CVV      string
}

func ExtractMetadata(args []string) (metadata string, found bool, rest []string) {
	for i := 0; i < len(args); i++ {
		if args[i] == "--metadata" {
			if i+1 < len(args) {
				return args[i+1], true, append(args[:i:i], args[i+2:]...)
			}
			return "", true, args[:i:i]
		}
	}
	return "", false, args
}

func ParseRecordType(s string) (models.RecordType, error) {
	rt := models.RecordType(strings.ToLower(strings.TrimSpace(s)))
	if !models.ValidRecordTypes[rt] {
		return "", fmt.Errorf("invalid record type %q; valid: login, text, binary, card", s)
	}
	return rt, nil
}

func BuildPayload(recordType models.RecordType, fields PayloadFields) (models.RecordPayload, error) {
	switch recordType {
	case models.RecordTypeLogin:
		return models.LoginPayload{Login: fields.Login, Password: fields.Password}, nil
	case models.RecordTypeText:
		return models.TextPayload{Content: fields.Content}, nil
	case models.RecordTypeCard:
		return models.CardPayload{
			Number:     fields.Number,
			HolderName: fields.Holder,
			ExpiryDate: fields.Expiry,
			CVV:        fields.CVV,
		}, nil
	case models.RecordTypeBinary:
		return models.BinaryPayload{}, nil
	default:
		return nil, fmt.Errorf("unsupported type: %s", recordType)
	}
}

func ReadBinaryFile(path string) ([]byte, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("file path is required")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file %s: %w", path, err)
	}
	return data, nil
}

func WriteBinaryFile(path string, data []byte) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("output path is required")
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write file %s: %w", path, err)
	}
	return nil
}

func PrintRecord(w io.Writer, rec *models.Record) error {
	_, err := fmt.Fprint(w, FormatRecord(rec))
	return err
}

func PrintRecordShort(w io.Writer, rec models.Record) error {
	_, err := fmt.Fprintln(w, FormatRecordShort(rec))
	return err
}

func FormatRecord(rec *models.Record) string {
	if rec == nil {
		return "record is nil\n"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "ID:       %d\n", rec.ID)
	fmt.Fprintf(&b, "Type:     %s\n", rec.Type)
	fmt.Fprintf(&b, "Name:     %s\n", rec.Name)
	fmt.Fprintf(&b, "Revision: %d\n", rec.Revision)
	fmt.Fprintf(&b, "Metadata: %s\n", rec.Metadata)

	switch p := rec.Payload.(type) {
	case models.LoginPayload:
		fmt.Fprintf(&b, "Login:    %s\n", p.Login)
		fmt.Fprintf(&b, "Password: %s\n", p.Password)
	case models.TextPayload:
		fmt.Fprintf(&b, "Content:  %s\n", p.Content)
	case models.BinaryPayload:
		fmt.Fprintf(&b, "Size:     %d bytes\n", len(p.Data))
	case models.CardPayload:
		fmt.Fprintf(&b, "Number:     %s\n", p.Number)
		fmt.Fprintf(&b, "Holder:     %s\n", p.HolderName)
		fmt.Fprintf(&b, "Expiry:     %s\n", p.ExpiryDate)
		fmt.Fprintf(&b, "CVV:        %s\n", p.CVV)
	default:
		fmt.Fprintf(&b, "Payload:  %v\n", rec.Payload)
	}

	if rec.IsDeleted() {
		fmt.Fprintf(&b, "Deleted:  %s\n", rec.DeletedAt.Format(time.RFC3339))
	}

	return b.String()
}

func FormatRecordShort(rec models.Record) string {
	deleted := ""
	if rec.IsDeleted() {
		deleted = " [deleted]"
	}
	meta := MetadataPreview(rec.Metadata, 40)
	if meta != "" {
		meta = "\t" + meta
	}
	return fmt.Sprintf("%d\t%s\t%s\trev=%d%s%s", rec.ID, rec.Type, rec.Name, rec.Revision, deleted, meta)
}

func FormatRecordListItem(rec models.Record) (string, string) {
	main := fmt.Sprintf("%s (%s)", rec.Name, rec.Type)
	secondary := fmt.Sprintf("id=%d rev=%d", rec.ID, rec.Revision)
	if rec.IsDeleted() {
		secondary += " deleted"
	}
	if meta := MetadataPreview(rec.Metadata, 60); meta != "" {
		secondary += " | " + meta
	}
	return main, secondary
}

func MetadataPreview(metadata string, max int) string {
	if metadata == "" {
		return ""
	}
	firstLine := metadata
	if idx := strings.Index(firstLine, "\n"); idx >= 0 {
		firstLine = firstLine[:idx]
	}
	if max > 0 && len(firstLine) > max {
		return firstLine[:max] + "..."
	}
	return firstLine
}
