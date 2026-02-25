package cmd

import "testing"

func TestFormatCommas(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{0, "0"},
		{1, "1"},
		{999, "999"},
		{1000, "1,000"},
		{9999, "9,999"},
		{10000, "10,000"},
		{100000, "100,000"},
		{142830, "142,830"},
		{1000000, "1,000,000"},
		{1234567, "1,234,567"},
		{9999999, "9,999,999"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatCommas(tt.input)
			if got != tt.want {
				t.Errorf("formatCommas(%d) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestQueryTotalSupportedTables(t *testing.T) {
	// Verify the set of --total supported tables
	supported := map[string]bool{
		"steps": true, "active-energy": true, "active_energy": true,
		"basal-energy": true, "basal_energy": true,
	}
	for tbl := range supported {
		if !supported[tbl] {
			t.Errorf("expected %q to be supported for --total", tbl)
		}
	}
	unsupported := []string{"hrv", "heart-rate", "sleep", "blood-pressure", "body-mass", "workouts"}
	for _, tbl := range unsupported {
		if supported[tbl] {
			t.Errorf("expected %q to NOT be supported for --total", tbl)
		}
	}
}
