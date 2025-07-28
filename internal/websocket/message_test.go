package websocket

import (
	"encoding/json"
	"testing"
	"time"
)

func TestMessageUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		jsonData string
		expected time.Time
	}{
		{
			name:     "Unix timestamp in milliseconds as string",
			jsonData: `{"type":"test","data":{},"timestamp":"1753104374613"}`,
			expected: time.Unix(0, 1753104374613*int64(time.Millisecond)),
		},
		{
			name:     "Unix timestamp in seconds as string",
			jsonData: `{"type":"test","data":{},"timestamp":"1753104374"}`,
			expected: time.Unix(1753104374, 0),
		},
		{
			name:     "Unix timestamp as number",
			jsonData: `{"type":"test","data":{},"timestamp":1753104374613}`,
			expected: time.Unix(0, 1753104374613*int64(time.Millisecond)),
		},
		{
			name:     "RFC3339 timestamp string",
			jsonData: `{"type":"test","data":{},"timestamp":"2025-07-21T09:26:14.613Z"}`,
			expected: time.Date(2025, 7, 21, 9, 26, 14, 613000000, time.UTC),
		},
		{
			name:     "No timestamp field",
			jsonData: `{"type":"test","data":{}}`,
			// Will be current time, so we'll check that it's recent
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var msg Message
			err := json.Unmarshal([]byte(tt.jsonData), &msg)
			if err != nil {
				t.Fatalf("Failed to unmarshal JSON: %v", err)
			}

			if msg.Type != "test" {
				t.Errorf("Expected type 'test', got '%s'", msg.Type)
			}

			if tt.name == "No timestamp field" {
				// Check that timestamp is recent (within last minute)
				if time.Since(msg.Timestamp) > time.Minute {
					t.Errorf("Expected recent timestamp, got %v", msg.Timestamp)
				}
			} else {
				expectedUTC := tt.expected.UTC()
				actualUTC := msg.Timestamp.UTC()

				// Allow for small differences due to precision
				diff := actualUTC.Sub(expectedUTC)
				if diff < 0 {
					diff = -diff
				}
				if diff > time.Millisecond {
					t.Errorf("Expected timestamp %v, got %v (diff: %v)", expectedUTC, actualUTC, diff)
				}
			}
		})
	}
}

func TestParseTimestamp(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected time.Time
	}{
		{
			name:  "Nil input",
			input: nil,
			// Will be current time
		},
		{
			name:     "String Unix timestamp milliseconds",
			input:    "1753104374613",
			expected: time.Unix(0, 1753104374613*int64(time.Millisecond)),
		},
		{
			name:     "String Unix timestamp seconds",
			input:    "1753104374",
			expected: time.Unix(1753104374, 0),
		},
		{
			name:     "Float64 timestamp",
			input:    float64(1753104374613),
			expected: time.Unix(0, 1753104374613*int64(time.Millisecond)),
		},
		{
			name:     "Int64 timestamp",
			input:    int64(1753104374613),
			expected: time.Unix(0, 1753104374613*int64(time.Millisecond)),
		},
		{
			name:     "Int timestamp",
			input:    int(1753104374),
			expected: time.Unix(1753104374, 0),
		},
		{
			name:     "RFC3339 string",
			input:    "2025-07-21T09:26:14.613Z",
			expected: time.Date(2025, 7, 21, 9, 26, 14, 613000000, time.UTC),
		},
		{
			name:  "Invalid string",
			input: "invalid",
			// Will be current time
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseTimestamp(tt.input)

			if tt.name == "Nil input" || tt.name == "Invalid string" {
				// Check that timestamp is recent (within last minute)
				if time.Since(result) > time.Minute {
					t.Errorf("Expected recent timestamp, got %v", result)
				}
			} else {
				expectedUTC := tt.expected.UTC()
				actualUTC := result.UTC()

				// Allow for small differences due to precision
				diff := actualUTC.Sub(expectedUTC)
				if diff < 0 {
					diff = -diff
				}
				if diff > time.Millisecond {
					t.Errorf("Expected timestamp %v, got %v (diff: %v)", expectedUTC, actualUTC, diff)
				}
			}
		})
	}
}
