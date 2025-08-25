package version

import "testing"

func TestGetVersionForAstarteComponent(t *testing.T) {
	tests := []struct {
		name             string
		astarteVersion   string
		componentVersion string
		want             string
	}{
		{
			name:             "ComponentVersion Provided",
			astarteVersion:   "1.0.0",
			componentVersion: "2.0.0",
			want:             "2.0.0",
		},
		{
			name:             "ComponentVersion Empty",
			astarteVersion:   "1.0.0",
			componentVersion: "",
			want:             "1.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetVersionForAstarteComponent(tt.astarteVersion, tt.componentVersion)
			if got != tt.want {
				t.Errorf("GetVersionForAstarteComponent() = %v, want %v", got, tt.want)
			}
		})
	}
}
