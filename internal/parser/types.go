package parser

// Record represents a single Apple Health Record element.
type Record struct {
	Type       string `xml:"type,attr"`
	Unit       string `xml:"unit,attr"`
	Value      string `xml:"value,attr"`
	SourceName string `xml:"sourceName,attr"`
	StartDate  string `xml:"startDate,attr"`
	EndDate    string `xml:"endDate,attr"`
}

// WorkoutStatistics is a child element of <Workout> that provides aggregated metrics.
type WorkoutStatistics struct {
	Type string `xml:"type,attr"`
	Sum  string `xml:"sum,attr"`
	Unit string `xml:"unit,attr"`
}

// Workout represents a single Apple Health Workout element.
type Workout struct {
	ActivityType          string               `xml:"workoutActivityType,attr"`
	Duration              string               `xml:"duration,attr"`
	DurationUnit          string               `xml:"durationUnit,attr"`
	TotalDistance         string               `xml:"totalDistance,attr"`
	TotalDistanceUnit     string               `xml:"totalDistanceUnit,attr"`
	TotalEnergyBurned     string               `xml:"totalEnergyBurned,attr"`
	TotalEnergyBurnedUnit string               `xml:"totalEnergyBurnedUnit,attr"`
	SourceName            string               `xml:"sourceName,attr"`
	StartDate             string               `xml:"startDate,attr"`
	EndDate               string               `xml:"endDate,attr"`
	Statistics            []WorkoutStatistics  `xml:"WorkoutStatistics"`
}

// TargetRecordTypes is the set of HK types we care about, mapping to table names.
// Blood pressure uses sentinel names "blood_pressure_systolic"/"blood_pressure_diastolic"
// and is paired in the parser before writing to the "blood_pressure" table.
var TargetRecordTypes = map[string]string{
	// Existing
	"HKQuantityTypeIdentifierHeartRate":        "heart_rate",
	"HKQuantityTypeIdentifierStepCount":        "steps",
	"HKQuantityTypeIdentifierOxygenSaturation": "spo2",
	"HKQuantityTypeIdentifierVO2Max":           "vo2_max",
	"HKCategoryTypeIdentifierSleepAnalysis":    "sleep",

	// Cardiac vitals
	"HKQuantityTypeIdentifierRestingHeartRate":           "resting_heart_rate",
	"HKQuantityTypeIdentifierHeartRateVariabilitySDNN":   "hrv",
	"HKQuantityTypeIdentifierHeartRateRecoveryOneMinute": "heart_rate_recovery",
	"HKQuantityTypeIdentifierRespiratoryRate":            "respiratory_rate",
	// Blood pressure: paired in parser, stored in blood_pressure table
	"HKQuantityTypeIdentifierBloodPressureSystolic":  "blood_pressure_systolic",
	"HKQuantityTypeIdentifierBloodPressureDiastolic": "blood_pressure_diastolic",

	// Activity / Energy
	"HKQuantityTypeIdentifierActiveEnergyBurned": "active_energy",
	"HKQuantityTypeIdentifierBasalEnergyBurned":  "basal_energy",
	"HKQuantityTypeIdentifierAppleExerciseTime":  "exercise_time",
	"HKQuantityTypeIdentifierAppleStandTime":     "stand_time",
	"HKQuantityTypeIdentifierFlightsClimbed":     "flights_climbed",

	// Distance
	"HKQuantityTypeIdentifierDistanceWalkingRunning": "distance_walking_running",
	"HKQuantityTypeIdentifierDistanceCycling":        "distance_cycling",

	// Body composition
	"HKQuantityTypeIdentifierBodyMass":      "body_mass",
	"HKQuantityTypeIdentifierBodyMassIndex": "body_mass_index",
	"HKQuantityTypeIdentifierHeight":        "height",

	// Mobility / Walking
	"HKQuantityTypeIdentifierWalkingSpeed":                   "walking_speed",
	"HKQuantityTypeIdentifierWalkingStepLength":              "walking_step_length",
	"HKQuantityTypeIdentifierWalkingAsymmetryPercentage":     "walking_asymmetry",
	"HKQuantityTypeIdentifierWalkingDoubleSupportPercentage": "walking_double_support",
	"HKQuantityTypeIdentifierAppleWalkingSteadiness":         "walking_steadiness",
	"HKQuantityTypeIdentifierStairAscentSpeed":               "stair_ascent_speed",
	"HKQuantityTypeIdentifierStairDescentSpeed":              "stair_descent_speed",
	"HKQuantityTypeIdentifierSixMinuteWalkTestDistance":      "six_minute_walk",

	// Running metrics
	"HKQuantityTypeIdentifierRunningSpeed":               "running_speed",
	"HKQuantityTypeIdentifierRunningPower":               "running_power",
	"HKQuantityTypeIdentifierRunningStrideLength":        "running_stride_length",
	"HKQuantityTypeIdentifierRunningGroundContactTime":   "running_ground_contact_time",
	"HKQuantityTypeIdentifierRunningVerticalOscillation": "running_vertical_oscillation",

	// Other quantity types
	"HKQuantityTypeIdentifierAppleSleepingWristTemperature": "wrist_temperature",
	"HKQuantityTypeIdentifierTimeInDaylight":                "time_in_daylight",
	"HKQuantityTypeIdentifierDietaryWater":                  "dietary_water",
	"HKQuantityTypeIdentifierPhysicalEffort":                "physical_effort",
	"HKQuantityTypeIdentifierWalkingHeartRateAverage":       "walking_heart_rate",

	// Category types (no unit attribute)
	"HKCategoryTypeIdentifierMindfulSession": "mindful_sessions",
	"HKCategoryTypeIdentifierAppleStandHour": "stand_hours",
}

// noUnitTables is the set of tables whose records have no unit attribute.
// These use a 4-column schema (source_name, start_date, end_date, value TEXT).
var noUnitTables = map[string]bool{
	"sleep":            true,
	"mindful_sessions": true,
	"stand_hours":      true,
}

// RecordColumns returns the column names for a given metric table.
func RecordColumns(table string) []string {
	if noUnitTables[table] {
		return []string{"source_name", "start_date", "end_date", "value"}
	}
	return []string{"source_name", "start_date", "end_date", "value", "unit"}
}

// BloodPressureColumns returns the column names for the blood_pressure table.
func BloodPressureColumns() []string {
	return []string{"source_name", "start_date", "end_date", "systolic", "diastolic", "unit"}
}

// WorkoutColumns returns the column names for the workouts table.
func WorkoutColumns() []string {
	return []string{
		"activity_type", "source_name", "start_date", "end_date",
		"duration", "duration_unit",
		"total_distance", "total_distance_unit",
		"total_energy_burned", "total_energy_burned_unit",
	}
}
