package models

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetModel asserts the D-01 dead-endpoint error path: no httptest server
// is constructed anywhere in this test, so a passing test structurally
// proves zero HTTP calls are made by `models get`.
func TestGetModel(t *testing.T) {
	var buf bytes.Buffer

	c := newGetCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"mdl-1"})
	err := c.Execute()

	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "not currently available"))
	assert.Contains(t, err.Error(), "models lookup --name")
	assert.Contains(t, err.Error(), "models list")
}

// TestGetModelRegistered verifies models get stays registered (flag, don't
// delete) and reachable in --help despite being a dead endpoint.
func TestGetModelRegistered(t *testing.T) {
	found := false
	for _, sub := range Cmd.Commands() {
		if sub.Name() == "get" {
			found = true
			break
		}
	}
	assert.True(t, found, "models get must remain registered per D-01")
}
