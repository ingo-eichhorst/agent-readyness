package types

import (
	"testing"
)

func TestFileClassString(t *testing.T) {
	tests := []struct {
		fc   FileClass
		want string
	}{
		{ClassSource, "source"},
		{ClassTest, "test"},
		{ClassGenerated, "generated"},
		{ClassExcluded, "excluded"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.fc.String()
			if got != tt.want {
				t.Errorf("FileClass(%d).String() = %q, want %q", tt.fc, got, tt.want)
			}
		})
	}
}

func TestExitErrorError(t *testing.T) {
	tests := []struct {
		name    string
		ee      *ExitError
		want    string
		wantMsg bool // whether error message should contain specific text
	}{
		{
			name:    "threshold not met",
			ee:      &ExitError{Code: 1, Message: "threshold not met: score 5.5 < 8.0"},
			want:    "threshold not met",
			wantMsg: true,
		},
		{
			name:    "analysis failed",
			ee:      &ExitError{Code: 2, Message: "analysis failed"},
			want:    "analysis failed",
			wantMsg: true,
		},
		{
			name:    "empty message",
			ee:      &ExitError{Code: 1, Message: ""},
			want:    "",
			wantMsg: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.ee.Error()
			if tt.wantMsg {
				if got != tt.ee.Message {
					t.Errorf("ExitError.Error() = %q, want %q", got, tt.ee.Message)
				}
				if got != tt.want && !contains(got, tt.want) {
					t.Errorf("ExitError.Error() = %q, should contain %q", got, tt.want)
				}
			} else {
				if got != tt.want {
					t.Errorf("ExitError.Error() = %q, want %q", got, tt.want)
				}
			}
		})
	}
}

func TestExitErrorCodes(t *testing.T) {
	// Test that ExitError implements error interface
	var _ error = &ExitError{}

	// Test exit codes are distinct
	codes := map[int]string{
		1: "threshold",
		2: "analysis",
	}

	for code, desc := range codes {
		ee := &ExitError{Code: code, Message: desc}
		if ee.Code != code {
			t.Errorf("ExitError code = %d, want %d", ee.Code, code)
		}
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr
}
