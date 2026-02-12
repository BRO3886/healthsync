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

// Workout represents a single Apple Health Workout element.
type Workout struct {
	ActivityType          string `xml:"workoutActivityType,attr"`
	Duration              string `xml:"duration,attr"`
	DurationUnit          string `xml:"durationUnit,attr"`
	TotalDistance          string `xml:"totalDistance,attr"`
	TotalDistanceUnit     string `xml:"totalDistanceUnit,attr"`
	TotalEnergyBurned     string `xml:"totalEnergyBurned,attr"`
	TotalEnergyBurnedUnit string `xml:"totalEnergyBurnedUnit,attr"`
	SourceName            string `xml:"sourceName,attr"`
	StartDate             string `xml:"startDate,attr"`
	EndDate               string `xml:"endDate,attr"`
}

// TargetRecordTypes is the set of HK types we care about.
var TargetRecordTypes = map[string]string{
	"HKQuantityTypeIdentifierHeartRate":         "heart_rate",
	"HKQuantityTypeIdentifierStepCount":         "steps",
	"HKQuantityTypeIdentifierOxygenSaturation":  "spo2",
	"HKQuantityTypeIdentifierVO2Max":            "vo2_max",
	"HKCategoryTypeIdentifierSleepAnalysis":     "sleep",
}

// RecordColumns returns the column names for a given metric table.
func RecordColumns(table string) []string {
	if table == "sleep" {
		return []string{"source_name", "start_date", "end_date", "value"}
	}
	return []string{"source_name", "start_date", "end_date", "value", "unit"}
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
