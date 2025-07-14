package models

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// FlexibleCDR handles any CDR response dynamically
type FlexibleCDR struct {
	RawData        map[string]interface{} `json:"-"`
	DetectedFields []string               `json:"-"`
}

// UnmarshalJSON implements the json.Unmarshaler interface
func (f *FlexibleCDR) UnmarshalJSON(data []byte) error {
	// Unmarshal everything into raw map
	if err := json.Unmarshal(data, &f.RawData); err != nil {
		return err
	}

	// Catalog what fields we actually received
	f.DetectedFields = make([]string, 0, len(f.RawData))
	for key := range f.RawData {
		f.DetectedFields = append(f.DetectedFields, key)
	}

	return nil
}

// String field access with fallback
func (f *FlexibleCDR) GetString(field string) string {
	if val, ok := f.RawData[field]; ok && val != nil {
		if str, ok := val.(string); ok {
			return str
		}
		// Try to convert other types to string
		return fmt.Sprintf("%v", val)
	}
	return ""
}

// Integer field access with fallback
func (f *FlexibleCDR) GetInt(field string) int {
	if val, ok := f.RawData[field]; ok && val != nil {
		switch v := val.(type) {
		case float64: // JSON numbers are float64
			return int(v)
		case int:
			return v
		case string:
			if i, err := strconv.Atoi(v); err == nil {
				return i
			}
		}
	}
	return 0
}

// Integer field access returning int64 for large phone numbers
func (f *FlexibleCDR) GetInt64(field string) int64 {
	if val, ok := f.RawData[field]; ok && val != nil {
		switch v := val.(type) {
		case float64:
			return int64(v)
		case int64:
			return v
		case int:
			return int64(v)
		case string:
			if i, err := strconv.ParseInt(v, 10, 64); err == nil {
				return i
			}
		}
	}
	return 0
}

// Boolean field access
func (f *FlexibleCDR) GetBool(field string) bool {
	if val, ok := f.RawData[field]; ok && val != nil {
		switch v := val.(type) {
		case bool:
			return v
		case float64:
			return v != 0
		case int:
			return v != 0
		case string:
			return v == "true" || v == "1" || v == "yes"
		}
	}
	return false
}

// Float field access for percentages and decimals
func (f *FlexibleCDR) GetFloat(field string) float64 {
	if val, ok := f.RawData[field]; ok && val != nil {
		switch v := val.(type) {
		case float64:
			return v
		case int:
			return float64(v)
		case string:
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				return f
			}
		}
	}
	return 0.0
}

// Time field access for datetime strings
func (f *FlexibleCDR) GetTime(field string) (time.Time, error) {
	timeStr := f.GetString(field)
	if timeStr == "" {
		return time.Time{}, fmt.Errorf("field %s is empty or missing", field)
	}

	// Try common NetSapiens time formats
	formats := []string{
		"2006-01-02T15:04:05Z[MST]", // Your sample format
		"2006-01-02T15:04:05Z",      // Standard ISO
		"2006-01-02 15:04:05",       // MySQL format
		time.RFC3339,                // Standard RFC3339
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timeStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse time %s for field %s", timeStr, field)
}

// Check if a field exists in the response
func (f *FlexibleCDR) HasField(field string) bool {
	_, exists := f.RawData[field]
	return exists
}

// Get all field names that were detected
func (f *FlexibleCDR) GetFieldNames() []string {
	return f.DetectedFields
}

// Get the raw value for a field (returns interface{})
func (f *FlexibleCDR) GetRaw(field string) interface{} {
	return f.RawData[field]
}

// Convenience methods for common CDR fields using your JSON sample field names

func (f *FlexibleCDR) GetID() string {
	// Try modern field name first, fallback to legacy
	if id := f.GetString("id"); id != "" {
		return id
	}
	return f.GetString("cdr_id")
}

func (f *FlexibleCDR) GetDomain() string {
	return f.GetString("domain")
}

func (f *FlexibleCDR) GetCallDirection() int {
	return f.GetInt("call-direction")
}

func (f *FlexibleCDR) GetCallStartTime() (time.Time, error) {
	return f.GetTime("call-start-datetime")
}

func (f *FlexibleCDR) GetCallDuration() int {
	// Try modern field name first
	if duration := f.GetInt("call-total-duration-seconds"); duration > 0 {
		return duration
	}
	return f.GetInt("duration")
}

func (f *FlexibleCDR) GetOrigCallerID() int64 {
	return f.GetInt64("call-orig-caller-id")
}

func (f *FlexibleCDR) GetTermCallerID() int64 {
	return f.GetInt64("call-term-caller-id")
}

func (f *FlexibleCDR) GetOrigUser() string {
	return f.GetString("call-orig-user")
}

func (f *FlexibleCDR) GetTermUser() string {
	return f.GetString("call-term-user")
}

func (f *FlexibleCDR) GetDisconnectReason() string {
	return f.GetString("call-disconnect-reason-text")
}

// Report generation methods

// ToKeyValuePairs returns the CDR as a slice of key-value pairs for simple table display
func (f *FlexibleCDR) ToKeyValuePairs() [][]string {
	pairs := make([][]string, 0, len(f.DetectedFields))

	for _, field := range f.DetectedFields {
		value := fmt.Sprintf("%v", f.RawData[field])
		if value == "<nil>" {
			value = "null"
		}
		pairs = append(pairs, []string{field, value})
	}

	return pairs
}

// ToCallSummary returns essential call information as a table
func (f *FlexibleCDR) ToCallSummary() [][]string {
	startTime, _ := f.GetCallStartTime()

	return [][]string{
		{"Field", "Value"},
		{"Call ID", f.GetID()},
		{"Domain", f.GetDomain()},
		{"Direction", fmt.Sprintf("%d", f.GetCallDirection())},
		{"Start Time", startTime.Format("2006-01-02 15:04:05")},
		{"Duration (seconds)", fmt.Sprintf("%d", f.GetCallDuration())},
		{"Origin User", f.GetOrigUser()},
		{"Term User", f.GetTermUser()},
		{"Disconnect Reason", f.GetDisconnectReason()},
		{"Field Count", fmt.Sprintf("%d", len(f.DetectedFields))},
	}
}

func (f *FlexibleCDR) HasTranscriptionData() bool {
	return f.HasField("call-intelligence-job-id")
}

func (f *FlexibleCDR) HasSentimentData() bool {
	return f.HasField("call-intelligence-percent-positive")
}

func (f *FlexibleCDR) GetAvailableReportFields() []string {
	reportFields := []string{}

	// Essential fields every report should check for
	essentialFields := []string{
		"id", "domain", "call-direction", "call-start-datetime",
		"call-total-duration-seconds", "call-orig-user", "call-term-user",
		"call-disconnect-reason-text", "call-orig-caller-id", "call-term-caller-id",
	}

	for _, field := range essentialFields {
		if f.HasField(field) {
			reportFields = append(reportFields, field)
		}
	}

	return reportFields
}
