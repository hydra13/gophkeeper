package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/hydra13/gophkeeper/cmd/client/cli/commands"
	"github.com/hydra13/gophkeeper/cmd/client/common"
	"github.com/hydra13/gophkeeper/internal/models"
	clientcore "github.com/hydra13/gophkeeper/pkg/clientcore"
	"github.com/hydra13/gophkeeper/pkg/clientui"
	"golang.org/x/term"
)

var newCoreFunc = common.NewCore

func defaultNewCore() (*clientcore.ClientCore, func(), error) {
	return common.NewCore()
}

func newCore() (*clientcore.ClientCore, func(), error) {
	return newCoreFunc()
}

var fatalFunc = defaultFatal

func defaultFatal(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}

func fatal(err error) {
	fatalFunc(err)
}

func defaultCacheDir() string {
	return common.DefaultCacheDir()
}

func defaultServerAddr() string {
	return common.DefaultServerAddr()
}

func defaultTLSCertFile() string {
	return common.DefaultTLSCertFile()
}

func hostname() string {
	return common.Hostname()
}

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

func cliRunner() *commands.Runner {
	return commands.New(commands.Dependencies{
		NewCore:      newCore,
		Fatal:        fatal,
		ReadPassword: readPassword,
		ReadLine:     readLine,
		Stdout:       os.Stdout,
		Stderr:       os.Stderr,
	})
}

func printUsage() {
	cliRunner().PrintUsage()
}

func extractMetadata(args []string) (metadata string, found bool, rest []string) {
	return clientui.ExtractMetadata(args)
}

func parseRecordType(s string) (models.RecordType, error) {
	return clientui.ParseRecordType(s)
}

func printRecord(rec *models.Record) {
	clientui.PrintRecord(os.Stdout, rec)
}

func printRecordShort(r models.Record) {
	clientui.PrintRecordShort(os.Stdout, r)
}

func buildPayload(recordType models.RecordType, data string) models.RecordPayload {
	return cliRunner().BuildPayload(recordType, data)
}

func runRegister(args []string) {
	cliRunner().RunRegister(args)
}

func runLogin(args []string) {
	cliRunner().RunLogin(args)
}

func runLogout() {
	cliRunner().RunLogout()
}

func runList(args []string) {
	cliRunner().RunList(args)
}

func runGet(args []string) {
	cliRunner().RunGet(args)
}

func runAdd(args []string) {
	cliRunner().RunAdd(args)
}

func runUpdate(args []string) {
	cliRunner().RunUpdate(args)
}

func runDelete(args []string) {
	cliRunner().RunDelete(args)
}

func runSync() {
	cliRunner().RunSync()
}
