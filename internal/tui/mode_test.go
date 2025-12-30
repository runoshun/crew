package tui

import "testing"

func TestMode_String(t *testing.T) {
	tests := []struct {
		mode Mode
		want string
	}{
		{ModeNormal, "normal"},
		{ModeFilter, "filter"},
		{ModeConfirm, "confirm"},
		{ModeInputTitle, "input_title"},
		{ModeInputDesc, "input_desc"},
		{ModeStart, "start"},
		{ModeHelp, "help"},
		{ModeDetail, "detail"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.mode.String(); got != tt.want {
				t.Errorf("Mode.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMode_IsInputMode(t *testing.T) {
	tests := []struct {
		mode Mode
		want bool
	}{
		{ModeNormal, false},
		{ModeFilter, true},
		{ModeConfirm, false},
		{ModeInputTitle, true},
		{ModeInputDesc, true},
		{ModeStart, false},
		{ModeHelp, false},
		{ModeDetail, false},
	}

	for _, tt := range tests {
		t.Run(tt.mode.String(), func(t *testing.T) {
			if got := tt.mode.IsInputMode(); got != tt.want {
				t.Errorf("Mode.IsInputMode() = %v, want %v", got, tt.want)
			}
		})
	}
}
