package commands

import (
	"fmt"
	"io"
	"os"

	"github.com/hydra13/gophkeeper/internal/models"
	"github.com/hydra13/gophkeeper/pkg/clientui"
)

func ExtractMetadata(args []string) (metadata string, found bool, rest []string) {
	return clientui.ExtractMetadata(args)
}

func ParseRecordType(s string) (models.RecordType, error) {
	return clientui.ParseRecordType(s)
}

func PrintRecord(w io.Writer, rec *models.Record) error {
	return clientui.PrintRecord(w, rec)
}

func PrintRecordShort(w io.Writer, rec models.Record) error {
	return clientui.PrintRecordShort(w, rec)
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
			payload, err := clientui.BuildPayload(recordType, clientui.PayloadFields{Content: data})
			if err != nil {
				r.fatal(err)
			}
			return payload
		}
		return r.promptPayload(recordType)
	case models.RecordTypeCard:
		if data != "" {
			payload, err := clientui.BuildPayload(recordType, clientui.PayloadFields{Number: data})
			if err != nil {
				r.fatal(err)
			}
			return payload
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
