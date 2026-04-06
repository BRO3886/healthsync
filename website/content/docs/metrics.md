---
title: "Supported Metrics"
description: "What Apple Health data healthsync parses, and what's available for future support."
date: 2026-02-12T00:00:00+05:30
lastmod: 2026-02-28T00:00:00+05:30
draft: false
weight: 500
toc: true
---

## Currently parsed

### Cardiac / Vitals

| Table | Apple Health Type | Unit | Notes |
|-------|-------------------|------|-------|
| `heart_rate` | `HKQuantityTypeIdentifierHeartRate` | count/min (BPM) | High-frequency |
| `resting_heart_rate` | `HKQuantityTypeIdentifierRestingHeartRate` | count/min | Daily reading |
| `hrv` | `HKQuantityTypeIdentifierHeartRateVariabilitySDNN` | ms | Nightly SDNN |
| `heart_rate_recovery` | `HKQuantityTypeIdentifierHeartRateRecoveryOneMinute` | count/min | Post-exercise |
| `respiratory_rate` | `HKQuantityTypeIdentifierRespiratoryRate` | count/min | Breaths/min |
| `blood_pressure` | `HKQuantityTypeIdentifierBloodPressureSystolic/Diastolic` | mmHg | Paired reading |
| `spo2` | `HKQuantityTypeIdentifierOxygenSaturation` | % (stored 0-1) | 0.98 = 98% |
| `vo2_max` | `HKQuantityTypeIdentifierVO2Max` | mL/min·kg | |

### Activity / Energy

| Table | Apple Health Type | Unit | Notes |
|-------|-------------------|------|-------|
| `steps` | `HKQuantityTypeIdentifierStepCount` | count | `--total` supported |
| `active_energy` | `HKQuantityTypeIdentifierActiveEnergyBurned` | kcal | `--total` supported |
| `basal_energy` | `HKQuantityTypeIdentifierBasalEnergyBurned` | kcal | `--total` supported |
| `exercise_time` | `HKQuantityTypeIdentifierAppleExerciseTime` | min | |
| `stand_time` | `HKQuantityTypeIdentifierAppleStandTime` | min | |
| `flights_climbed` | `HKQuantityTypeIdentifierFlightsClimbed` | count | |
| `distance_walking_running` | `HKQuantityTypeIdentifierDistanceWalkingRunning` | km/mi | |
| `distance_cycling` | `HKQuantityTypeIdentifierDistanceCycling` | km/mi | |

### Body

| Table | Apple Health Type | Unit |
|-------|-------------------|------|
| `body_mass` | `HKQuantityTypeIdentifierBodyMass` | kg/lb |
| `body_mass_index` | `HKQuantityTypeIdentifierBodyMassIndex` | count |
| `height` | `HKQuantityTypeIdentifierHeight` | m/ft |

### Mobility / Walking

| Table | Apple Health Type | Unit |
|-------|-------------------|------|
| `walking_speed` | `HKQuantityTypeIdentifierWalkingSpeed` | m/s |
| `walking_step_length` | `HKQuantityTypeIdentifierWalkingStepLength` | m |
| `walking_asymmetry` | `HKQuantityTypeIdentifierWalkingAsymmetryPercentage` | % |
| `walking_double_support` | `HKQuantityTypeIdentifierWalkingDoubleSupportPercentage` | % |
| `walking_steadiness` | `HKQuantityTypeIdentifierAppleWalkingSteadiness` | % |
| `stair_ascent_speed` | `HKQuantityTypeIdentifierStairAscentSpeed` | ft/s |
| `stair_descent_speed` | `HKQuantityTypeIdentifierStairDescentSpeed` | ft/s |
| `six_minute_walk` | `HKQuantityTypeIdentifierSixMinuteWalkTestDistance` | m |

### Running metrics

| Table | Apple Health Type | Unit |
|-------|-------------------|------|
| `running_speed` | `HKQuantityTypeIdentifierRunningSpeed` | m/s |
| `running_power` | `HKQuantityTypeIdentifierRunningPower` | W |
| `running_stride_length` | `HKQuantityTypeIdentifierRunningStrideLength` | m |
| `running_ground_contact_time` | `HKQuantityTypeIdentifierRunningGroundContactTime` | ms |
| `running_vertical_oscillation` | `HKQuantityTypeIdentifierRunningVerticalOscillation` | cm |

### Sleep / Mindfulness / Category

| Table | Apple Health Type | Notes |
|-------|-------------------|-------|
| `sleep` | `HKCategoryTypeIdentifierSleepAnalysis` | Sleep stages — no unit column; `--total` supported |
| `mindful_sessions` | `HKCategoryTypeIdentifierMindfulSession` | Category — no unit column |
| `stand_hours` | `HKCategoryTypeIdentifierAppleStandHour` | Category — no unit column |

### Other

| Table | Apple Health Type | Unit |
|-------|-------------------|------|
| `wrist_temperature` | `HKQuantityTypeIdentifierAppleSleepingWristTemperature` | °C deviation |
| `time_in_daylight` | `HKQuantityTypeIdentifierTimeInDaylight` | min |
| `dietary_water` | `HKQuantityTypeIdentifierDietaryWater` | mL/L |
| `physical_effort` | `HKQuantityTypeIdentifierPhysicalEffort` | MET |
| `walking_heart_rate` | `HKQuantityTypeIdentifierWalkingHeartRateAverage` | count/min |
| `workouts` | All `HKWorkoutActivityType*` | varies |

## Not yet parsed

These types exist in Apple Health exports. [Open an issue](https://github.com/BRO3886/healthsync/issues) if you'd like support for any of them.

### Audio exposure

| Type | Description |
|------|-------------|
| `HKQuantityTypeIdentifierEnvironmentalAudioExposure` | Environmental noise level |
| `HKQuantityTypeIdentifierHeadphoneAudioExposure` | Headphone volume level |
| `HKQuantityTypeIdentifierEnvironmentalSoundReduction` | Sound reduction |

### Other category types

| Type | Description |
|------|-------------|
| `HKCategoryTypeIdentifierHandwashingEvent` | Handwashing events |
| `HKCategoryTypeIdentifierToothbrushingEvent` | Toothbrushing events |
| `HKCategoryTypeIdentifierMenstrualFlow` | Menstrual cycle tracking |
