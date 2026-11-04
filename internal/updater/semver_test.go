package updater

import "testing"

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"0.7.0", "0.6.0", 1},
		{"v0.7.0", "0.7.0", 0},
		{"0.8.0-beta", "0.7.0", 1},
		{"0.8.0", "0.8.0-beta", 1},
		{"0.8.0-beta.1", "0.8.0-beta", 1},
		{"0.8.0-beta.38", "0.8.0-beta.1", 1},
		{"0.8.0", "0.8.0-beta.38", 1},
		{"0.8.0-beta.2", "0.8.0-beta.1", 1},
		{"0.8.0-rc.1", "0.8.0-beta.9", 1},
		{"snapshot", "0.7.0", 0},
		{"2026-11-04", "0.7.0", 0},
	}
	for _, tc := range cases {
		if got := CompareVersions(tc.a, tc.b); got != tc.want {
			t.Errorf("CompareVersions(%q, %q) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestParseVersionRejectsNonSemver(t *testing.T) {
	for _, raw := range []string{"vsnapshot", "2026-11-04", "", "latest"} {
		if _, ok := ParseVersion(raw); ok {
			t.Errorf("ParseVersion(%q) should fail", raw)
		}
	}
}
