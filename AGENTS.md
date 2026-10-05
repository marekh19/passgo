# Pass Go project guide

Pass Go replaces Monopoly's paper money with a web app. TigerBeetle records money
movements; SQLite stores sessions and players.

## Project work

For project work without a more specific user request, start with the single task
under **Now** in [project status](docs/STATUS.md). Read its linked task file for
scope and completion checks. **Later** is a queue, not active work.

After every codebase change, compare the result with project status and the
relevant task file. Update them in the same change when behavior, remaining work, or
completion criteria changed. Finish a task only after checking every **Done when**
item. Record any check that could not run and keep the task active until its
remaining completion conditions are met. Follow the completion steps in project
status before starting the next task.

Read [product intent](docs/PRODUCT.md) when making scope or experience decisions.
Check the code for current behavior; use `go.mod` and `Taskfile.yaml` for current
tool versions and commands. For design work, follow the visual-reference guidance
in the linked design task before changing screens.

## Architecture and data

For changes to slices, data flows, auth, or live updates, read
[architecture](docs/ARCHITECTURE.md) for ownership and cross-cutting seams.
Check current behavior in code before extending a pattern; use small hand-written
fakes in slice tests and the existing Go test setup.

## Tooling

- Run project commands through `Taskfile.yaml`, including `task build` and
  `task test`. Its macOS linker setting is needed for tigerbeetle-go.
- Generate templ code with `go tool templ generate`. Commit generated
  `*_templ.go` files. Generate SQL bindings with `sqlc`; leave
  `internal/store/generated/` to the generator.
- Use the standalone Tailwind v4 CLI and CSS-first configuration in `input.css`.
- Run `go mod tidy` after changing imports or Go tools.
