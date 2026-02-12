---
title: "Getting Started"
description: "Install healthsync and parse your first Apple Health export."
date: 2026-02-12T00:00:00+05:30
draft: false
weight: 100
toc: true
---

## Installation

### From source

```bash
git clone git@github.com:BRO3886/healthsync.git
cd healthsync
go build -o healthsync .
```

### With `go install`

```bash
go install github.com/BRO3886/healthsync@latest
```

## Export your Apple Health data

1. Open the **Health** app on your iPhone
2. Tap your profile picture (top right)
3. Scroll down and tap **Export All Health Data**
4. Wait for the export to complete (can take a few minutes)
5. Share the `export.zip` file to your Mac

## Parse the export

```bash
healthsync parse export.zip
```

This will:
- Extract `export.xml` from the zip
- Stream-parse ~500k+ records in constant memory
- Store data in `~/.healthsync/healthsync.db`
- Deduplicate on re-import (safe to run weekly)

Use `-v` for verbose output with progress rate:

```bash
healthsync parse export.zip -v
```

## Query your data

```bash
# Recent heart rate
healthsync query heart-rate --limit 10

# Steps in a date range
healthsync query steps --from 2024-01-01 --to 2024-06-30

# Workouts as JSON
healthsync query workouts --format json

# All available tables
# heart-rate, steps, spo2, vo2max, sleep, workouts
```

## Override database path

All commands accept `--db` to use a custom database location:

```bash
healthsync parse export.zip --db ./my-health.db
healthsync query heart-rate --db ./my-health.db
```
