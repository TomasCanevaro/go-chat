package services

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateMessageContent(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		want        string
		expectedErr error
	}{
		{
			name:  "valid message",
			input: "Hello Bob!",
			want:  "Hello Bob!",
		},
		{
			name:  "trim whitespace",
			input: "   Hello Bob!   ",
			want:  "Hello Bob!",
		},
		{
			name:        "empty message",
			input:       "",
			expectedErr: ErrEmptyMessage,
		},
		{
			name:        "whitespace only",
			input:       "   \n\t   ",
			expectedErr: ErrEmptyMessage,
		},
		{
			name:  "unicode characters",
			input: "Hola! 👋 ¿Cómo estás?",
			want:  "Hola! 👋 ¿Cómo estás?",
		},
		{
			name:  "exactly 4000 characters",
			input: strings.Repeat("a", 4000),
			want:  strings.Repeat("a", 4000),
		},
		{
			name:        "more than 4000 characters",
			input:       strings.Repeat("a", 4001),
			expectedErr: ErrMessageTooLong,
		},
		{
			name:  "4000 unicode characters",
			input: strings.Repeat("🚀", 4000),
			want:  strings.Repeat("🚀", 4000),
		},
		{
			name:        "4001 unicode characters",
			input:       strings.Repeat("🚀", 4001),
			expectedErr: ErrMessageTooLong,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateMessageContent(tt.input)

			if !errors.Is(err, tt.expectedErr) {
				t.Fatalf(
					"Expected error %v, got %v",
					tt.expectedErr,
					err,
				)
			}

			if got != tt.want {
				t.Errorf(
					"Expected content %q, got %q",
					tt.want,
					got,
				)
			}
		})
	}
}
