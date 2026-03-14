# Project Mandates

- Do not perform any git commits unless explicitly requested by the user.
- Before commit must do: `go fmt ./...` in project root and `pnpm format` in `frontend` folder
- do not use `interface{}` in golang, just use `any`
- no autoincrement integer as id primary key.
- Use Go version 1.25 and Node version 24.
- Do not use `fetch`, use `api.ts`
- Do not use `log`, use `pkg/logger` in golang