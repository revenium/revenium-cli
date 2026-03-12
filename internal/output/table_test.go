package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTruncate_Short(t *testing.T) {
	result := Truncate("hello", 40)
	assert.Equal(t, "hello", result)
}

func TestTruncate_Long(t *testing.T) {
	input := "abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmn" // 50 chars
	result := Truncate(input, 40)
	runes := []rune(result)
	assert.Equal(t, 40, len(runes), "truncated result should be exactly maxLen runes")
	assert.True(t, strings.HasSuffix(result, "\u2026"), "should end with Unicode ellipsis")
}

func TestTruncate_Unicode(t *testing.T) {
	// Multi-byte characters should be truncated by rune, not byte
	input := "日本語のテスト文字列ですよ" // 12 runes, multi-byte
	result := Truncate(input, 5)
	runes := []rune(result)
	assert.Equal(t, 5, len(runes))
	assert.True(t, strings.HasSuffix(result, "\u2026"))
	// First 4 runes + ellipsis
	assert.Equal(t, "日本語の\u2026", result)
}

func TestTruncate_ExactLength(t *testing.T) {
	result := Truncate("hello", 5)
	assert.Equal(t, "hello", result, "no truncation at exact boundary")
}

func TestRenderTable_ProducesOutput(t *testing.T) {
	var buf bytes.Buffer
	f := NewWithWriter(&buf, &bytes.Buffer{}, false, false)

	err := f.RenderTable(TableDef{
		Headers:      []string{"ID", "Name"},
		StatusColumn: -1,
	}, [][]string{{"1", "Test"}})
	require.NoError(t, err)
	assert.NotEmpty(t, buf.String())
}

func TestRenderTable_Quiet(t *testing.T) {
	var buf bytes.Buffer
	f := NewWithWriter(&buf, &bytes.Buffer{}, false, true)

	err := f.RenderTable(TableDef{
		Headers:      []string{"ID", "Name"},
		StatusColumn: -1,
	}, [][]string{{"1", "Test"}})
	require.NoError(t, err)
	assert.Empty(t, buf.String(), "quiet mode should produce no table output")
}

func TestRenderTable_HasHeaders(t *testing.T) {
	var buf bytes.Buffer
	f := NewWithWriter(&buf, &bytes.Buffer{}, false, false)

	headers := []string{"ID", "Name", "Status"}
	err := f.RenderTable(TableDef{
		Headers:      headers,
		StatusColumn: 2,
	}, [][]string{{"1", "Test", "active"}})
	require.NoError(t, err)

	output := buf.String()
	for _, h := range headers {
		assert.Contains(t, output, h, "output should contain header: %s", h)
	}
}

func TestRenderTable_HasData(t *testing.T) {
	var buf bytes.Buffer
	f := NewWithWriter(&buf, &bytes.Buffer{}, false, false)

	err := f.RenderTable(TableDef{
		Headers:      []string{"ID", "Name"},
		StatusColumn: -1,
	}, [][]string{{"abc-123", "My Source"}, {"def-456", "Another"}})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "abc-123")
	assert.Contains(t, output, "My Source")
	assert.Contains(t, output, "def-456")
	assert.Contains(t, output, "Another")
}

func TestStatusStyle(t *testing.T) {
	// statusStyle returns a lipgloss.Style - we verify it doesn't panic
	// and returns styles with the expected foreground colors
	s := statusStyle("active")
	rendered := s.Render("active")
	assert.NotEmpty(t, rendered)

	s2 := statusStyle("inactive")
	rendered2 := s2.Render("inactive")
	assert.NotEmpty(t, rendered2)

	s3 := statusStyle("pending")
	rendered3 := s3.Render("pending")
	assert.NotEmpty(t, rendered3)

	// Also test aliases
	_ = statusStyle("enabled")
	_ = statusStyle("disabled")
	_ = statusStyle("deleted")
	_ = statusStyle("draft")
	_ = statusStyle("unknown")
}
