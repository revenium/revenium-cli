package output

import (
	"fmt"
	"time"
)

const timestampLayout = "2006-01-02 15:04:05 UTC"

// FormatTimestamp returns any value RFC 3339 cannot parse unchanged: a
// date-only or zone-less value states no zone, so labelling it UTC would
// invent one.
func FormatTimestamp(raw string) string {
	t, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return raw
	}
	return t.UTC().Format(timestampLayout)
}

func TimestampVal(m map[string]interface{}, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	if s, isString := v.(string); isString {
		return FormatTimestamp(s)
	}
	return fmt.Sprint(v)
}
