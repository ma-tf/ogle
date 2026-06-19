# Agent Instructions

ogle — A terminal UI for observing and operating Docker Compose projects, no setup required.

## Commands

| Command | What it does |
|---|---|
| `make generate` | Regenerate mockery mocks |
| `make test` | Run tests with race detector |
| `make lint` | `go vet ./...` + `golangci-lint run ./...` |
| `make build` | Build to `./bin/ogle` |

## Agent workflow

- After writing files, run `make lint` before finishing. Run `make test` when feasible.
- If files were written, provide a commit message inline when finishing the task (but do not commit unless asked).

## Progressive disclosure

For deeper context the agent can pull on demand:

| File | Contents |
|---|---|
| [docs/TOOLCHAIN.md](./docs/TOOLCHAIN.md) | Go version, mockery, golangci-lint |
| [docs/CONVENTIONS.md](./docs/CONVENTIONS.md) | Coding conventions |
| [docs/plans/WORKFLOW.md](./docs/plans/WORKFLOW.md) | Plan workflow |
| [docs/SKILLS.md](./docs/SKILLS.md) | Available agent skills |
| [CONTEXT.md](./CONTEXT.md) | Domain terminology — Service, Project, Dashboard, etc. |
| [docs/arch.md](./docs/arch.md) | Package structure, dependency graph |
| [docs/flows.md](./docs/flows.md) | State machines, screen transitions |
| [docs/TESTING.md](./docs/TESTING.md) | Unit test and UI model test conventions |
| [docs/agents/issue-tracker.md](./docs/agents/issue-tracker.md) | GitHub issue tracker conventions |
| [docs/agents/triage-labels.md](./docs/agents/triage-labels.md) | Triage role-to-label mapping |
| [docs/agents/domain.md](./docs/agents/domain.md) | Domain doc consumer rules |

## Agent skills

### Issue tracker

Issues tracked on GitHub Issues via the `gh` CLI. See `docs/agents/issue-tracker.md`.

### Triage labels

Five canonical role labels: `needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`. See `docs/agents/triage-labels.md`.

### Domain docs

Single-context repo. See `docs/agents/domain.md`.

## Because the agent NEVER listens

- NEVER expose internal state on ui components.
- ui components MUST only expose Init, Update, and View methods. Nothing else.
- NEVER export internals for white box testing. White box testing is FORBIDDEN. All tests MUST be black box.
- NO EXCEPTIONS to these rules. These MUST ALWAYS be obeyed. This takes precedence over all other rules or plans.
