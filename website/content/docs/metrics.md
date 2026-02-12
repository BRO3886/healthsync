---
title: "Supported Metrics"
description: "What Apple Health data healthsync parses, and what's available for future support."
date: 2026-02-12T00:00:00+05:30
draft: false
weight: 500
toc: true
---

## Currently parsed

| Table | Apple Health Type | Unit | Notes |
|-------|-------------------|------|-------|
| `heart_rate` | `HKQuantityTypeIdentifierHeartRate` | count/min (BPM) | |
| `steps` | `HKQuantityTypeIdentifierStepCount` | count | |
| `spo2` | `HKQuantityTypeIdentifierOxygenSaturation` | % (stored as 0-1 fraction) | 0.98 = 98% |
| `vo2_max` | `HKQuantityTypeIdentifierVO2Max` | mL/min·kg | |
| `sleep` | `HKCategoryTypeIdentifierSleepAnalysis` | — (category) | Sleep stages |
| `workouts` | All `HKWorkoutActivityType*` | varies | Duration, distance, energy |

## Available but not yet parsed

These types exist in Apple Health exports. [Open an issue](https://github.com/BRO3886/healthsync/issues) if you'd like support for any of them.

### Vitals

| Type | Description |
|------|-------------|
| `HKQuantityTypeIdentifierRestingHeartRate` | Resting heart rate |
| `HKQuantityTypeIdentifierHeartRateVariabilitySDNN` | HRV (SDNN) |
| `HKQuantityTypeIdentifierHeartRateRecoveryOneMinute` | HR recovery after exercise |
| `HKQuantityTypeIdentifierRespiratoryRate` | Breaths per minute |
| `HKQuantityTypeIdentifierBloodPressureSystolic` | Blood pressure (systolic) |
| `HKQuantityTypeIdentifierBloodPressureDiastolic` | Blood pressure (diastolic) |

### Activity

| Type | Description |
|------|-------------|
| `HKQuantityTypeIdentifierActiveEnergyBurned` | Active calories |
| `HKQuantityTypeIdentifierBasalEnergyBurned` | Resting calories |
| `HKQuantityTypeIdentifierAppleExerciseTime` | Exercise minutes |
| `HKQuantityTypeIdentifierAppleStandTime` | Stand hours |
| `HKQuantityTypeIdentifierFlightsClimbed` | Flights of stairs |
| `HKQuantityTypeIdentifierDistanceWalkingRunning` | Walk/run distance |
| `HKQuantityTypeIdentifierDistanceCycling` | Cycling distance |

### Body

| Type | Description |
|------|-------------|
| `HKQuantityTypeIdentifierBodyMass` | Body weight |
| `HKQuantityTypeIdentifierBodyMassIndex` | BMI |
| `HKQuantityTypeIdentifierHeight` | Height |

### Mobility

| Type | Description |
|------|-------------|
| `HKQuantityTypeIdentifierWalkingSpeed` | Average walking speed |
| `HKQuantityTypeIdentifierWalkingStepLength` | Average step length |
| `HKQuantityTypeIdentifierWalkingAsymmetryPercentage` | Gait asymmetry |
| `HKQuantityTypeIdentifierWalkingDoubleSupportPercentage` | Double support time |
| `HKQuantityTypeIdentifierAppleWalkingSteadiness` | Walking steadiness score |
| `HKQuantityTypeIdentifierStairAscentSpeed` | Stair climbing speed |
| `HKQuantityTypeIdentifierStairDescentSpeed` | Stair descending speed |
| `HKQuantityTypeIdentifierSixMinuteWalkTestDistance` | 6-minute walk test |

### Running metrics

| Type | Description |
|------|-------------|
| `HKQuantityTypeIdentifierRunningSpeed` | Running pace |
| `HKQuantityTypeIdentifierRunningPower` | Running power (watts) |
| `HKQuantityTypeIdentifierRunningStrideLength` | Stride length |
| `HKQuantityTypeIdentifierRunningGroundContactTime` | Ground contact time |
| `HKQuantityTypeIdentifierRunningVerticalOscillation` | Vertical oscillation |

### Audio exposure

| Type | Description |
|------|-------------|
| `HKQuantityTypeIdentifierEnvironmentalAudioExposure` | Environmental noise level |
| `HKQuantityTypeIdentifierHeadphoneAudioExposure` | Headphone volume level |
| `HKQuantityTypeIdentifierEnvironmentalSoundReduction` | Sound reduction |

### Other

| Type | Description |
|------|-------------|
| `HKQuantityTypeIdentifierAppleSleepingWristTemperature` | Wrist temperature during sleep |
| `HKQuantityTypeIdentifierTimeInDaylight` | Time spent in daylight |
| `HKQuantityTypeIdentifierDietaryWater` | Water intake |
| `HKQuantityTypeIdentifierPhysicalEffort` | Physical effort score |
| `HKQuantityTypeIdentifierWalkingHeartRateAverage` | Average HR while walking |

### Category types

| Type | Description |
|------|-------------|
| `HKCategoryTypeIdentifierMindfulSession` | Mindfulness minutes |
| `HKCategoryTypeIdentifierAppleStandHour` | Stand hour achievements |
| `HKCategoryTypeIdentifierHandwashingEvent` | Handwashing events |
| `HKCategoryTypeIdentifierToothbrushingEvent` | Toothbrushing events |
| `HKCategoryTypeIdentifierMenstrualFlow` | Menstrual cycle tracking |
