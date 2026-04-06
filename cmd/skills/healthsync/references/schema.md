# SQLite Schema Reference

## Schema variants

There are 4 schema shapes. The vast majority of tables use the standard 5-column schema:

| Variant | Tables |
|---------|--------|
| Standard 5-col (`value REAL, unit TEXT`) | all except the three below |
| 4-col no-unit (`value TEXT`) | `sleep`, `mindful_sessions`, `stand_hours` |
| 6-col blood pressure | `blood_pressure` |
| 10-col workouts | `workouts` |

All tables have:
- `id INTEGER PRIMARY KEY AUTOINCREMENT`
- `created_at TEXT DEFAULT CURRENT_TIMESTAMP`
- `UNIQUE` constraint for dedup (exact columns vary by variant)
- `CREATE INDEX ON <table>(start_date)` for efficient date-range queries

---

## Cardiac / Vitals

### heart_rate

```sql
CREATE TABLE heart_rate (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    source_name TEXT NOT NULL,
    start_date  TEXT NOT NULL,
    end_date    TEXT NOT NULL,
    value       REAL NOT NULL,    -- count/min (BPM)
    unit        TEXT NOT NULL,
    created_at  TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_name, start_date, end_date, value)
);
CREATE INDEX idx_heart_rate_start_date ON heart_rate(start_date);
```

### resting_heart_rate

```sql
CREATE TABLE resting_heart_rate (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    source_name TEXT NOT NULL,
    start_date  TEXT NOT NULL,
    end_date    TEXT NOT NULL,
    value       REAL NOT NULL,    -- count/min (BPM); one reading per day
    unit        TEXT NOT NULL,
    created_at  TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_name, start_date, end_date, value)
);
CREATE INDEX idx_resting_heart_rate_start_date ON resting_heart_rate(start_date);
```

### hrv

```sql
CREATE TABLE hrv (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    source_name TEXT NOT NULL,
    start_date  TEXT NOT NULL,
    end_date    TEXT NOT NULL,
    value       REAL NOT NULL,    -- ms (HRV SDNN); nightly
    unit        TEXT NOT NULL,
    created_at  TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_name, start_date, end_date, value)
);
CREATE INDEX idx_hrv_start_date ON hrv(start_date);
```

### heart_rate_recovery

```sql
CREATE TABLE heart_rate_recovery (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    source_name TEXT NOT NULL,
    start_date  TEXT NOT NULL,
    end_date    TEXT NOT NULL,
    value       REAL NOT NULL,    -- count/min; drop in HR 1 min after exercise
    unit        TEXT NOT NULL,
    created_at  TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_name, start_date, end_date, value)
);
CREATE INDEX idx_heart_rate_recovery_start_date ON heart_rate_recovery(start_date);
```

### respiratory_rate

```sql
CREATE TABLE respiratory_rate (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    source_name TEXT NOT NULL,
    start_date  TEXT NOT NULL,
    end_date    TEXT NOT NULL,
    value       REAL NOT NULL,    -- breaths/min
    unit        TEXT NOT NULL,
    created_at  TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_name, start_date, end_date, value)
);
CREATE INDEX idx_respiratory_rate_start_date ON respiratory_rate(start_date);
```

### spo2

```sql
CREATE TABLE spo2 (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    source_name TEXT NOT NULL,
    start_date  TEXT NOT NULL,
    end_date    TEXT NOT NULL,
    value       REAL NOT NULL,    -- fraction (0.98 = 98%)
    unit        TEXT NOT NULL DEFAULT '%',
    created_at  TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_name, start_date, end_date, value)
);
CREATE INDEX idx_spo2_start_date ON spo2(start_date);
```

**Note:** Values are stored as fractions (0.0–1.0), not percentages. 0.98 means 98%.

### vo2_max

```sql
CREATE TABLE vo2_max (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    source_name TEXT NOT NULL,
    start_date  TEXT NOT NULL,
    end_date    TEXT NOT NULL,
    value       REAL NOT NULL,    -- mL/min·kg
    unit        TEXT NOT NULL,
    created_at  TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_name, start_date, end_date, value)
);
CREATE INDEX idx_vo2_max_start_date ON vo2_max(start_date);
```

### blood_pressure

Stored as a paired reading — systolic and diastolic matched by `source_name + start_date` during parsing and written as a single row. Unpaired records are silently dropped.

```sql
CREATE TABLE blood_pressure (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    source_name TEXT NOT NULL,
    start_date  TEXT NOT NULL,
    end_date    TEXT NOT NULL,
    systolic    REAL NOT NULL,    -- mmHg
    diastolic   REAL NOT NULL,    -- mmHg
    unit        TEXT NOT NULL,    -- "mmHg"
    created_at  TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_name, start_date, end_date, systolic, diastolic)
);
CREATE INDEX idx_blood_pressure_start_date ON blood_pressure(start_date);
```

### walking_heart_rate

```sql
CREATE TABLE walking_heart_rate (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    source_name TEXT NOT NULL,
    start_date  TEXT NOT NULL,
    end_date    TEXT NOT NULL,
    value       REAL NOT NULL,    -- count/min; average HR while walking
    unit        TEXT NOT NULL,
    created_at  TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_name, start_date, end_date, value)
);
CREATE INDEX idx_walking_heart_rate_start_date ON walking_heart_rate(start_date);
```

---

## Activity / Energy

### steps

```sql
CREATE TABLE steps (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    source_name TEXT NOT NULL,
    start_date  TEXT NOT NULL,
    end_date    TEXT NOT NULL,
    value       REAL NOT NULL,    -- count
    unit        TEXT NOT NULL DEFAULT 'count',
    created_at  TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_name, start_date, end_date, value)
);
CREATE INDEX idx_steps_start_date ON steps(start_date);
```

**Note:** Supports `--total` for deduplicated daily totals (Watch > iPhone priority).

### active_energy

```sql
CREATE TABLE active_energy (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    source_name TEXT NOT NULL,
    start_date  TEXT NOT NULL,
    end_date    TEXT NOT NULL,
    value       REAL NOT NULL,    -- kcal
    unit        TEXT NOT NULL,
    created_at  TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_name, start_date, end_date, value)
);
CREATE INDEX idx_active_energy_start_date ON active_energy(start_date);
```

**Note:** Supports `--total` for deduplicated daily totals.

### basal_energy

```sql
CREATE TABLE basal_energy (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    source_name TEXT NOT NULL,
    start_date  TEXT NOT NULL,
    end_date    TEXT NOT NULL,
    value       REAL NOT NULL,    -- kcal (resting/BMR energy)
    unit        TEXT NOT NULL,
    created_at  TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_name, start_date, end_date, value)
);
CREATE INDEX idx_basal_energy_start_date ON basal_energy(start_date);
```

**Note:** Supports `--total` for deduplicated daily totals.

### exercise_time

```sql
CREATE TABLE exercise_time (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    source_name TEXT NOT NULL,
    start_date  TEXT NOT NULL,
    end_date    TEXT NOT NULL,
    value       REAL NOT NULL,    -- minutes
    unit        TEXT NOT NULL,
    created_at  TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_name, start_date, end_date, value)
);
CREATE INDEX idx_exercise_time_start_date ON exercise_time(start_date);
```

### stand_time

```sql
CREATE TABLE stand_time (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    source_name TEXT NOT NULL,
    start_date  TEXT NOT NULL,
    end_date    TEXT NOT NULL,
    value       REAL NOT NULL,    -- minutes
    unit        TEXT NOT NULL,
    created_at  TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_name, start_date, end_date, value)
);
CREATE INDEX idx_stand_time_start_date ON stand_time(start_date);
```

### flights_climbed

```sql
CREATE TABLE flights_climbed (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    source_name TEXT NOT NULL,
    start_date  TEXT NOT NULL,
    end_date    TEXT NOT NULL,
    value       REAL NOT NULL,    -- count
    unit        TEXT NOT NULL,
    created_at  TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_name, start_date, end_date, value)
);
CREATE INDEX idx_flights_climbed_start_date ON flights_climbed(start_date);
```

### distance_walking_running

```sql
CREATE TABLE distance_walking_running (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    source_name TEXT NOT NULL,
    start_date  TEXT NOT NULL,
    end_date    TEXT NOT NULL,
    value       REAL NOT NULL,    -- km or mi depending on device locale
    unit        TEXT NOT NULL,
    created_at  TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_name, start_date, end_date, value)
);
CREATE INDEX idx_distance_walking_running_start_date ON distance_walking_running(start_date);
```

### distance_cycling

```sql
CREATE TABLE distance_cycling (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    source_name TEXT NOT NULL,
    start_date  TEXT NOT NULL,
    end_date    TEXT NOT NULL,
    value       REAL NOT NULL,    -- km or mi depending on device locale
    unit        TEXT NOT NULL,
    created_at  TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_name, start_date, end_date, value)
);
CREATE INDEX idx_distance_cycling_start_date ON distance_cycling(start_date);
```

### time_in_daylight

```sql
CREATE TABLE time_in_daylight (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    source_name TEXT NOT NULL,
    start_date  TEXT NOT NULL,
    end_date    TEXT NOT NULL,
    value       REAL NOT NULL,    -- minutes
    unit        TEXT NOT NULL,
    created_at  TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_name, start_date, end_date, value)
);
CREATE INDEX idx_time_in_daylight_start_date ON time_in_daylight(start_date);
```

### physical_effort

```sql
CREATE TABLE physical_effort (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    source_name TEXT NOT NULL,
    start_date  TEXT NOT NULL,
    end_date    TEXT NOT NULL,
    value       REAL NOT NULL,    -- MET score
    unit        TEXT NOT NULL,
    created_at  TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_name, start_date, end_date, value)
);
CREATE INDEX idx_physical_effort_start_date ON physical_effort(start_date);
```

---

## Body

### body_mass

```sql
CREATE TABLE body_mass (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    source_name TEXT NOT NULL,
    start_date  TEXT NOT NULL,
    end_date    TEXT NOT NULL,
    value       REAL NOT NULL,    -- kg or lb depending on device locale
    unit        TEXT NOT NULL,
    created_at  TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_name, start_date, end_date, value)
);
CREATE INDEX idx_body_mass_start_date ON body_mass(start_date);
```

### body_mass_index

```sql
CREATE TABLE body_mass_index (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    source_name TEXT NOT NULL,
    start_date  TEXT NOT NULL,
    end_date    TEXT NOT NULL,
    value       REAL NOT NULL,    -- BMI (kg/m²)
    unit        TEXT NOT NULL,
    created_at  TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_name, start_date, end_date, value)
);
CREATE INDEX idx_body_mass_index_start_date ON body_mass_index(start_date);
```

### height

```sql
CREATE TABLE height (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    source_name TEXT NOT NULL,
    start_date  TEXT NOT NULL,
    end_date    TEXT NOT NULL,
    value       REAL NOT NULL,    -- m or ft depending on device locale
    unit        TEXT NOT NULL,
    created_at  TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_name, start_date, end_date, value)
);
CREATE INDEX idx_height_start_date ON height(start_date);
```

### dietary_water

```sql
CREATE TABLE dietary_water (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    source_name TEXT NOT NULL,
    start_date  TEXT NOT NULL,
    end_date    TEXT NOT NULL,
    value       REAL NOT NULL,    -- mL or L depending on device locale
    unit        TEXT NOT NULL,
    created_at  TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_name, start_date, end_date, value)
);
CREATE INDEX idx_dietary_water_start_date ON dietary_water(start_date);
```

---

## Mobility / Walking

### walking_speed

```sql
CREATE TABLE walking_speed (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    source_name TEXT NOT NULL,
    start_date  TEXT NOT NULL,
    end_date    TEXT NOT NULL,
    value       REAL NOT NULL,    -- m/s
    unit        TEXT NOT NULL,
    created_at  TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_name, start_date, end_date, value)
);
CREATE INDEX idx_walking_speed_start_date ON walking_speed(start_date);
```

### walking_step_length

```sql
CREATE TABLE walking_step_length (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    source_name TEXT NOT NULL,
    start_date  TEXT NOT NULL,
    end_date    TEXT NOT NULL,
    value       REAL NOT NULL,    -- m
    unit        TEXT NOT NULL,
    created_at  TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_name, start_date, end_date, value)
);
CREATE INDEX idx_walking_step_length_start_date ON walking_step_length(start_date);
```

### walking_asymmetry

```sql
CREATE TABLE walking_asymmetry (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    source_name TEXT NOT NULL,
    start_date  TEXT NOT NULL,
    end_date    TEXT NOT NULL,
    value       REAL NOT NULL,    -- % (gait asymmetry)
    unit        TEXT NOT NULL,
    created_at  TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_name, start_date, end_date, value)
);
CREATE INDEX idx_walking_asymmetry_start_date ON walking_asymmetry(start_date);
```

### walking_double_support

```sql
CREATE TABLE walking_double_support (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    source_name TEXT NOT NULL,
    start_date  TEXT NOT NULL,
    end_date    TEXT NOT NULL,
    value       REAL NOT NULL,    -- % of gait cycle with both feet on ground
    unit        TEXT NOT NULL,
    created_at  TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_name, start_date, end_date, value)
);
CREATE INDEX idx_walking_double_support_start_date ON walking_double_support(start_date);
```

### walking_steadiness

```sql
CREATE TABLE walking_steadiness (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    source_name TEXT NOT NULL,
    start_date  TEXT NOT NULL,
    end_date    TEXT NOT NULL,
    value       REAL NOT NULL,    -- % (fall risk score; higher = more steady)
    unit        TEXT NOT NULL,
    created_at  TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_name, start_date, end_date, value)
);
CREATE INDEX idx_walking_steadiness_start_date ON walking_steadiness(start_date);
```

### stair_ascent_speed

```sql
CREATE TABLE stair_ascent_speed (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    source_name TEXT NOT NULL,
    start_date  TEXT NOT NULL,
    end_date    TEXT NOT NULL,
    value       REAL NOT NULL,    -- ft/s
    unit        TEXT NOT NULL,
    created_at  TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_name, start_date, end_date, value)
);
CREATE INDEX idx_stair_ascent_speed_start_date ON stair_ascent_speed(start_date);
```

### stair_descent_speed

```sql
CREATE TABLE stair_descent_speed (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    source_name TEXT NOT NULL,
    start_date  TEXT NOT NULL,
    end_date    TEXT NOT NULL,
    value       REAL NOT NULL,    -- ft/s
    unit        TEXT NOT NULL,
    created_at  TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_name, start_date, end_date, value)
);
CREATE INDEX idx_stair_descent_speed_start_date ON stair_descent_speed(start_date);
```

### six_minute_walk

```sql
CREATE TABLE six_minute_walk (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    source_name TEXT NOT NULL,
    start_date  TEXT NOT NULL,
    end_date    TEXT NOT NULL,
    value       REAL NOT NULL,    -- m (distance covered in 6-minute walk test)
    unit        TEXT NOT NULL,
    created_at  TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_name, start_date, end_date, value)
);
CREATE INDEX idx_six_minute_walk_start_date ON six_minute_walk(start_date);
```

---

## Running metrics

### running_speed

```sql
CREATE TABLE running_speed (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    source_name TEXT NOT NULL,
    start_date  TEXT NOT NULL,
    end_date    TEXT NOT NULL,
    value       REAL NOT NULL,    -- m/s
    unit        TEXT NOT NULL,
    created_at  TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_name, start_date, end_date, value)
);
CREATE INDEX idx_running_speed_start_date ON running_speed(start_date);
```

### running_power

```sql
CREATE TABLE running_power (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    source_name TEXT NOT NULL,
    start_date  TEXT NOT NULL,
    end_date    TEXT NOT NULL,
    value       REAL NOT NULL,    -- W
    unit        TEXT NOT NULL,
    created_at  TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_name, start_date, end_date, value)
);
CREATE INDEX idx_running_power_start_date ON running_power(start_date);
```

### running_stride_length

```sql
CREATE TABLE running_stride_length (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    source_name TEXT NOT NULL,
    start_date  TEXT NOT NULL,
    end_date    TEXT NOT NULL,
    value       REAL NOT NULL,    -- m
    unit        TEXT NOT NULL,
    created_at  TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_name, start_date, end_date, value)
);
CREATE INDEX idx_running_stride_length_start_date ON running_stride_length(start_date);
```

### running_ground_contact_time

```sql
CREATE TABLE running_ground_contact_time (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    source_name TEXT NOT NULL,
    start_date  TEXT NOT NULL,
    end_date    TEXT NOT NULL,
    value       REAL NOT NULL,    -- ms (foot contact time per step)
    unit        TEXT NOT NULL,
    created_at  TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_name, start_date, end_date, value)
);
CREATE INDEX idx_running_ground_contact_time_start_date ON running_ground_contact_time(start_date);
```

### running_vertical_oscillation

```sql
CREATE TABLE running_vertical_oscillation (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    source_name TEXT NOT NULL,
    start_date  TEXT NOT NULL,
    end_date    TEXT NOT NULL,
    value       REAL NOT NULL,    -- cm (bounce per stride)
    unit        TEXT NOT NULL,
    created_at  TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_name, start_date, end_date, value)
);
CREATE INDEX idx_running_vertical_oscillation_start_date ON running_vertical_oscillation(start_date);
```

---

## Other quantity types

### wrist_temperature

```sql
CREATE TABLE wrist_temperature (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    source_name TEXT NOT NULL,
    start_date  TEXT NOT NULL,
    end_date    TEXT NOT NULL,
    value       REAL NOT NULL,    -- °C deviation from personal baseline (nightly)
    unit        TEXT NOT NULL,
    created_at  TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_name, start_date, end_date, value)
);
CREATE INDEX idx_wrist_temperature_start_date ON wrist_temperature(start_date);
```

---

## Category types (no unit column — value TEXT)

These tables store Apple Health category values rather than numeric measurements. There is no `unit` column.

### sleep

```sql
CREATE TABLE sleep (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    source_name TEXT NOT NULL,
    start_date  TEXT NOT NULL,
    end_date    TEXT NOT NULL,
    value       TEXT NOT NULL,    -- HKCategoryValueSleepAnalysis*
    created_at  TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_name, start_date, end_date, value)
);
CREATE INDEX idx_sleep_start_date ON sleep(start_date);
```

**Sleep stage values:**

| Value | Meaning |
|-------|---------|
| `HKCategoryValueSleepAnalysisInBed` | In bed |
| `HKCategoryValueSleepAnalysisAsleepCore` | Core sleep |
| `HKCategoryValueSleepAnalysisAsleepDeep` | Deep sleep |
| `HKCategoryValueSleepAnalysisAsleepREM` | REM sleep |
| `HKCategoryValueSleepAnalysisAwake` | Awake |
| `HKCategoryValueSleepAnalysisAsleepUnspecified` | Unspecified |

Session duration = `end_date - start_date`. Supports `--total` for nightly sleep hours.

### mindful_sessions

```sql
CREATE TABLE mindful_sessions (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    source_name TEXT NOT NULL,
    start_date  TEXT NOT NULL,
    end_date    TEXT NOT NULL,
    value       TEXT NOT NULL,    -- HKCategoryValueNotApplicable
    created_at  TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_name, start_date, end_date, value)
);
CREATE INDEX idx_mindful_sessions_start_date ON mindful_sessions(start_date);
```

Session duration = `end_date - start_date`.

### stand_hours

```sql
CREATE TABLE stand_hours (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    source_name TEXT NOT NULL,
    start_date  TEXT NOT NULL,
    end_date    TEXT NOT NULL,
    value       TEXT NOT NULL,    -- HKCategoryValueAppleStandHourStood or HKCategoryValueAppleStandHourIdle
    created_at  TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_name, start_date, end_date, value)
);
CREATE INDEX idx_stand_hours_start_date ON stand_hours(start_date);
```

---

## Workouts

### workouts

Distance and energy fields are nullable — not all workout types track them (e.g. Yoga has no distance).

```sql
CREATE TABLE workouts (
    id                       INTEGER PRIMARY KEY AUTOINCREMENT,
    activity_type            TEXT NOT NULL,    -- HKWorkoutActivityType*
    source_name              TEXT NOT NULL,
    start_date               TEXT NOT NULL,
    end_date                 TEXT NOT NULL,
    duration                 REAL,             -- minutes (nullable)
    duration_unit            TEXT,             -- "min"
    total_distance           REAL,             -- nullable
    total_distance_unit      TEXT,             -- "km", "mi", etc.
    total_energy_burned      REAL,             -- nullable
    total_energy_burned_unit TEXT,             -- "kcal"
    created_at               TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(activity_type, start_date, end_date, source_name)
);
CREATE INDEX idx_workouts_start_date ON workouts(start_date);
```

Common `activity_type` values: `HKWorkoutActivityTypeRunning`, `HKWorkoutActivityTypeWalking`, `HKWorkoutActivityTypeCycling`, `HKWorkoutActivityTypeYoga`, `HKWorkoutActivityTypeSwimming`, `HKWorkoutActivityTypeHighIntensityIntervalTraining`, `HKWorkoutActivityTypeTraditionalStrengthTraining`.

---

## Date format

All dates are stored as text in local time (timezone offset stripped at parse time):

```
2024-01-15 08:30:00
```

Compatible with SQLite's `date()`, `julianday()`, and other date functions. When filtering with SQLite `WHERE`, use `date(start_date)` for day-level grouping or string prefix comparison for ranges.

---

## Common query patterns

### Daily step totals

```sql
SELECT date(start_date) as day, ROUND(SUM(value)) as total_steps
FROM steps
GROUP BY day
ORDER BY day DESC
LIMIT 7;
```

### Weekly average heart rate

```sql
SELECT strftime('%Y-W%W', start_date) as week,
       ROUND(AVG(value), 1) as avg_bpm
FROM heart_rate
GROUP BY week
ORDER BY week DESC
LIMIT 12;
```

### Blood pressure history

```sql
SELECT date(start_date) as day, systolic, diastolic
FROM blood_pressure
ORDER BY start_date DESC
LIMIT 30;
```

### Sleep duration per night (asleep only)

Via CLI: `healthsync query sleep --total --from 2024-01-01`

Or via SQL (note the `-6 hours` shift for correct night grouping):
```sql
SELECT date(start_date, '-6 hours') as night,
       ROUND(SUM((julianday(end_date) - julianday(start_date)) * 24), 1) as hours
FROM sleep
WHERE value LIKE '%Asleep%'
GROUP BY night
ORDER BY night DESC
LIMIT 14;
```

### Workout summary by type

```sql
SELECT activity_type,
       COUNT(*) as count,
       ROUND(AVG(duration), 1) as avg_min,
       ROUND(SUM(total_energy_burned)) as total_kcal
FROM workouts
GROUP BY activity_type
ORDER BY count DESC;
```

### VO2 Max trend (weekly)

```sql
SELECT strftime('%Y-W%W', start_date) as week,
       ROUND(AVG(value), 2) as avg_vo2
FROM vo2_max
GROUP BY week
ORDER BY week DESC
LIMIT 12;
```

### Mindfulness minutes per day

```sql
SELECT date(start_date) as day,
       ROUND(SUM((julianday(end_date) - julianday(start_date)) * 1440), 1) as minutes
FROM mindful_sessions
GROUP BY day
ORDER BY day DESC
LIMIT 30;
```

### HRV trend (nightly)

```sql
SELECT date(start_date) as night, ROUND(AVG(value), 1) as avg_hrv_ms
FROM hrv
GROUP BY night
ORDER BY night DESC
LIMIT 30;
```

### Body weight trend

```sql
SELECT date(start_date) as day, ROUND(AVG(value), 2) as weight, unit
FROM body_mass
GROUP BY day
ORDER BY day DESC
LIMIT 30;
```
