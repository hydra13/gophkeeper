package commands

import (
	"io"
	"os"

	"github.com/hydra13/gophkeeper/pkg/clientcore"
)

// Dependencies описывает внешние зависимости Runner.
type Dependencies struct {
	NewCore      func() (*clientcore.ClientCore, func(), error)
	Fatal        func(error)
	ReadPassword func(string) string
	ReadLine     func(string) string
	Stdout       io.Writer
	Stderr       io.Writer
}

// Runner выполняет команды CLI-клиента.
type Runner struct {
	deps Dependencies
}

// New создает Runner и подставляет стандартные stdout и stderr при необходимости.
func New(deps Dependencies) *Runner {
	if deps.Stdout == nil {
		deps.Stdout = os.Stdout
	}
	if deps.Stderr == nil {
		deps.Stderr = os.Stderr
	}
	return &Runner{deps: deps}
}

func (r *Runner) fatal(err error) {
	r.deps.Fatal(err)
}

func (r *Runner) readPassword(prompt string) string {
	return r.deps.ReadPassword(prompt)
}

func (r *Runner) readLine(prompt string) string {
	return r.deps.ReadLine(prompt)
}

func (r *Runner) newCore() (*clientcore.ClientCore, func(), error) {
	return r.deps.NewCore()
}
