package output

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderJSON_PrettyPrints(t *testing.T) {
	var buf bytes.Buffer
	f := NewWithWriter(&buf, &bytes.Buffer{}, true, false)

	data := map[string]string{"name": "test", "status": "active"}
	err := f.RenderJSON(data)
	require.NoError(t, err)

	output := buf.String()
	// Should be indented with 2-space indent
	assert.Contains(t, output, "  ")
	assert.Contains(t, output, "\"name\"")
	assert.Contains(t, output, "\"test\"")
}

func TestRenderJSON_Array(t *testing.T) {
	var buf bytes.Buffer
	f := NewWithWriter(&buf, &bytes.Buffer{}, true, false)

	data := []map[string]string{
		{"id": "1", "name": "first"},
		{"id": "2", "name": "second"},
	}
	err := f.RenderJSON(data)
	require.NoError(t, err)

	// Parse output to verify it's a JSON array
	var result []map[string]string
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "first", result[0]["name"])
	assert.Equal(t, "second", result[1]["name"])
}

func TestRenderJSON_SingleObject(t *testing.T) {
	var buf bytes.Buffer
	f := NewWithWriter(&buf, &bytes.Buffer{}, true, false)

	type Source struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	data := Source{ID: "abc-123", Name: "My Source"}
	err := f.RenderJSON(data)
	require.NoError(t, err)

	var result map[string]string
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "abc-123", result["id"])
	assert.Equal(t, "My Source", result["name"])
}

func TestRenderJSON_QuietStillOutputs(t *testing.T) {
	var buf bytes.Buffer
	f := NewWithWriter(&buf, &bytes.Buffer{}, true, true) // jsonMode=true, quiet=true

	data := map[string]string{"key": "value"}
	err := f.RenderJSON(data)
	require.NoError(t, err)

	assert.NotEmpty(t, buf.String(), "JSON output should still appear when quiet+jsonMode")
	var result map[string]string
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "value", result["key"])
}

func TestRenderJSONError_Shape(t *testing.T) {
	var errBuf bytes.Buffer
	f := NewWithWriter(&bytes.Buffer{}, &errBuf, true, false)

	err := f.RenderJSONError("Invalid API key", 401, 2)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(errBuf.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "Invalid API key", result["error"])
	assert.Equal(t, float64(401), result["status"])
	assert.Equal(t, float64(2), result["exit_code"])
}

func TestRenderJSONError_NoStatus(t *testing.T) {
	var errBuf bytes.Buffer
	f := NewWithWriter(&bytes.Buffer{}, &errBuf, true, false)

	err := f.RenderJSONError("Something went wrong", 0, 1)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(errBuf.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "Something went wrong", result["error"])
	assert.Equal(t, float64(0), result["status"])
	assert.Equal(t, float64(1), result["exit_code"])
}

func TestRenderJSON_FormatterDecision(t *testing.T) {
	var buf bytes.Buffer
	f := NewWithWriter(&buf, &bytes.Buffer{}, true, false)

	data := map[string]string{"id": "1"}
	def := TableDef{Headers: []string{"ID"}, StatusColumn: -1}
	rows := [][]string{{"1"}}

	// Render should route to JSON when jsonMode=true
	err := f.Render(def, rows, data)
	require.NoError(t, err)

	// Output should be JSON, not a table
	var result map[string]string
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "1", result["id"])
}
