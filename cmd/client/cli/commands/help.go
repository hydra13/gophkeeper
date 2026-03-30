package commands

import "fmt"

func (r *Runner) PrintUsage() {
	fmt.Fprintln(r.deps.Stderr, "Usage: gophkeeper-cli <command> [args]")
	fmt.Fprintln(r.deps.Stderr, "")
	fmt.Fprintln(r.deps.Stderr, "Commands:")
	fmt.Fprintln(r.deps.Stderr, "  register                           Register new user")
	fmt.Fprintln(r.deps.Stderr, "  login                              Login")
	fmt.Fprintln(r.deps.Stderr, "  logout                             Logout")
	fmt.Fprintln(r.deps.Stderr, "  list     [type]                    List records (login|text|binary|card)")
	fmt.Fprintln(r.deps.Stderr, "  get      name <name> [path]        Get record by name")
	fmt.Fprintln(r.deps.Stderr, "  get      id <id> [path]            Get record by ID")
	fmt.Fprintln(r.deps.Stderr, "                                    binary: save to path")
	fmt.Fprintln(r.deps.Stderr, "  add      <type> <name> [data]      Add new record")
	fmt.Fprintln(r.deps.Stderr, "                                    binary: data=file-path")
	fmt.Fprintln(r.deps.Stderr, "                                    --metadata <text>  set metadata")
	fmt.Fprintln(r.deps.Stderr, "  update   name <name> <new-name>    Update record by name")
	fmt.Fprintln(r.deps.Stderr, "  update   id <id> <new-name>        Update record by ID")
	fmt.Fprintln(r.deps.Stderr, "                                    binary: data=file-path")
	fmt.Fprintln(r.deps.Stderr, "                                    --metadata <text>  set metadata")
	fmt.Fprintln(r.deps.Stderr, "                                    --metadata \"\"      clear metadata")
	fmt.Fprintln(r.deps.Stderr, "  delete   name <name>               Delete record by name")
	fmt.Fprintln(r.deps.Stderr, "  delete   id <id>                   Delete record by ID")
	fmt.Fprintln(r.deps.Stderr, "  sync                               Sync with server")
	fmt.Fprintln(r.deps.Stderr, "  version                            Show version")
}
