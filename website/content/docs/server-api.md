---
title: "Server API"
description: "HTTP API reference for the healthsync server."
date: 2026-02-12T00:00:00+05:30
lastmod: 2026-02-28T00:00:00+05:30
draft: false
weight: 300
toc: true
---

Start the server with:

```bash
healthsync server --port 8080
```

## `POST /api/upload`

Upload an Apple Health export file for parsing.

- **Content-Type:** `multipart/form-data`
- **Field:** `file` — the `.zip` or `.xml` file
- **Max size:** 2GB

**Response:** `202 Accepted`

```json
{
  "status": "accepted",
  "message": "file uploaded, parsing in background",
  "poll": "/api/upload/status"
}
```

Parsing runs asynchronously in the background. Poll `/api/upload/status` for progress.

**Error responses:**

| Code | Reason |
|------|--------|
| `400` | Missing file, unsupported extension |
| `409` | A parse job is already running |

**Example:**

```bash
curl -F "file=@export.zip" http://localhost:8080/api/upload
```

---

## `GET /api/upload/status`

Check the status of the current or most recent parse job.

**Response:** `200 OK`

```json
{
  "status": "running",
  "running": true,
  "records": 340000,
  "workouts": 0,
  "started_at": "2026-02-12T14:30:00+05:30",
  "elapsed": "8.5s"
}
```

**Status values:**

| Status | Description |
|--------|-------------|
| `idle` | No parse has been run since server start |
| `running` | Parse is in progress |
| `completed` | Last parse finished successfully |
| `failed` | Last parse encountered an error |

When completed:

```json
{
  "status": "completed",
  "running": false,
  "records": 540110,
  "workouts": 1011,
  "total_records": 540110,
  "total_workouts": 1011,
  "started_at": "2026-02-12T14:30:00+05:30",
  "elapsed": "30.6s"
}
```

---

## `GET /api/health/{table}`

Query health data as JSON.

**Path parameters:**

| Parameter | Description |
|-----------|-------------|
| `table` | One of: `heart-rate`, `steps`, `spo2`, `vo2max`, `sleep`, `workouts` |

**Query parameters:**

| Parameter | Description | Default |
|-----------|-------------|---------|
| `from` | Filter records from this date (inclusive) | — |
| `to` | Filter records to this date (inclusive) | — |
| `limit` | Maximum records to return | `50` |

**Response:** `200 OK` — JSON array of records

```json
[
  {
    "id": 1,
    "source_name": "Siddhartha's Apple Watch",
    "start_date": "2026-02-12 19:11:31",
    "end_date": "2026-02-12 19:11:31",
    "value": 72,
    "unit": "count/min",
    "created_at": "2026-02-12 14:24:12"
  }
]
```

**Examples:**

```bash
# Recent heart rate
curl "http://localhost:8080/api/health/heart-rate?limit=5"

# Steps in date range
curl "http://localhost:8080/api/health/steps?from=2024-01-01&to=2024-06-30&limit=100"

# All workouts
curl "http://localhost:8080/api/health/workouts?limit=0"
```

---

## Tailscale + iPhone Shortcuts

The server is designed to receive uploads from iPhone over Tailscale:

1. Run `healthsync server` on your Mac
2. Both devices on the same Tailscale network
3. Create an iPhone Shortcut that:
   - Exports Apple Health data
   - Sends it via `POST` to `http://<tailscale-ip>:8080/api/upload`
4. Run the Shortcut weekly for fresh data
