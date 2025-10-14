![CI](https://github.com/cdinu/myuplink-naive-client/actions/workflows/ci.yml/badge.svg)
[![Go Reference](https://pkg.go.dev/badge/github.com/cdinu/myuplink-naive-client.svg)](https://pkg.go.dev/github.com/cdinu/myuplink-naive-client)
[![Go Report Card](https://goreportcard.com/badge/github.com/cdinu/myuplink-naive-client)](https://goreportcard.com/report/github.com/cdinu/myuplink-naive-client)

# MyUplink Naive Client

This tool retrieves telemetry data from the MyUplink API for a single device. It authenticates with the client-credentials flow, caches bearer tokens between runs, appends the raw payload to JSONL storage, and records each metric in a denormalised SQLite table. The program is designed for cron-style execution: it runs once, persists data, logs to disk, and exits without writing to stdout or stderr.

## Prerequisites

- Go 1.24 or newer.
- API credentials that allow you to request `READSYSTEM` scope from `https://api.myuplink.com`.
- Network access from the machine that will execute the script.

## Getting Started

1. Create an API application at [dev.myuplink.com/apps](https://dev.myuplink.com/apps?activeTab=0). After completing the form you will receive a **client identifier** and **client secret**.
2. Obtain the device ID you want to poll:
   - Open the MyUplink Swagger explorer at [api.myuplink.com/swagger/index.html](https://api.myuplink.com/swagger/index.html).
   - Use the “Authorize” button, supplying the client ID and secret. If you prefer the CLI, base64 encode `client_id:client_secret`:
     ```sh
     printf "%s:%s" "$CLIENT_ID" "$CLIENT_SECRET" | base64
     ```
   - The resulting string is what you place into `config.basicAuth`.
   - With authorization in place, call `GET /v2/systems/me`. The response lists `systems[*].devices[*].id`; pick the device ID that matches the unit you want to monitor.
3. Clone or download this repository (or download the released binary).
4. Initialise the workspace. With the binary, you can let the CLI create a skeleton configuration and default directories:
   ```sh
   ./nibe-fetch -config ./config.json -init
   ```
   This writes placeholder values to `config.json` and creates `cache`, `data`, `logs`, and `state` relative to the config file. Alternatively, copy `config.example.json` to `config.json` and create the directories manually.
5. Edit `config.json` to include:
   - `deviceId`: the value from the `systems[*].devices[*].id` field.
   - `basicAuth`: the base64 output from the previous step.
   - Local paths for `tokenFilePath`, `storageRoot`, `sqlitePath`, and `logsPath`. Production paths might resemble `/home/ubuntu/exploration/2025-10-nibe-ducktaped/…`, while development defaults in the example stay in the project directory.
6. Create the directories referenced in your config if they do not already exist:
   ```sh
   mkdir -p ./cache ./data ./logs ./state
   mkdir -p /home/ubuntu/exploration/2025-10-nibe-ducktaped/{cache,data,logs,state}  # example prod paths
   ```
   The program will also attempt to create missing directories at runtime, but creating them upfront ensures expected permissions.

## Building

Compile the single binary:

```sh
go build -o bin/nibe-fetch ./cmd/nibe-fetch
```

This produces `bin/nibe-fetch`, which you can copy to the host that will run the job.

## Running Manually

Execute the fetcher by pointing it at your configuration file:

```sh
./bin/nibe-fetch -config /path/to/config.json
```

On each run the program will:

- Request or reuse a bearer token stored at `config.tokenFilePath`.
- Fetch device telemetry and append the full JSON response to `config.storageRoot/<deviceId>.<YYYY-MM-DD>.jsonl`.
- Insert all metrics into the `telemetry` table inside `config.sqlitePath`.
- Write log lines to `config.logsPath/nibs-fetch.<YYYY-MM-DD>.log`.

Errors are logged to the daily log file; nothing is printed to stdout or stderr.

## Scheduling with Cron

Add an entry similar to the following (runs every 15 minutes):

```
*/15 * * * * /opt/nibe/bin/nibe-fetch -config /opt/nibe/config.json
```

Ensure the cron user has permission to write to the paths declared in the configuration file. Logs rotate daily by timestamp; rely on external rotation if you need retention limits.

## Token Cache Notes

The token cache stored at `config.tokenFilePath` contains the access token plus its calculated expiry time. The program refreshes the token automatically when it has expired or is missing. Make sure the cache file is writable by the scheduled user.

## Testing

Run the unit tests before deploying changes:

```sh
go test ./...
```

Tests cover token caching and end-to-end persistence (JSONL + SQLite) using test fixtures.

## Development

- Use the provided Makefile for common tasks:
  ```sh
  make fmt   # gofumpt formatting (falls back to gofmt)
  make lint  # installs golangci-lint if missing, then run with .golangci.yml
  make test  # go test ./...
  make build # go build -o bin/nibe-fetch ./cmd/nibe-fetch
  ```
- Continuous integration runs `go test ./...` and `golangci-lint` on pushes and pull requests.
- Release binaries can be produced with [GoReleaser](https://goreleaser.com/):
  ```sh
  goreleaser release --snapshot --clean
  ```

## Troubleshooting

- **Authentication failures**: confirm `basicAuth` contains the base64 client credential string and the client has `READSYSTEM` scope enabled.
- **Permission issues**: check that the configured directories exist and are writable by the runtime user, particularly for the SQLite database and log files.
- **Unexpected schema changes**: the collector logs JSON parsing errors but still writes the raw payload to JSONL so you can inspect differences.
