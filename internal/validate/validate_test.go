package validate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResourceID_Valid(t *testing.T) {
	valid := []string{
		"abc-123",
		"550e8400-e29b-41d4-a716-446655440000",
		"my-resource",
		"UPPER_CASE",
		"with.dots",
		"123",
	}
	for _, id := range valid {
		assert.NoError(t, ResourceID(id), "expected %q to be valid", id)
	}
}

func TestResourceID_Empty(t *testing.T) {
	assert.EqualError(t, ResourceID(""), "resource ID must not be empty")
}

func TestResourceID_ControlChars(t *testing.T) {
	assert.Error(t, ResourceID("abc\x00def"))
	assert.Error(t, ResourceID("abc\ndef"))
	assert.Error(t, ResourceID("abc\tdef"))
	assert.Error(t, ResourceID("\x7f"))
}

func TestResourceID_QueryParams(t *testing.T) {
	assert.Error(t, ResourceID("id?foo=bar"))
	assert.Error(t, ResourceID("id&extra"))
	assert.Error(t, ResourceID("id#fragment"))
}

func TestResourceID_PathTraversal(t *testing.T) {
	assert.Error(t, ResourceID("../etc/passwd"))
	assert.Error(t, ResourceID("..\\windows"))
	assert.NoError(t, ResourceID("not..traversal"))
}

func TestResourceID_PercentEncoded(t *testing.T) {
	assert.Error(t, ResourceID("id%20space"))
	assert.Error(t, ResourceID("%2F"))
}
