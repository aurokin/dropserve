# Task: T062 - Web UI: sequential upload engine with overall progress and speed

## Goal

Implement the upload queue processor and compute overall progress and transfer speed.

## Required reading (MUST read before starting)

- `docs/web-ui.md`
- `docs/api.md`

## Deliverables

- XHR upload logic
- Overall progress + speed display

## Steps

1. Compute total bytes as sum of queued files.
2. Process uploads sequentially: init -> PUT -> commit response.
3. Use XHR upload progress to compute bytes sent and rolling speed.

## Acceptance criteria

- [ ] Overall progress updates correctly.
- [ ] Speed is displayed and updates smoothly.

## Tests to run

- Manual test with multiple files of different sizes.

## Notes

- None.
