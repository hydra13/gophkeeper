package commands

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/hydra13/gophkeeper/internal/models"
)

func (r *Runner) RunList(args []string) {
	core, cleanup, err := r.newCore()
	if err != nil {
		r.fatal(err)
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var recordType models.RecordType
	if len(args) > 0 {
		recordType = models.RecordType(args[0])
	}

	records, err := core.ListRecords(ctx, recordType)
	if err != nil {
		r.fatal(err)
	}

	if len(records) == 0 {
		fmt.Fprintln(r.deps.Stdout, "no records")
		return
	}

	fmt.Fprintf(r.deps.Stdout, "%-8s %-10s %-30s %-10s\t%s\n", "ID", "TYPE", "NAME", "REVISION", "METADATA")
	for _, record := range records {
		PrintRecordShort(r.deps.Stdout, record)
	}
}

func (r *Runner) RunGet(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(r.deps.Stderr, "Usage: gophkeeper-cli get <id> [output-path]")
		os.Exit(1)
	}

	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		r.fatal(fmt.Errorf("invalid id: %w", err))
	}

	core, cleanup, err := r.newCore()
	if err != nil {
		r.fatal(err)
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	rec, err := core.GetRecord(ctx, id)
	if err != nil {
		r.fatal(err)
	}

	if rec.Type == models.RecordTypeBinary {
		outputPath := ""
		if len(args) > 1 {
			outputPath = args[1]
		}

		data, err := core.DownloadBinary(ctx, id, 64*1024)
		if err != nil {
			r.fatal(fmt.Errorf("download binary: %w", err))
		}

		if outputPath == "" {
			PrintRecord(r.deps.Stdout, rec)
			fmt.Fprintf(r.deps.Stdout, "Data size: %d bytes\n", len(data))
			fmt.Fprintln(r.deps.Stdout, "Use: gophkeeper-cli get <id> <output-path> to save to file")
			return
		}

		if err := os.WriteFile(outputPath, data, 0600); err != nil {
			r.fatal(fmt.Errorf("write file %s: %w", outputPath, err))
		}
		fmt.Fprintf(r.deps.Stdout, "saved %d bytes to %s\n", len(data), outputPath)
		return
	}

	PrintRecord(r.deps.Stdout, rec)
}

func (r *Runner) RunAdd(args []string) {
	metadata, _, args := ExtractMetadata(args)

	if len(args) < 2 {
		fmt.Fprintln(r.deps.Stderr, "Usage: gophkeeper-cli add <type> <name> [data] [--metadata <text>]")
		fmt.Fprintln(r.deps.Stderr, "  type: login|text|binary|card")
		fmt.Fprintln(r.deps.Stderr, "  binary: data=file-path")
		fmt.Fprintln(r.deps.Stderr, "  other types: omit data to enter interactively")
		os.Exit(1)
	}

	recordType, err := ParseRecordType(args[0])
	if err != nil {
		r.fatal(err)
	}
	name := args[1]
	data := ""
	if len(args) > 2 {
		data = args[2]
	}

	var fileData []byte
	var payload models.RecordPayload

	if recordType == models.RecordTypeBinary {
		if data == "" {
			data = r.readLine("File path: ")
		}
		fileData, err = os.ReadFile(data)
		if err != nil {
			r.fatal(fmt.Errorf("read file %s: %w", data, err))
		}
		payload = models.BinaryPayload{}
	} else {
		payload = r.BuildPayload(recordType, data)
	}

	rec := &models.Record{
		Type:     recordType,
		Name:     name,
		Payload:  payload,
		Metadata: metadata,
	}

	core, cleanup, err := r.newCore()
	if err != nil {
		r.fatal(err)
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := core.SaveRecord(ctx, rec)
	if err != nil {
		r.fatal(err)
	}

	if recordType == models.RecordTypeBinary && len(fileData) > 0 {
		if err := core.UploadBinary(ctx, result.ID, fileData, 64*1024); err != nil {
			r.fatal(fmt.Errorf("upload binary: %w", err))
		}
	}

	fmt.Fprintf(r.deps.Stdout, "added: id=%d rev=%d\n", result.ID, result.Revision)
}

func (r *Runner) RunUpdate(args []string) {
	metadata, metadataFound, args := ExtractMetadata(args)

	if len(args) < 2 {
		fmt.Fprintln(r.deps.Stderr, "Usage: gophkeeper-cli update <id> <name> [data] [--metadata <text>]")
		fmt.Fprintln(r.deps.Stderr, "  binary: data=file-path")
		fmt.Fprintln(r.deps.Stderr, "  --metadata \"\" to clear metadata")
		os.Exit(1)
	}

	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		r.fatal(fmt.Errorf("invalid id: %w", err))
	}

	core, cleanup, err := r.newCore()
	if err != nil {
		r.fatal(err)
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	existing, err := core.GetRecord(ctx, id)
	if err != nil {
		r.fatal(err)
	}

	existing.Name = args[1]

	if metadataFound {
		existing.Metadata = metadata
	}

	var fileData []byte

	if len(args) > 2 {
		if existing.Type == models.RecordTypeBinary {
			fileData, err = os.ReadFile(args[2])
			if err != nil {
				r.fatal(fmt.Errorf("read file %s: %w", args[2], err))
			}
		} else {
			existing.Payload = r.BuildPayload(existing.Type, args[2])
		}
	} else if !metadataFound {
		if existing.Type == models.RecordTypeBinary {
			filePath := r.readLine("File path: ")
			fileData, err = os.ReadFile(filePath)
			if err != nil {
				r.fatal(fmt.Errorf("read file %s: %w", filePath, err))
			}
		} else {
			existing.Payload = r.promptPayload(existing.Type)
		}
	}

	result, err := core.SaveRecord(ctx, existing)
	if err != nil {
		r.fatal(err)
	}

	if existing.Type == models.RecordTypeBinary && len(fileData) > 0 {
		if err := core.UploadBinary(ctx, result.ID, fileData, 64*1024); err != nil {
			r.fatal(fmt.Errorf("upload binary: %w", err))
		}
	}

	fmt.Fprintf(r.deps.Stdout, "updated: id=%d rev=%d\n", result.ID, result.Revision)
}

func (r *Runner) RunDelete(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(r.deps.Stderr, "Usage: gophkeeper-cli delete <id>")
		os.Exit(1)
	}

	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		r.fatal(fmt.Errorf("invalid id: %w", err))
	}

	core, cleanup, err := r.newCore()
	if err != nil {
		r.fatal(err)
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := core.DeleteRecord(ctx, id); err != nil {
		r.fatal(err)
	}

	fmt.Fprintln(r.deps.Stdout, "deleted")
}
