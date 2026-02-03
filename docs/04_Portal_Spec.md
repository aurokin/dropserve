
# Portal Specification

This document defines portal behavior and is the source of truth for lifecycle logic.

## Definitions

- **Portal**: A time-bound capability that authorizes uploads to a single destination directory on the server.
- **Open window**: Period during which the portal can be claimed/used. Default: 15 minutes.
- **One-time portal**: Default. Can be claimed once; uploads are allowed until the open window expires.
- **Reusable portal**: Optional. Can be used multiple times until the duration ends or explicitly closed; does not require claiming.
- **Active upload**: A file upload currently streaming bytes to the server.

## Required properties

- Each portal has a globally-unique, unguessable `portal_id`.
- Each portal is bound to a canonical `dest_abs` directory (set by CLI via `/api/control/portals`).
- Each portal has `open_until = created_at + duration`.
- A portal MUST NOT be forcibly closed while `active_uploads > 0`.

## State machine

States:

1. `OPEN` — Portal exists, not expired, can be claimed.
2. `CLAIMED` — Portal has been claimed by a browser client (one-time portals only).
3. `IN_USE` — One or more uploads are active OR queue has started.
4. `CLOSING` — Portal is scheduled to close after active uploads drain to zero.
5. `CLOSED` — Portal is closed and invalid; temp cleaned.
6. `EXPIRED` — Portal open window ended before use; temp cleaned.

Transitions:

- OPEN -> EXPIRED
  - when now > open_until AND portal not used

- OPEN -> CLAIMED
  - when a browser claims the portal (one-time portal)
  - server returns a `client_token`

- CLAIMED -> IN_USE
  - when the first upload is initialized OR starts streaming

- OPEN -> IN_USE
  - reusable portals may move directly to IN_USE on first upload

- IN_USE -> CLOSING
  - when now > open_until (duration expired), mark `closing_requested = true`

- IN_USE -> CLOSED
  - only when explicitly closed OR duration expired AND uploads drained

- CLOSING -> CLOSED
  - when active_uploads == 0

## Expiration rules

- If portal is never used:
  - when now > open_until, portal becomes EXPIRED and is cleaned.

- If portal is used:
- reaching open_until triggers closing_requested but does not interrupt uploads.
- portal closes only after active_uploads drains to 0.

## Claim rules

One-time portals:

- The first browser to claim receives a `client_token`.
- All subsequent state-changing requests MUST include `X-Client-Token` and match the claimed token.
- If another browser tries to claim, it receives HTTP 409.

Reusable portals:

- Claim is not required; reusable portals ignore the claim/token system.
- Reusable portals should still use capability-token strength and LAN restriction.

## Close rules

- Browser MAY call an explicit close endpoint.
- Server MAY auto-close a reusable portal after an idle timeout (optional).
