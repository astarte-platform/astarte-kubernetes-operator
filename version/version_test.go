package version

import (
	"testing"

	apiv1alpha2 "github.com/astarte-platform/astarte-kubernetes-operator/apis/api/v1alpha2"
)

// TestCompareAstarteVersions contains the test suite for the compareAstarteVersions function.
func TestCompareAstarteVersions(t *testing.T) { //nolint:funlen
	// testCases defines the inputs and expected outputs for each test scenario.
	testCases := []struct {
		name    string // The name of the test case
		v1      string // The first version string
		v2      string // The second version string
		want    int    // The expected comparison result: -1, 0, or 1
		wantErr bool   // Whether an error is expected
	}{
		// --- Equality Cases ---
		{
			name: "equal versions",
			v1:   "1.2.3",
			v2:   "1.2.3",
			want: 0,
		},
		{
			name: "equal snapshot versions",
			v1:   "1.2-snapshot",
			v2:   "1.2-snapshot",
			want: 0,
		},
		// --- Standard SemVer Comparison Cases (v1 > v2) ---
		{
			name: "v1 greater on major version",
			v1:   "2.0.0",
			v2:   "1.9.9",
			want: 1,
		},
		{
			name: "v1 greater on minor version",
			v1:   "1.3.0",
			v2:   "1.2.9",
			want: 1,
		},
		{
			name: "v1 greater on patch version",
			v1:   "1.2.4",
			v2:   "1.2.3",
			want: 1,
		},

		// --- Standard SemVer Comparison Cases (v1 < v2) ---
		{
			name: "v2 greater on major version",
			v1:   "1.9.9",
			v2:   "2.0.0",
			want: -1,
		},
		{
			name: "v2 greater on minor version",
			v1:   "1.2.9",
			v2:   "1.3.0",
			want: -1,
		},
		{
			name: "v2 greater on patch version",
			v1:   "1.2.3",
			v2:   "1.2.4",
			want: -1,
		},

		// --- Custom Snapshot Logic Cases (same base version) ---
		{
			name: "snapshot is greater than its base version",
			v1:   "1.2-snapshot",
			v2:   "1.2.3",
			want: 1,
		},
		{
			name: "snapshot is greater than its base version",
			v1:   "1.2.3",
			v2:   "1.2-snapshot",
			want: -1,
		},
		{
			name: "snapshot is greater than its base version",
			v1:   "1.2.99",
			v2:   "1.2-snapshot",
			want: -1,
		},
		{
			name: "base version is less than its snapshot",
			v1:   "1.2.3",
			v2:   "1.2.3-snapshot",
			want: -1,
		},

		// --- Precedence Cases (different base versions) ---
		{
			name: "base version comparison takes precedence over snapshot (v1 greater)",
			v1:   "1.3.0",
			v2:   "1.2-snapshot",
			want: 1,
		},
		{
			name: "base version comparison takes precedence over snapshot (v2 greater)",
			v1:   "1.2.0-snapshot",
			v2:   "1.3.0",
			want: -1,
		},
		{
			name: "both snapshots, v1 base is greater",
			v1:   "1.3.0-snapshot",
			v2:   "1.2-snapshot",
			want: 1,
		},
		{
			name: "both snapshots, v2 base is greater",
			v1:   "1.2-snapshot",
			v2:   "1.3-snapshot",
			want: -1,
		},

		// --- Interaction with Standard Pre-releases ---
		{
			name: "release is greater than standard pre-release",
			v1:   "1.0.0",
			v2:   "1.0.0-beta",
			want: 1,
		},
		{
			name: "snapshot vs standard pre-release (base comparison wins)",
			v1:   "1.0-snapshot", // base becomes "1.0.0"
			v2:   "1.0.0-beta",   // base is "1.0.0-beta"
			want: 1,              // "1.0.0" > "1.0.0-beta", so snapshot logic is not reached
		},

		// --- Error Cases ---
		{
			name:    "invalid v1 string",
			v1:      "not-a-version",
			v2:      "1.2.3",
			wantErr: true,
		},
		{
			name:    "invalid v2 string",
			v1:      "1.2.3",
			v2:      "not-a-version",
			wantErr: true,
		},
		{
			name:    "invalid v1 with snapshot suffix",
			v1:      "1.a.3-snapshot",
			v2:      "1.2.3",
			wantErr: true,
		},
	}

	// Iterate over all test cases.
	for _, tc := range testCases {
		// t.Run enables running each case as a distinct sub-test.
		t.Run(tc.name, func(t *testing.T) {
			got, err := CompareAstarteVersions(tc.v1, tc.v2)

			// Check if an error was expected.
			if tc.wantErr {
				if err == nil {
					t.Errorf("compareAstarteVersions(%q, %q) expected an error, but got none", tc.v1, tc.v2)
				}
				return // End test if error was expected and received.
			}

			// Check if an unexpected error occurred.
			if err != nil {
				t.Fatalf("compareAstarteVersions(%q, %q) returned an unexpected error: %v", tc.v1, tc.v2, err)
			}

			// Check if the comparison result is correct.
			if got != tc.want {
				t.Errorf("compareAstarteVersions(%q, %q) = %d; want %d", tc.v1, tc.v2, got, tc.want)
			}
		})
	}
}

func TestAstarteVersionImplementsErlangClustering(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		wantErr  bool
		wantImpl bool
	}{
		{
			name:     "no version implies true",
			version:  "",
			wantErr:  false,
			wantImpl: true,
		},
		{
			name:     "old version returns false",
			version:  "1.1.0",
			wantErr:  false,
			wantImpl: false,
		},
		{
			name:     "equal version returns true",
			version:  "1.2.1",
			wantErr:  false,
			wantImpl: true,
		},
		{
			name:     "new version returns true",
			version:  "1.2.2",
			wantErr:  false,
			wantImpl: true,
		},
		{
			name:     "compare returns error",
			version:  "invalid",
			wantErr:  true,
			wantImpl: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &apiv1alpha2.Astarte{
				Spec: apiv1alpha2.AstarteSpec{
					Version: tt.version,
				},
			}
			err, implements := AstarteVersionImplementsErlangClustering(r)
			if (err != nil) != tt.wantErr {
				t.Errorf("got err = %v, wantErr %v", err, tt.wantErr)
			}
			if implements != tt.wantImpl {
				t.Errorf("got implements = %v, want %v", implements, tt.wantImpl)
			}
		})
	}
}
