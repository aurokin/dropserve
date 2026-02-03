
# Agent Workflow

## Branching

- Work in small, reviewable increments.
- Keep each task's work self-contained.
- Prefer adding tests alongside implementation.

## Implementation approach

- Start with the server core:
  - portal store
  - file I/O pipeline
  - cleanup sweeper
- Then build the CLI to open portals.
- Then build the web UI.
- Finally write deployment artifacts and run the test plan.

## Documentation updates

If you discover ambiguity:
- Update the relevant `docs/*.md`
- Add a note to `docs/README.md` if the source-of-truth order changes
