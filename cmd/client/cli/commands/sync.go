package commands

import (
	"context"
	"time"
)

func (r *Runner) RunSync() {
	core, cleanup, err := r.newCore()
	if err != nil {
		r.fatal(err)
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := core.SyncNow(ctx); err != nil {
		r.fatal(err)
	}

	r.println(r.deps.Stdout, "synced")
}
