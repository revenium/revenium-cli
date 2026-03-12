package errors

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAPIErrorError(t *testing.T) {
	err := &APIError{
		StatusCode: 401,
		Message:    "unauthorized",
		Body:       `{"error":"invalid key"}`,
	}
	require.Equal(t, "unauthorized", err.Error())
}

func TestAPIErrorVerboseError(t *testing.T) {
	err := &APIError{
		StatusCode: 404,
		Message:    "not found",
		Body:       `{"error":"resource missing"}`,
	}
	verbose := err.VerboseError()
	require.Contains(t, verbose, "not found")
	require.Contains(t, verbose, "404")
	require.Contains(t, verbose, `{"error":"resource missing"}`)
}

func TestRenderError(t *testing.T) {
	result := RenderError("something went wrong")
	require.NotEmpty(t, result)
	require.Contains(t, result, "something went wrong")
}
