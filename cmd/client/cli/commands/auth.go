package commands

import (
	"context"
	"time"
)

func (r *Runner) RunRegister(args []string) {
	var email string

	if len(args) >= 1 {
		email = args[0]
	} else {
		email = r.readLine("Email: ")
	}
	password := r.readPassword("Password: ")

	core, cleanup, err := r.newCore()
	if err != nil {
		r.fatal(err)
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := core.Register(ctx, email, password); err != nil {
		r.fatal(err)
	}

	r.println(r.deps.Stdout, "registered successfully")
}

func (r *Runner) RunLogin(args []string) {
	var email string

	if len(args) >= 1 {
		email = args[0]
	} else {
		email = r.readLine("Email: ")
	}
	password := r.readPassword("Password: ")

	core, cleanup, err := r.newCore()
	if err != nil {
		r.fatal(err)
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := core.Login(ctx, email, password); err != nil {
		r.fatal(err)
	}

	r.println(r.deps.Stdout, "logged in successfully")
}

func (r *Runner) RunLogout() {
	core, cleanup, err := r.newCore()
	if err != nil {
		r.fatal(err)
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := core.Logout(ctx); err != nil {
		r.fatal(err)
	}

	r.println(r.deps.Stdout, "logged out")
}
