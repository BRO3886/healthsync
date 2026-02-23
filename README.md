# healthsync

Sync Apple Health export data into a local queryable SQLite database. Two modes: CLI for local file parsing, HTTP server for receiving uploads over Tailscale from iPhone Shortcuts.

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

Or build from source (requires Go 1.21+):

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

## Metrics

### Currently parsed

| Table        | Apple Health Type                          | Fields                                                   |
| ------------ | ------------------------------------------ | -------------------------------------------------------- |
| `heart_rate` | `HKQuantityTypeIdentifierHeartRate`        | source, start/end date, value (BPM), unit                |
| `steps`      | `HKQuantityTypeIdentifierStepCount`        | source, start/end date, value (count), unit              |
| `spo2`       | `HKQuantityTypeIdentifierOxygenSaturation` | source, start/end date, value (0-1 fraction), unit       |
| `vo2_max`    | `HKQuantityTypeIdentifierVO2Max`           | source, start/end date, value (mL/min·kg), unit          |
| `sleep`      | `HKCategoryTypeIdentifierSleepAnalysis`    | source, start/end date, sleep stage                      |
| `workouts`   | All `HKWorkoutActivityType*`               | type, source, start/end date, duration, distance, energy |

### Available but not yet parsed

These types exist in Apple Health exports but are not currently stored. Open an issue if you'd like support for any of these.

| Category     | Types                                                                                                                                                                    |
| ------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| **Vitals**   | RestingHeartRate, HeartRateVariabilitySDNN, HeartRateRecoveryOneMinute, RespiratoryRate, BloodPressureSystolic/Diastolic                                                 |
| **Activity** | ActiveEnergyBurned, BasalEnergyBurned, AppleExerciseTime, AppleStandTime, FlightsClimbed, DistanceWalkingRunning, DistanceCycling                                        |
| **Body**     | BodyMass, BodyMassIndex, Height                                                                                                                                          |
| **Mobility** | WalkingSpeed, WalkingStepLength, WalkingAsymmetryPercentage, WalkingDoubleSupportPercentage, AppleWalkingSteadiness, StairAscent/DescentSpeed, SixMinuteWalkTestDistance |
| **Running**  | RunningSpeed, RunningPower, RunningStrideLength, RunningGroundContactTime, RunningVerticalOscillation                                                                    |
| **Audio**    | EnvironmentalAudioExposure, HeadphoneAudioExposure, EnvironmentalSoundReduction                                                                                          |
| **Other**    | AppleSleepingWristTemperature, TimeInDaylight, DietaryWater, PhysicalEffort, WalkingHeartRateAverage                                                                     |
| **Category** | MindfulSession, AppleStandHour, HandwashingEvent, ToothbrushingEvent, MenstrualFlow                                                                                      |

## Design

- **Streaming XML parser** — constant memory (~10MB) for 950MB+ files using `xml.Decoder` token-based parsing
- **Dedup** — `INSERT OR IGNORE` with UNIQUE constraints for idempotent re-imports
- **Batch inserts** — 1000 rows per transaction for performance
- **Async uploads** — HTTP server parses in background, poll `/api/upload/status` for progress
- **Pure Go SQLite** — uses `modernc.org/sqlite`, no CGO required

Database is stored at `~/.healthsync/healthsync.db` by default (override with `--db`).

## License

MIT
