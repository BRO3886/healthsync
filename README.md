# healthsync

Sync Apple Health export data into a local queryable SQLite database. Two modes: CLI for local file parsing, HTTP server for receiving uploads over Tailscale from iPhone Shortcuts.

## Install

```bash
go install github.com/BRO3886/healthsync@latest
```

Or build from source:

```bash
git clone git@github.com:BRO3886/healthsync.git
cd healthsync
go build -o healthsync .
```

## Usage

### Parse an export

Export your Apple Health data from the Health app on iPhone (Settings > Health > Export All Health Data). Then:

```bash
healthsync parse export.zip
healthsync parse export.zip -v  # verbose logging
```

Accepts `.zip` (auto-extracts `export.xml`) or raw `.xml` files.

### Query data

```bash
healthsync query heart-rate --limit 10
healthsync query steps --from 2024-01-01 --to 2024-06-30
healthsync query workouts --format json
healthsync query spo2 --format csv
```

Available tables: `heart-rate`, `steps`, `spo2`, `vo2max`, `sleep`, `workouts`

Output formats: `table` (default), `json`, `csv`

### HTTP server

Start a server for receiving uploads (e.g. from iPhone Shortcuts over Tailscale):

```bash
healthsync server --port 8080 --host 0.0.0.0
```

Endpoints:

- `POST /api/upload` — upload a `.zip` or `.xml` file (multipart form, field: `file`). Returns `202 Accepted` and parses asynchronously.
- `GET /api/upload/status` — poll parse job progress.
- `GET /api/health/{table}?from=&to=&limit=` — query health data as JSON.

```bash
# Upload
curl -F "file=@export.zip" http://localhost:8080/api/upload

# Check progress
curl http://localhost:8080/api/upload/status

# Query
curl "http://localhost:8080/api/health/heart-rate?limit=5"
```

## Data stored

| Table | Apple Health Type | Fields |
|-------|-------------------|--------|
| `heart_rate` | HeartRate | source, start/end date, value (BPM), unit |
| `steps` | StepCount | source, start/end date, value (count), unit |
| `spo2` | OxygenSaturation | source, start/end date, value (%), unit |
| `vo2_max` | VO2Max | source, start/end date, value, unit |
| `sleep` | SleepAnalysis | source, start/end date, sleep stage |
| `workouts` | Workout | type, source, start/end date, duration, distance, energy |

## Design

- **Streaming XML parser** — constant memory (~10MB) for 950MB+ files using `xml.Decoder` token-based parsing
- **Dedup** — `INSERT OR IGNORE` with UNIQUE constraints for idempotent re-imports
- **Batch inserts** — 1000 rows per transaction for performance
- **Async uploads** — HTTP server parses in background, poll `/api/upload/status` for progress
- **Pure Go SQLite** — uses `modernc.org/sqlite`, no CGO required

Database is stored at `~/.healthsync/healthsync.db` by default (override with `--db`).

## License

MIT
