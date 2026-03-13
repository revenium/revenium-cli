package output

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilterFields_Map(t *testing.T) {
	data := map[string]interface{}{
		"id":     "abc",
		"name":   "Test",
		"status": "active",
		"type":   "API",
	}

	result := FilterFields(data, []string{"id", "name"})
	m := result.(map[string]interface{})
	assert.Len(t, m, 2)
	assert.Equal(t, "abc", m["id"])
	assert.Equal(t, "Test", m["name"])
}

func TestFilterFields_Slice(t *testing.T) {
	data := []map[string]interface{}{
		{"id": "1", "name": "First", "extra": "x"},
		{"id": "2", "name": "Second", "extra": "y"},
	}

	result := FilterFields(data, []string{"id", "name"})
	items := result.([]map[string]interface{})
	assert.Len(t, items, 2)
	assert.Len(t, items[0], 2)
	assert.Equal(t, "1", items[0]["id"])
}

func TestFilterFields_EmptyFields(t *testing.T) {
	data := map[string]interface{}{"id": "abc"}
	result := FilterFields(data, nil)
	assert.Equal(t, data, result)
}

func TestFilterTableDef(t *testing.T) {
	def := TableDef{
		Headers:      []string{"ID", "Name", "Status", "Type"},
		StatusColumn: 2,
	}
	rows := [][]string{
		{"1", "First", "active", "API"},
		{"2", "Second", "inactive", "AI"},
	}

	newDef, newRows := FilterTableDef(def, rows, []string{"id", "status"})
	assert.Equal(t, []string{"ID", "Status"}, newDef.Headers)
	assert.Equal(t, 1, newDef.StatusColumn)
	assert.Equal(t, [][]string{{"1", "active"}, {"2", "inactive"}}, newRows)
}

func TestFilterTableDef_EmptyFields(t *testing.T) {
	def := TableDef{Headers: []string{"ID"}, StatusColumn: -1}
	rows := [][]string{{"1"}}
	newDef, newRows := FilterTableDef(def, rows, nil)
	assert.Equal(t, def, newDef)
	assert.Equal(t, rows, newRows)
}
