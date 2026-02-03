
# Documentation Index

This documentation pack is the **source of truth** for building DropServe.

## Read order (recommended)

1. `01_Product_Brief.md`
2. `02_User_Flows.md`
3. `03_Architecture.md`
4. `04_Portal_Spec.md`
5. `05_API_Spec.md`
6. `06_File_IO_and_Cleanup.md`
7. `07_Path_Safety.md`
8. `08_Web_UI_Spec.md`
9. `09_CLI_Spec.md`
10. `11_Config_Reference.md`
11. `12_Test_Plan.md`
12. `13_Troubleshooting.md`
13. `14_Licensing.md`
14. `15_Development_Setup.md`

## Source-of-truth rules

- **Portal behavior** must match `04_Portal_Spec.md`.
- **Server endpoints** must match `05_API_Spec.md`.
- **File write + cleanup invariants** must match `06_File_IO_and_Cleanup.md`.
- **Path traversal safety** must match `07_Path_Safety.md`.
- **Deployment** uses Caddy-only; see the README example (no NGINX guidance).

When docs conflict:
1) `06_File_IO_and_Cleanup.md` wins for file safety and cleanup.
2) `04_Portal_Spec.md` wins for lifecycle/timers.
3) `05_API_Spec.md` wins for endpoints and payloads.

## Glossary

- **Portal**: A short-lived capability link that authorizes uploads to a specific destination directory.
- **Control API**: CLI-only endpoints under `/api/control/*`; block these in Caddy.
- **Public API**: Proxied by Caddy; used by browsers to claim portals and upload files.
- **Commit**: The atomic rename/move step that makes a file appear in the final destination path.
