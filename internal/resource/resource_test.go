package resource

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfirmDeleteSkipConfirm(t *testing.T) {
	ok, err := ConfirmDelete("source", "abc-123", true, false)
	require.NoError(t, err)
	assert.True(t, ok, "ConfirmDelete should return true when skipConfirm is true")
}

func TestConfirmDeleteJSONMode(t *testing.T) {
	ok, err := ConfirmDelete("source", "abc-123", false, true)
	require.NoError(t, err)
	assert.True(t, ok, "ConfirmDelete should return true when jsonMode is true")
}

func TestConfirmDeleteBothFlags(t *testing.T) {
	ok, err := ConfirmDelete("source", "abc-123", true, true)
	require.NoError(t, err)
	assert.True(t, ok, "ConfirmDelete should return true when both flags are true")
}

func TestConfirmDeleteNonTTY(t *testing.T) {
	// In test environment, stdin is not a TTY, so ConfirmDelete should auto-confirm
	ok, err := ConfirmDelete("source", "abc-123", false, false)
	require.NoError(t, err)
	assert.True(t, ok, "ConfirmDelete should return true when stdin is not a TTY")
}
