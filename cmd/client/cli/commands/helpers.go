package commands

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/hydra13/gophkeeper/internal/models"
)

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
	rt := models.RecordType(strings.ToLower(s))
	if !models.ValidRecordTypes[rt] {
		return "", fmt.Errorf("invalid record type %q; valid: login, text, binary, card", s)
	}
	return rt, nil
}

func PrintRecord(w io.Writer, rec *models.Record) {
	fmt.Fprintf(w, "ID:       %d\n", rec.ID)
	fmt.Fprintf(w, "Type:     %s\n", rec.Type)
	fmt.Fprintf(w, "Name:     %s\n", rec.Name)
	fmt.Fprintf(w, "Revision: %d\n", rec.Revision)
	fmt.Fprintf(w, "Metadata: %s\n", rec.Metadata)

	switch p := rec.Payload.(type) {
	case models.LoginPayload:
		fmt.Fprintf(w, "Login:    %s\n", p.Login)
		fmt.Fprintf(w, "Password: %s\n", p.Password)
	case models.TextPayload:
		fmt.Fprintf(w, "Content:  %s\n", p.Content)
	case models.BinaryPayload:
		fmt.Fprintf(w, "Size:     %d bytes\n", len(p.Data))
	case models.CardPayload:
		fmt.Fprintf(w, "Number:     %s\n", p.Number)
		fmt.Fprintf(w, "Holder:     %s\n", p.HolderName)
		fmt.Fprintf(w, "Expiry:     %s\n", p.ExpiryDate)
		fmt.Fprintf(w, "CVV:        %s\n", p.CVV)
	default:
		fmt.Fprintf(w, "Payload:  %v\n", rec.Payload)
	}

	if rec.IsDeleted() {
		fmt.Fprintf(w, "Deleted:  %s\n", rec.DeletedAt.Format(time.RFC3339))
	}
}

func PrintRecordShort(w io.Writer, r models.Record) {
	deleted := ""
	if r.IsDeleted() {
		deleted = " [deleted]"
	}
	meta := ""
	if r.Metadata != "" {
		firstLine := r.Metadata
		if idx := strings.Index(r.Metadata, "\n"); idx >= 0 {
			firstLine = r.Metadata[:idx]
		}
		if len(firstLine) > 40 {
			firstLine = firstLine[:40] + "..."
		}
		meta = fmt.Sprintf("\t%s", firstLine)
	}
	fmt.Fprintf(w, "%d\t%s\t%s\trev=%d%s%s\n", r.ID, r.Type, r.Name, r.Revision, deleted, meta)
}

func (r *Runner) promptPayload(recordType models.RecordType) models.RecordPayload {
	switch recordType {
	case models.RecordTypeLogin:
		login := r.readLine("Login: ")
		password := r.readPassword("Password: ")
		return models.LoginPayload{Login: login, Password: password}
	case models.RecordTypeText:
		content := r.readLine("Content: ")
		return models.TextPayload{Content: content}
	case models.RecordTypeCard:
		number := r.readLine("Card number: ")
		holder := r.readLine("Holder name: ")
		expiry := r.readLine("Expiry date (MM/YY): ")
		cvv := r.readPassword("CVV: ")
		return models.CardPayload{Number: number, HolderName: holder, ExpiryDate: expiry, CVV: cvv}
	case models.RecordTypeBinary:
		filePath := r.readLine("File path: ")
		fileData, err := os.ReadFile(filePath)
		if err != nil {
			r.fatal(fmt.Errorf("read file %s: %w", filePath, err))
		}
		return models.BinaryPayload{Data: fileData}
	default:
		r.fatal(fmt.Errorf("interactive input not supported for type %q", recordType))
		return nil
	}
}

func (r *Runner) BuildPayload(recordType models.RecordType, data string) models.RecordPayload {
	switch recordType {
	case models.RecordTypeLogin:
		return r.promptPayload(recordType)
	case models.RecordTypeText:
		if data != "" {
			return models.TextPayload{Content: data}
		}
		return r.promptPayload(recordType)
	case models.RecordTypeCard:
		if data != "" {
			return models.CardPayload{Number: data}
		}
		return r.promptPayload(recordType)
	case models.RecordTypeBinary:
		if data == "" {
			data = r.readLine("File path: ")
		}
		fileData, err := os.ReadFile(data)
		if err != nil {
			r.fatal(fmt.Errorf("read file %s: %w", data, err))
		}
		return models.BinaryPayload{Data: fileData}
	default:
		r.fatal(fmt.Errorf("unsupported type: %s", recordType))
		return nil
	}
}
