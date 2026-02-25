# SQLite Schema Reference

## Standard quantity table schema

Most tables share this structure. Only exceptions are `blood_pressure`, `sleep`, `mindful_sessions`, `stand_hours`, and `workouts` (documented separately below).

```sql
CREATE TABLE <table_name> (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    source_name TEXT NOT NULL,
    start_date  TEXT NOT NULL,
    end_date    TEXT NOT NULL,
    value       REAL NOT NULL,
    unit        TEXT NOT NULL,
    created_at  TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_name, start_date, end_date, value)
);
CREATE INDEX idx_<table_name>_start_date ON <table_name>(start_date);
```

Tables using this schema:

| Table | Unit | Notes |
|-------|------|-------|
| `heart_rate` | count/min (BPM) | High-frequency |
| `resting_heart_rate` | count/min | Daily RHR |
| `hrv` | ms | HRV SDNN; nightly |
| `heart_rate_recovery` | count/min | Post-exercise |
| `respiratory_rate` | count/min | Breaths/min |
| `active_energy` | kcal | `--total` supported |
| `basal_energy` | kcal | `--total` supported |
| `exercise_time` | min | |
| `stand_time` | min | |
| `flights_climbed` | count | |
| `distance_walking_running` | km/mi | |
| `distance_cycling` | km/mi | |
| `body_mass` | kg/lb | |
| `body_mass_index` | count | BMI |
| `height` | m/ft | |
| `walking_speed` | m/s | |
| `walking_step_length` | m | |
| `walking_asymmetry` | % | |
| `walking_double_support` | % | |
| `walking_steadiness` | % | |
| `stair_ascent_speed` | ft/s | |
| `stair_descent_speed` | ft/s | |
| `six_minute_walk` | m | |
| `running_speed` | m/s | |
| `running_power` | W | |
| `running_stride_length` | m | |
| `running_ground_contact_time` | ms | |
| `running_vertical_oscillation` | cm | |
| `spo2` | % (stored 0-1) | 0.98 = 98% |
| `vo2_max` | mL/min·kg | |
| `wrist_temperature` | °C deviation | |
| `time_in_daylight` | min | |
| `dietary_water` | mL/L | |
| `physical_effort` | MET | |
| `walking_heart_rate` | count/min | |

## steps

```sql
CREATE TABLE steps (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source_name TEXT NOT NULL,
    start_date TEXT NOT NULL,
    end_date TEXT NOT NULL,
    value REAL NOT NULL,          -- count
    unit TEXT NOT NULL DEFAULT 'count',
    created_at TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_name, start_date, end_date, value)
);
CREATE INDEX idx_steps_start_date ON steps(start_date);
```

## blood_pressure

Stored as paired reading — systolic and diastolic matched by `source_name + start_date` during parsing.

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

## sleep

```sql
CREATE TABLE sleep (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source_name TEXT NOT NULL,
    start_date TEXT NOT NULL,
    end_date TEXT NOT NULL,
    value TEXT NOT NULL,           -- sleep stage (HKCategoryValueSleepAnalysis*)
    created_at TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_name, start_date, end_date, value)
);
CREATE INDEX idx_sleep_start_date ON sleep(start_date);
```

**Note:** No `unit` column — category type.

**Sleep stage values:**

| Value | Meaning |
|-------|---------|
| `HKCategoryValueSleepAnalysisInBed` | In bed |
| `HKCategoryValueSleepAnalysisAsleepCore` | Core sleep |
| `HKCategoryValueSleepAnalysisAsleepDeep` | Deep sleep |
| `HKCategoryValueSleepAnalysisAsleepREM` | REM sleep |
| `HKCategoryValueSleepAnalysisAwake` | Awake |
| `HKCategoryValueSleepAnalysisAsleepUnspecified` | Unspecified |

## mindful_sessions

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

**Note:** No `unit` column — category type. Session duration = `end_date - start_date`.

## stand_hours

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

**Note:** No `unit` column — category type.

## workouts

```sql
CREATE TABLE workouts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    activity_type TEXT NOT NULL,           -- HKWorkoutActivityType*
    source_name TEXT NOT NULL,
    start_date TEXT NOT NULL,
    end_date TEXT NOT NULL,
    duration REAL,                         -- nullable, in minutes
    duration_unit TEXT,                    -- "min"
    total_distance REAL,                   -- nullable
    total_distance_unit TEXT,              -- "km", "mi", etc.
    total_energy_burned REAL,              -- nullable
    total_energy_burned_unit TEXT,         -- "kcal"
    created_at TEXT DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(activity_type, start_date, end_date, source_name)
);
CREATE INDEX idx_workouts_start_date ON workouts(start_date);
```

**Note:** distance and energy fields are nullable — not all workout types track them (e.g. Yoga has no distance).
