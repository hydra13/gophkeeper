package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/hydra13/gophkeeper/internal/models"
	grpcc "github.com/hydra13/gophkeeper/pkg/apiclient/grpc"
	"github.com/hydra13/gophkeeper/pkg/buildinfo"
	"github.com/hydra13/gophkeeper/pkg/cache"
	clientcore "github.com/hydra13/gophkeeper/pkg/clientcore"
	"golang.org/x/term"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "login":
		runLogin(args)
	case "register":
		runRegister(args)
	case "logout":
		runLogout()
	case "list":
		runList(args)
	case "get":
		runGet(args)
	case "add":
		runAdd(args)
	case "update":
		runUpdate(args)
	case "delete":
		runDelete(args)
	case "sync":
		runSync()
	case "version":
		fmt.Printf("gophkeeper-cli %s\n", buildinfo.Short())
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "Usage: gophkeeper-cli <command> [args]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  register                           Register new user")
	fmt.Fprintln(os.Stderr, "  login                              Login")
	fmt.Fprintln(os.Stderr, "  logout                             Logout")
	fmt.Fprintln(os.Stderr, "  list     [type]                    List records (login|text|binary|card)")
	fmt.Fprintln(os.Stderr, "  get      <id> [output-path]        Get record by ID (binary: save to output-path)")
	fmt.Fprintln(os.Stderr, "  add      <type> <name> [data]      Add new record")
	fmt.Fprintln(os.Stderr, "                                    binary: data=file-path")
	fmt.Fprintln(os.Stderr, "                                    --metadata <text>  set metadata")
	fmt.Fprintln(os.Stderr, "  update   <id> <name> [data]        Update existing record")
	fmt.Fprintln(os.Stderr, "                                    binary: data=file-path")
	fmt.Fprintln(os.Stderr, "                                    --metadata <text>  set metadata")
	fmt.Fprintln(os.Stderr, "                                    --metadata \"\"      clear metadata")
	fmt.Fprintln(os.Stderr, "  delete   <id>                      Delete record")
	fmt.Fprintln(os.Stderr, "  sync                               Sync with server")
	fmt.Fprintln(os.Stderr, "  version                            Show version")
}

// ---------- helpers ----------

// extractMetadata extracts a --metadata value from args.
// It returns: the metadata value (empty string if not found), whether the flag was present,
// and the remaining args with the flag and its value removed.
func extractMetadata(args []string) (metadata string, found bool, rest []string) {
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

// newCoreFunc allows tests to inject a mock core factory.
var newCoreFunc = defaultNewCore

func defaultNewCore() (*clientcore.ClientCore, func(), error) {
	ctx := context.Background()
	addr := defaultServerAddr()
	certFile := defaultTLSCertFile()

	if certFile == "" {
		return nil, nil, fmt.Errorf("TLS certificate is required: set GK_TLS_CERT_FILE or run from project root (configs/certs/dev.crt)")
	}

	client, err := grpcc.NewClient(ctx, grpcc.Config{
		Address:     addr,
		TLSCertFile: certFile,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("connect to server: %w", err)
	}

	store, err := cache.NewFileStore(defaultCacheDir())
	if err != nil {
		client.Close()
		return nil, nil, fmt.Errorf("init cache: %w", err)
	}

	core := clientcore.New(client, store, clientcore.Config{
		DeviceID: "cli-" + hostname(),
	})

	core.RestoreAuth()

	cleanup := func() {
		store.Flush()
		client.Close()
	}

	return core, cleanup, nil
}

func newCore() (*clientcore.ClientCore, func(), error) {
	return newCoreFunc()
}

// fatalFunc allows tests to capture fatal errors without os.Exit.
var fatalFunc = defaultFatal

func defaultFatal(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}

func fatal(err error) {
	fatalFunc(err)
}

func defaultCacheDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".gophkeeper", "cache")
}

func defaultServerAddr() string {
	if v := os.Getenv("GK_GRPC_ADDRESS"); v != "" {
		return v
	}
	return "localhost:9090"
}

func defaultTLSCertFile() string {
	if v := os.Getenv("GK_TLS_CERT_FILE"); v != "" {
		return v
	}
	if _, err := os.Stat("configs/certs/dev.crt"); err == nil {
		return "configs/certs/dev.crt"
	}
	return ""
}

func hostname() string {
	name, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return name
}

// readPasswordFunc allows tests to inject a mock password reader.
var readPasswordFunc = defaultReadPassword

func defaultReadPassword(prompt string) string {
	fmt.Fprint(os.Stderr, prompt)
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		fatal(fmt.Errorf("read password: %w", err))
	}
	return string(b)
}

func readPassword(prompt string) string {
	return readPasswordFunc(prompt)
}

// readLineFunc allows tests to inject a mock line reader.
var readLineFunc = defaultReadLine

func defaultReadLine(prompt string) string {
	fmt.Fprint(os.Stderr, prompt)
	var line string
	if _, err := fmt.Scanln(&line); err != nil {
		fatal(fmt.Errorf("read input: %w", err))
	}
	return strings.TrimSpace(line)
}

func readLine(prompt string) string {
	return readLineFunc(prompt)
}

// parseRecordType validates and returns a RecordType.
func parseRecordType(s string) (models.RecordType, error) {
	rt := models.RecordType(strings.ToLower(s))
	if !models.ValidRecordTypes[rt] {
		return "", fmt.Errorf("invalid record type %q; valid: login, text, binary, card", s)
	}
	return rt, nil
}

// printRecord formats and prints a single record.
func printRecord(rec *models.Record) {
	fmt.Printf("ID:       %d\n", rec.ID)
	fmt.Printf("Type:     %s\n", rec.Type)
	fmt.Printf("Name:     %s\n", rec.Name)
	fmt.Printf("Revision: %d\n", rec.Revision)
	fmt.Printf("Metadata: %s\n", rec.Metadata)

	switch p := rec.Payload.(type) {
	case models.LoginPayload:
		fmt.Printf("Login:    %s\n", p.Login)
		fmt.Printf("Password: %s\n", p.Password)
	case models.TextPayload:
		fmt.Printf("Content:  %s\n", p.Content)
	case models.BinaryPayload:
		fmt.Printf("Size:     %d bytes\n", len(p.Data))
	case models.CardPayload:
		fmt.Printf("Number:     %s\n", p.Number)
		fmt.Printf("Holder:     %s\n", p.HolderName)
		fmt.Printf("Expiry:     %s\n", p.ExpiryDate)
		fmt.Printf("CVV:        %s\n", p.CVV)
	default:
		fmt.Printf("Payload:  %v\n", rec.Payload)
	}

	if rec.IsDeleted() {
		fmt.Printf("Deleted:  %s\n", rec.DeletedAt.Format(time.RFC3339))
	}
}

// printRecordShort prints a single-line record summary.
func printRecordShort(r models.Record) {
	deleted := ""
	if r.IsDeleted() {
		deleted = " [deleted]"
	}
	meta := ""
	if r.Metadata != "" {
		// Show first line of metadata for short view
		firstLine := r.Metadata
		if idx := strings.Index(r.Metadata, "\n"); idx >= 0 {
			firstLine = r.Metadata[:idx]
		}
		if len(firstLine) > 40 {
			firstLine = firstLine[:40] + "..."
		}
		meta = fmt.Sprintf("\t%s", firstLine)
	}
	fmt.Printf("%d\t%s\t%s\trev=%d%s%s\n", r.ID, r.Type, r.Name, r.Revision, deleted, meta)
}

// promptPayload interactively asks for payload fields based on recordType.
func promptPayload(recordType models.RecordType) models.RecordPayload {
	switch recordType {
	case models.RecordTypeLogin:
		login := readLine("Login: ")
		password := readPassword("Password: ")
		return models.LoginPayload{Login: login, Password: password}
	case models.RecordTypeText:
		content := readLine("Content: ")
		return models.TextPayload{Content: content}
	case models.RecordTypeCard:
		number := readLine("Card number: ")
		holder := readLine("Holder name: ")
		expiry := readLine("Expiry date (MM/YY): ")
		cvv := readPassword("CVV: ")
		return models.CardPayload{Number: number, HolderName: holder, ExpiryDate: expiry, CVV: cvv}
	case models.RecordTypeBinary:
		filePath := readLine("File path: ")
		fileData, err := os.ReadFile(filePath)
		if err != nil {
			fatal(fmt.Errorf("read file %s: %w", filePath, err))
		}
		return models.BinaryPayload{Data: fileData}
	default:
		fatal(fmt.Errorf("interactive input not supported for type %q", recordType))
		return nil
	}
}

// buildPayload creates a payload from CLI arguments.
// For binary type, data is a file path to read from.
// For other types, interactive prompt is used when data is empty.
func buildPayload(recordType models.RecordType, data string) models.RecordPayload {
	switch recordType {
	case models.RecordTypeLogin:
		return promptPayload(recordType)
	case models.RecordTypeText:
		if data != "" {
			return models.TextPayload{Content: data}
		}
		return promptPayload(recordType)
	case models.RecordTypeCard:
		if data != "" {
			return models.CardPayload{Number: data}
		}
		return promptPayload(recordType)
	case models.RecordTypeBinary:
		if data == "" {
			data = readLine("File path: ")
		}
		fileData, err := os.ReadFile(data)
		if err != nil {
			fatal(fmt.Errorf("read file %s: %w", data, err))
		}
		return models.BinaryPayload{Data: fileData}
	default:
		fatal(fmt.Errorf("unsupported type: %s", recordType))
		return nil
	}
}

// ---------- commands ----------

func runRegister(args []string) {
	var email string

	if len(args) >= 1 {
		email = args[0]
	} else {
		email = readLine("Email: ")
	}
	password := readPassword("Password: ")

	core, cleanup, err := newCore()
	if err != nil {
		fatal(err)
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := core.Register(ctx, email, password); err != nil {
		fatal(err)
	}

	fmt.Println("registered successfully")
}

func runLogin(args []string) {
	var email string

	if len(args) >= 1 {
		email = args[0]
	} else {
		email = readLine("Email: ")
	}
	password := readPassword("Password: ")

	core, cleanup, err := newCore()
	if err != nil {
		fatal(err)
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := core.Login(ctx, email, password); err != nil {
		fatal(err)
	}

	fmt.Println("logged in successfully")
}

func runLogout() {
	core, cleanup, err := newCore()
	if err != nil {
		fatal(err)
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := core.Logout(ctx); err != nil {
		fatal(err)
	}

	fmt.Println("logged out")
}

func runList(args []string) {
	core, cleanup, err := newCore()
	if err != nil {
		fatal(err)
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
		fatal(err)
	}

	if len(records) == 0 {
		fmt.Println("no records")
		return
	}

	fmt.Printf("%-8s %-10s %-30s %-10s\t%s\n", "ID", "TYPE", "NAME", "REVISION", "METADATA")
	for _, r := range records {
		printRecordShort(r)
	}
}

func runGet(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: gophkeeper-cli get <id> [output-path]")
		os.Exit(1)
	}

	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		fatal(fmt.Errorf("invalid id: %w", err))
	}

	core, cleanup, err := newCore()
	if err != nil {
		fatal(err)
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	rec, err := core.GetRecord(ctx, id)
	if err != nil {
		fatal(err)
	}

	if rec.Type == models.RecordTypeBinary {
		// For binary records, download data and save to file
		outputPath := ""
		if len(args) > 1 {
			outputPath = args[1]
		}

		data, err := core.DownloadBinary(ctx, id, 64*1024)
		if err != nil {
			fatal(fmt.Errorf("download binary: %w", err))
		}

		if outputPath == "" {
			printRecord(rec)
			fmt.Printf("Data size: %d bytes\n", len(data))
			fmt.Println("Use: gophkeeper-cli get <id> <output-path> to save to file")
			return
		}

		if err := os.WriteFile(outputPath, data, 0600); err != nil {
			fatal(fmt.Errorf("write file %s: %w", outputPath, err))
		}
		fmt.Printf("saved %d bytes to %s\n", len(data), outputPath)
		return
	}

	printRecord(rec)
}

func runAdd(args []string) {
	metadata, _, args := extractMetadata(args)

	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: gophkeeper-cli add <type> <name> [data] [--metadata <text>]")
		fmt.Fprintln(os.Stderr, "  type: login|text|binary|card")
		fmt.Fprintln(os.Stderr, "  binary: data=file-path")
		fmt.Fprintln(os.Stderr, "  other types: omit data to enter interactively")
		os.Exit(1)
	}

	recordType, err := parseRecordType(args[0])
	if err != nil {
		fatal(err)
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
			data = readLine("File path: ")
		}
		fileData, err = os.ReadFile(data)
		if err != nil {
			fatal(fmt.Errorf("read file %s: %w", data, err))
		}
		payload = models.BinaryPayload{}
	} else {
		payload = buildPayload(recordType, data)
	}

	rec := &models.Record{
		Type:     recordType,
		Name:     name,
		Payload:  payload,
		Metadata: metadata,
	}

	core, cleanup, err := newCore()
	if err != nil {
		fatal(err)
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := core.SaveRecord(ctx, rec)
	if err != nil {
		fatal(err)
	}

	// Upload binary data after record is created
	if recordType == models.RecordTypeBinary && len(fileData) > 0 {
		if err := core.UploadBinary(ctx, result.ID, fileData, 64*1024); err != nil {
			fatal(fmt.Errorf("upload binary: %w", err))
		}
	}

	fmt.Printf("added: id=%d rev=%d\n", result.ID, result.Revision)
}

func runUpdate(args []string) {
	metadata, metadataFound, args := extractMetadata(args)

	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: gophkeeper-cli update <id> <name> [data] [--metadata <text>]")
		fmt.Fprintln(os.Stderr, "  binary: data=file-path")
		fmt.Fprintln(os.Stderr, "  --metadata \"\" to clear metadata")
		os.Exit(1)
	}

	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		fatal(fmt.Errorf("invalid id: %w", err))
	}

	core, cleanup, err := newCore()
	if err != nil {
		fatal(err)
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	existing, err := core.GetRecord(ctx, id)
	if err != nil {
		fatal(err)
	}

	existing.Name = args[1]

	if metadataFound {
		existing.Metadata = metadata
	}

	var fileData []byte

	// If --metadata is the only change (no data arg provided), skip payload update
	if len(args) > 2 {
		// Explicit data argument provided — update payload
		if existing.Type == models.RecordTypeBinary {
			fileData, err = os.ReadFile(args[2])
			if err != nil {
				fatal(fmt.Errorf("read file %s: %w", args[2], err))
			}
		} else {
			existing.Payload = buildPayload(existing.Type, args[2])
		}
	} else if !metadataFound {
		// No data and no metadata — prompt for payload as before
		if existing.Type == models.RecordTypeBinary {
			filePath := readLine("File path: ")
			fileData, err = os.ReadFile(filePath)
			if err != nil {
				fatal(fmt.Errorf("read file %s: %w", filePath, err))
			}
		} else {
			existing.Payload = promptPayload(existing.Type)
		}
	}
	// else: metadata-only update, existing payload is preserved

	result, err := core.SaveRecord(ctx, existing)
	if err != nil {
		fatal(err)
	}

	// Upload new binary data after record is updated
	if existing.Type == models.RecordTypeBinary && len(fileData) > 0 {
		if err := core.UploadBinary(ctx, result.ID, fileData, 64*1024); err != nil {
			fatal(fmt.Errorf("upload binary: %w", err))
		}
	}

	fmt.Printf("updated: id=%d rev=%d\n", result.ID, result.Revision)
}

func runDelete(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: gophkeeper-cli delete <id>")
		os.Exit(1)
	}

	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		fatal(fmt.Errorf("invalid id: %w", err))
	}

	core, cleanup, err := newCore()
	if err != nil {
		fatal(err)
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := core.DeleteRecord(ctx, id); err != nil {
		fatal(err)
	}

	fmt.Println("deleted")
}

func runSync() {
	core, cleanup, err := newCore()
	if err != nil {
		fatal(err)
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := core.SyncNow(ctx); err != nil {
		fatal(err)
	}

	fmt.Println("synced")
}
