# healthsync

Parse Apple Health exports into a local SQLite database — queryable by AI agents, the CLI, or directly via SQL.

The primary motivation is to make your Apple Health data accessible to AI coding agents. Run `healthsync skills install` to give Claude Code, Codex CLI, or OpenClaw the schema, CLI reference, and SQL examples it needs to answer questions about your health data in conversation.

**Docs**: [healthsync.sidv.dev](https://healthsync.sidv.dev)

## Install

```bash
curl -fsSL https://healthsync.sidv.dev/install | bash
```

Or install with Go:

```bash
go install github.com/BRO3886/healthsync@latest
```

Or download a pre-built binary from [GitHub Releases](https://github.com/BRO3886/healthsync/releases) (macOS and Linux, arm64 and amd64).

Or build from source (requires Go 1.24+):

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
healthsync query resting-heart-rate --limit 30
healthsync query blood-pressure --limit 20
healthsync query body-mass --limit 30
```

Output formats: `table` (default), `json`, `csv`

Use `--total` with `steps`, `active-energy`, or `basal-energy` for deduplicated daily totals:

```bash
healthsync query steps --total --from 2024-01-01
healthsync query active-energy --total --from 2024-01-01
```

### AI agent skills

Install the healthsync skill to teach your AI coding agent how to query your health data:

```bash
healthsync skills install
```

This writes the database schema, CLI reference, and SQL query examples to `~/.claude/skills/healthsync/` (Claude Code), `~/.agents/skills/healthsync/` (Codex CLI), or `~/.openclaw/skills/healthsync/` (OpenClaw). The agent picks it up automatically on the next session start and can then answer questions like "What was my average heart rate last week?" by running queries against your local database.

```bash
# Check installation status
healthsync skills status

# Uninstall
healthsync skills uninstall --agent claude
```

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

## Metrics

### Currently parsed

**Cardiac / Vitals**

| Table | Apple Health Type | Notes |
|-------|-------------------|-------|
| `heart_rate` | `HKQuantityTypeIdentifierHeartRate` | BPM |
| `resting_heart_rate` | `HKQuantityTypeIdentifierRestingHeartRate` | Daily RHR |
| `hrv` | `HKQuantityTypeIdentifierHeartRateVariabilitySDNN` | ms (SDNN) |
| `heart_rate_recovery` | `HKQuantityTypeIdentifierHeartRateRecoveryOneMinute` | Post-exercise |
| `respiratory_rate` | `HKQuantityTypeIdentifierRespiratoryRate` | Breaths/min |
| `blood_pressure` | `HKQuantityTypeIdentifier BloodPressureSystolic/Diastolic` | Paired mmHg |
| `spo2` | `HKQuantityTypeIdentifierOxygenSaturation` | 0-1 fraction |
| `vo2_max` | `HKQuantityTypeIdentifierVO2Max` | mL/min·kg |

**Activity / Energy**

| Table | Apple Health Type | Notes |
|-------|-------------------|-------|
| `steps` | `HKQuantityTypeIdentifierStepCount` | `--total` supported |
| `active_energy` | `HKQuantityTypeIdentifierActiveEnergyBurned` | kcal; `--total` supported |
| `basal_energy` | `HKQuantityTypeIdentifierBasalEnergyBurned` | kcal; `--total` supported |
| `exercise_time` | `HKQuantityTypeIdentifierAppleExerciseTime` | Minutes |
| `stand_time` | `HKQuantityTypeIdentifierAppleStandTime` | Minutes |
| `flights_climbed` | `HKQuantityTypeIdentifierFlightsClimbed` | Count |
| `distance_walking_running` | `HKQuantityTypeIdentifierDistanceWalkingRunning` | km/mi |
| `distance_cycling` | `HKQuantityTypeIdentifierDistanceCycling` | km/mi |

**Body**

| Table | Apple Health Type | Notes |
|-------|-------------------|-------|
| `body_mass` | `HKQuantityTypeIdentifierBodyMass` | kg/lb |
| `body_mass_index` | `HKQuantityTypeIdentifierBodyMassIndex` | |
| `height` | `HKQuantityTypeIdentifierHeight` | m/ft |

**Mobility / Walking**

| Table | Apple Health Type |
|-------|-------------------|
| `walking_speed` | `HKQuantityTypeIdentifierWalkingSpeed` |
| `walking_step_length` | `HKQuantityTypeIdentifierWalkingStepLength` |
| `walking_asymmetry` | `HKQuantityTypeIdentifierWalkingAsymmetryPercentage` |
| `walking_double_support` | `HKQuantityTypeIdentifierWalkingDoubleSupportPercentage` |
| `walking_steadiness` | `HKQuantityTypeIdentifierAppleWalkingSteadiness` |
| `stair_ascent_speed` | `HKQuantityTypeIdentifierStairAscentSpeed` |
| `stair_descent_speed` | `HKQuantityTypeIdentifierStairDescentSpeed` |
| `six_minute_walk` | `HKQuantityTypeIdentifierSixMinuteWalkTestDistance` |

**Running**

| Table | Apple Health Type |
|-------|-------------------|
| `running_speed` | `HKQuantityTypeIdentifierRunningSpeed` |
| `running_power` | `HKQuantityTypeIdentifierRunningPower` |
| `running_stride_length` | `HKQuantityTypeIdentifierRunningStrideLength` |
| `running_ground_contact_time` | `HKQuantityTypeIdentifierRunningGroundContactTime` |
| `running_vertical_oscillation` | `HKQuantityTypeIdentifierRunningVerticalOscillation` |

**Sleep / Mindfulness / Other**

| Table | Apple Health Type | Notes |
|-------|-------------------|-------|
| `sleep` | `HKCategoryTypeIdentifierSleepAnalysis` | Sleep stages (no unit) |
| `mindful_sessions` | `HKCategoryTypeIdentifierMindfulSession` | Category (no unit) |
| `stand_hours` | `HKCategoryTypeIdentifierAppleStandHour` | Category (no unit) |
| `wrist_temperature` | `HKQuantityTypeIdentifierAppleSleepingWristTemperature` | °C deviation |
| `time_in_daylight` | `HKQuantityTypeIdentifierTimeInDaylight` | Minutes |
| `dietary_water` | `HKQuantityTypeIdentifierDietaryWater` | mL/L |
| `physical_effort` | `HKQuantityTypeIdentifierPhysicalEffort` | MET |
| `walking_heart_rate` | `HKQuantityTypeIdentifierWalkingHeartRateAverage` | BPM |
| `workouts` | All `HKWorkoutActivityType*` | duration, distance, energy |

### Not yet parsed

| Category | Types |
|----------|-------|
| **Audio** | EnvironmentalAudioExposure, HeadphoneAudioExposure, EnvironmentalSoundReduction |
| **Category** | HandwashingEvent, ToothbrushingEvent, MenstrualFlow |

## Design

- **Streaming XML parser** — constant memory (~10MB) for 950MB+ files using `xml.Decoder` token-based parsing
- **Dedup** — `INSERT OR IGNORE` with UNIQUE constraints for idempotent re-imports
- **Batch inserts** — 1000 rows per transaction for performance
- **Async uploads** — HTTP server parses in background, poll `/api/upload/status` for progress
- **Pure Go SQLite** — uses `modernc.org/sqlite`, no CGO required
- **Agent skills** — embedded skill files (go:embed) installed via `healthsync skills install`

Database is stored at `~/.healthsync/healthsync.db` by default (override with `--db`).

## License

MIT
