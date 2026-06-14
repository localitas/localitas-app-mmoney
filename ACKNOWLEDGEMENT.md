# Acknowledgement

This app contains code forked from [monarchmoney-cli](https://github.com/thedavidweng/monarchmoney-cli)
by David Weng, originally licensed under the MIT License.

The following packages under `internal/` are derived from the original project:

- `internal/monarch/` — Monarch Money GraphQL API client (from `internal/monarch/`)
- `internal/monarchauth/` — Authentication logic (from `internal/auth/`)
- `internal/monarchgql/` — GraphQL HTTP client (from `internal/graphql/`)
- `internal/monarcherr/` — Error types (from `internal/errors/`)
- `internal/queries/` — GraphQL query files (from `queries/`)

Import paths have been adapted for the localitas module structure.
All other application code (store, sync, handlers, UI) is original work.

Thank you to David Weng for the excellent Monarch Money CLI tool.
