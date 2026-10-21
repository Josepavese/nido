package hardware

import "testing"

func TestValidateAcceleratorID(t *testing.T) {
	tests := []struct {
		id      string
		wantErr bool
	}{
		{id: "auto"},
		{id: "virtual:gpu"},
		{id: "01:00.0"},
		{id: "0000:01:00.0"},
		{id: "01:00.8", wantErr: true},
		{id: "01:00.0;touch /tmp/x", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			err := ValidateAcceleratorID(tt.id)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateAcceleratorID(%q) error = %v, wantErr %v", tt.id, err, tt.wantErr)
			}
		})
	}
}
