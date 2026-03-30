package main

import (
	"fmt"
	"os"

	"github.com/hydra13/gophkeeper/cmd/client/common"
	tuiapp "github.com/hydra13/gophkeeper/cmd/client/tui/app"
)

func main() {
	core, cleanup, err := common.NewCore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer cleanup()

	if err := tuiapp.New(core).Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
