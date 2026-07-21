package logging

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"
)

func TestNewWritesJSONTimestampInUTC(t *testing.T) {
	var output bytes.Buffer
	logger := New(&output)

	logger.Info("test message", "component", "test")

	var entry map[string]any
	if err := json.Unmarshal(output.Bytes(), &entry); err != nil {
		t.Fatalf("log is not valid JSON: %v", err)
	}

	timestamp, err := time.Parse(time.RFC3339Nano, entry["time"].(string))
	if err != nil {
		t.Fatalf("time is not RFC3339: %v", err)
	}
	if timestamp.Location() != time.UTC {
		t.Fatalf("timestamp location = %v, want UTC", timestamp.Location())
	}
	if entry["msg"] != "test message" || entry["component"] != "test" {
		t.Fatalf("unexpected structured log: %v", entry)
	}
}
