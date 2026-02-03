# Portal Lifecycle

This document is the source of truth for portal behavior.

## Definitions

- **Portal**: Capability that authorizes uploads to a destination directory.
- **Open window**: Period the portal can be claimed/used (default 15 minutes).
- **One-time portal**: Default; claim required and a client token is issued.
- **Reusable portal**: Optional; no claim required, can be used multiple times.
- **Active upload**: A file currently streaming bytes to the server.

## Required properties

- Portals have unguessable `portal_id` values.
- Portals are bound to a canonical `dest_abs` set by the CLI.
- `open_until = created_at + duration`.
- Portals must not close while `active_uploads > 0`.

## States

- `OPEN`
- `CLAIMED` (one-time only)
- `IN_USE`
- `CLOSING`
- `CLOSED`
- `EXPIRED`

## Transitions

- `OPEN -> EXPIRED` when `now > open_until` and portal unused.
- `OPEN -> CLAIMED` on claim (one-time portals only).
- `CLAIMED -> IN_USE` when first upload starts.
- `OPEN -> IN_USE` for reusable portals when first upload starts.
- `IN_USE -> CLOSING` when duration expires (set closing requested).
- `CLOSING -> CLOSED` when `active_uploads == 0`.
- `IN_USE -> CLOSED` only when explicitly closed or duration expired and uploads drained.

## Expiration rules

- Unused portals expire at `open_until` and are cleaned.
- Used portals can pass `open_until` without interruption; they close after uploads drain.

## Claim rules

- One-time portals issue `client_token` on first claim.
- Subsequent state-changing requests must include matching `X-Client-Token`.
- A second claim attempt returns HTTP 409.
- Reusable portals skip claims and ignore client tokens.

## Close rules

- Browser may call explicit close.
- Server may close after duration expiration once uploads drain.
