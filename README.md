# Go Database Visualizer

Web-based tool to visualize database structures as interactive ER diagrams. Supports Postgres, MySQL, SQL Server and (optionally) Oracle. UI is searchable, filterable and zoomable; table visuals are colored proportioinal to table size in bytes.

## Quick start (Windows / PowerShell)

1. Open project root:
```
cd "erddiagram"
```

2. Prepare modules:
```
go mod tidy
```

3. Build (no Oracle):
```
go build ./...
```

4. Run server (example):
```
go run ./cmd/server -port=8080
```

5. Open UI:
http://localhost:8080  
Use the connection form to POST DB settings (or raw DSN) to `/api/connect`. 

## Enabling Oracle (optional)

godror requires Oracle Instant Client and CGO. The project keeps godror optional via a build tag.

- To include Oracle support:
  - Install Oracle Instant Client and ensure CGO is enabled.
  - Add godror if needed:
    ```
    go get github.com/godror/godror@v0.49.5
    ```
  - Build/run with the `oracle` tag:
    ```
    $env:CGO_ENABLED = "1"
    go build -tags oracle ./...
    go run -tags oracle ./cmd/server -driver=godror -dsn="user/pass@host:1521/service"
    ```

- If you encounter godror build errors and do not need Oracle, remove it:
```
go mod edit -droprequire github.com/godror/godror
go mod tidy
go clean -modcache
```

## Testing

Run unit tests:
```
go test ./...
```

## Endpoints

- GET  /api/schema        — returns extracted schema for active connection
- POST /api/connect       — set & test connection (JSON body: type, host, port, username, password, database_name or dsn)
- GET  /api/getConnect    - returns database connection information

## Notes & Troubleshooting

- Module name in this repo: `erddiagram` — ensure imports use this module path.
- Some drivers (e.g., `godror`) require CGO or native libs. Install prerequisites before enabling.
- If you see driver-related build errors, remove optional drivers from go.mod or build with the appropriate tag after installing native dependencies.

Contributions and fixes welcome — open issues or PRs.