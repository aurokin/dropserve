
# Read This First (AI Agents)

You are an AI coding agent working on DropServe.

## Your workflow

1. Read `docs/README.md` for the source-of-truth rules.
2. Read these specs fully before writing code:
   - `docs/portal-lifecycle.md`
   - `docs/api.md`
   - `docs/file-safety.md`
   - `docs/web-ui.md`

3. Choose the next task from `agents/tasks/` in numerical order unless instructed otherwise.
4. For each task:
   - Read the “Required reading” files listed at the top.
   - Implement exactly what the task requests.
   - Add tests and ensure acceptance criteria pass.
   - Update documentation if the implementation reveals missing details.

## Ground rules

- Do not change portal semantics or cleanup invariants without updating the spec docs.
- Do not expose `/api/control/*` via Caddy.
- Do not add resumable upload protocols in v1 unless a task explicitly introduces it.
- Prefer simplicity and reliability over performance tricks.
