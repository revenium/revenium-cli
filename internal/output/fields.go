package output

import "strings"

// FilterFields filters a data object to include only the specified fields.
// Supports map[string]interface{} and []map[string]interface{}.
// Returns the input unchanged for other types or when fields is empty.
func FilterFields(data interface{}, fields []string) interface{} {
	if len(fields) == 0 {
		return data
	}

	fieldSet := make(map[string]bool, len(fields))
	for _, f := range fields {
		fieldSet[strings.TrimSpace(f)] = true
	}

	switch v := data.(type) {
	case map[string]interface{}:
		return filterMap(v, fieldSet)
	case []map[string]interface{}:
		result := make([]map[string]interface{}, len(v))
		for i, item := range v {
			result[i] = filterMap(item, fieldSet)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				result[i] = filterMap(m, fieldSet)
			} else {
				result[i] = item
			}
		}
		return result
	default:
		return data
	}
}

func filterMap(m map[string]interface{}, fields map[string]bool) map[string]interface{} {
	result := make(map[string]interface{}, len(fields))
	for k, v := range m {
		if fields[k] {
			result[k] = v
		}
	}
	return result
}

// FilterTableDef filters table columns to include only those whose headers
// match the specified fields. Returns the filtered TableDef and rows.
// When fields is empty, returns the inputs unchanged.
func FilterTableDef(def TableDef, rows [][]string, fields []string) (TableDef, [][]string) {
	if len(fields) == 0 {
		return def, rows
	}

	fieldSet := make(map[string]bool, len(fields))
	for _, f := range fields {
		fieldSet[strings.TrimSpace(strings.ToLower(f))] = true
	}

	// Find matching column indices
	var indices []int
	var headers []string
	newStatusCol := -1
	for i, h := range def.Headers {
		if fieldSet[strings.ToLower(h)] {
			if i == def.StatusColumn {
				newStatusCol = len(indices)
			}
			indices = append(indices, i)
			headers = append(headers, h)
		}
	}

	if len(indices) == 0 {
		return def, rows
	}

	newDef := TableDef{
		Headers:      headers,
		StatusColumn: newStatusCol,
	}

	newRows := make([][]string, len(rows))
	for i, row := range rows {
		newRow := make([]string, len(indices))
		for j, idx := range indices {
			if idx < len(row) {
				newRow[j] = row[idx]
			}
		}
		newRows[i] = newRow
	}

	return newDef, newRows
}
