package plugin

import "testing"

func TestCheckKomariXVersion(t *testing.T) {
	tests := []struct {
		constraint string
		wantErr    bool
	}{
		{"", false},
		{"0.0.1", false},
		{"v0.0.1", false},
		{">=0.0.1", false},
		{">0.0.1", false}, // satisfied by the legacy plugin API baseline
		{"<1.0.0", false}, // satisfied by the running product version
		{"<=0.0.1", false},
		{">=1.4.0", false},
		{">=1.4.3", false},
		{">1.4.3", true},
		{">=1.5.0", true},
		{">=99.0.0", true},
		{"0.1", true},
		{"1", true},
		{"1.2.3.4", true},
		{"abc", true},
		{">", true},
	}
	for _, tt := range tests {
		err := CheckKomariXVersion(tt.constraint)
		if (err != nil) != tt.wantErr {
			t.Errorf("CheckKomariXVersion(%q) error = %v, wantErr %v", tt.constraint, err, tt.wantErr)
		}
	}
}
