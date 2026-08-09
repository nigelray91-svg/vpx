package api

import "testing"

func TestClampSession(t *testing.T) {
	cases := []struct {
		name    string
		plan    string
		seconds int
		want    int
	}{
		{"zero falls back to the default", "resi_pergb", 0, defaultSessionSeconds},
		{"within range is untouched", "resi_pergb", 3600, 3600},
		{"resi_pergb caps at 6h", "resi_pergb", 999999, 21600},
		{"shared_isp caps at 24h", "shared_isp", 999999, 86400},
		{"mobile_pergb caps at 2h", "mobile_pergb", 10000, 7200},
		{"mobile_pergb has a 60s floor", "mobile_pergb", 30, 60},
		{"resi_unlim has a 60s floor", "resi_unlim", 5, 60},
		{"unknown plans pass through", "dc_unlim", 500000, 500000},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := clampSession(tc.plan, tc.seconds); got != tc.want {
				t.Fatalf("clampSession(%q, %d) = %d, want %d", tc.plan, tc.seconds, got, tc.want)
			}
		})
	}
}
