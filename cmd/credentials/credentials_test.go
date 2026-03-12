package credentials

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMaskSecret(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", ""},
		{"short string", "short", "****hort"},
		{"exactly 4 chars", "abcd", "****abcd"},
		{"with prefix", "sk-abc123xyz7f3a", "sk-****7f3a"},
		{"no prefix", "abc123xyz7f3a", "****7f3a"},
		{"hyphen at end area", "abcdefgh-ij", "****h-ij"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, maskSecret(tt.input))
		})
	}
}
