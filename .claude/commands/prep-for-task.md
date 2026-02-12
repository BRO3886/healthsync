Bootstrap context for a new session on the `healthsync` project. Run this BEFORE starting any implementation work.

## Instructions

Launch these exploration subagents **in parallel** to gain comprehensive project context:

### Agent 1: Project Overview
Read these files and summarize the project state:
- `README.md` — features, usage, design
- Auto memory — check `~/.claude/projects/` for a directory matching the current working directory (the folder name is the absolute path with `/` replaced by `-`). Read `memory/MEMORY.md` inside it for accumulated patterns and gotchas.
- `go.mod` — module name and Go version
- Run `git log --oneline -20` for recent commit history
- Run `git status` for working tree state

### Agent 2: Codebase Structure
Explore the full directory tree and map out:
- All Go files in each package (`cmd/`, `internal/parser/`, `internal/storage/`, `internal/server/`)
- Test files and their coverage areas
- Dependencies in `go.mod`

### Agent 3: Journal History
Read `journals/` — find the latest journal file and read ALL sessions. Extract:
- What has been built so far (per session)
- Key technical gotchas and failures
- Architectural decisions and their rationale
- What was deferred and why

## After Exploration

Summarize your findings to the user in this format:

```
## Project State
- Features implemented: <list>
- Features remaining/planned: <list>
- Test coverage: <numbers>

## Recent Changes
<last 5 commits, one line each>

## Key Gotchas to Remember
<top technical pitfalls from journals and memory>

## Ready for Task
<confirm context is loaded, ask what to work on>
```

## Rules
- Use subagents for ALL exploration — maximize parallelism
- Do NOT write or modify any files during prep
- Do NOT start implementation until the user gives you a task
