package models

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestFlexibleCDR_UnmarshalJSON(t *testing.T) {
	// Test data mimicking NetSapiens response
	jsonData := `{
        "id": "test-123",
        "domain": "example.com",
        "call-orig-caller-id": "5551234567",
        "call-term-caller-id": "5559876543",
        "call-duration": 120,
        "call-start-datetime": "2024-01-15T10:30:00Z"
    }`

	var cdr FlexibleCDR
	err := json.Unmarshal([]byte(jsonData), &cdr)

	if err != nil {
		t.Fatalf("Failed to unmarshal CDR: %v", err)
	}

	// Test field access
	if cdr.GetID() != "test-123" {
		t.Errorf("Expected ID 'test-123', got '%s'", cdr.GetID())
	}

	if cdr.GetDomain() != "example.com" {
		t.Errorf("Expected domain 'example.com', got '%s'", cdr.GetDomain())
	}

	if cdr.GetCallDuration() != 120 {
		t.Errorf("Expected duration 120, got %d", cdr.GetCallDuration())
	}

	// Test that we have the expected number of fields
	if len(cdr.DetectedFields) != 6 {
		t.Errorf("Expected 6 detected fields, got %d", len(cdr.DetectedFields))
	}

	// Test MarshalJSON
	marshaled, err := json.Marshal(cdr)
	if err != nil {
		t.Fatalf("Failed to marshal CDR: %v", err)
	}

	// Should contain original data
	marshaledStr := string(marshaled)
	if !strings.Contains(marshaledStr, "test-123") {
		t.Errorf("Marshaled JSON should contain CDR ID: %s", marshaledStr)
	}

	if !strings.Contains(marshaledStr, "example.com") {
		t.Errorf("Marshaled JSON should contain domain: %s", marshaledStr)
	}
}

func TestFlexibleCDR_EmptyData(t *testing.T) {
	var cdr FlexibleCDR

	// Test with nil RawData
	if cdr.GetString("any-field") != "" {
		t.Error("Expected empty string for nil RawData")
	}

	if cdr.GetInt("any-field") != 0 {
		t.Error("Expected 0 for nil RawData")
	}

	if cdr.HasField("any-field") {
		t.Error("Expected false for HasField with nil RawData")
	}
}

func TestFlexibleCDR_MarshalJSON(t *testing.T) {
	// Test that MarshalJSON works correctly
	cdr := FlexibleCDR{
		RawData: map[string]interface{}{
			"test-field": "test-value",
			"number":     42,
		},
	}

	marshaled, err := json.Marshal(cdr)
	if err != nil {
		t.Fatalf("Failed to marshal CDR: %v", err)
	}

	// Unmarshal to verify structure
	var result map[string]interface{}
	err = json.Unmarshal(marshaled, &result)
	if err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	if result["test-field"] != "test-value" {
		t.Errorf("Expected 'test-value', got %v", result["test-field"])
	}

	// JSON numbers unmarshal as float64
	if result["number"] != float64(42) {
		t.Errorf("Expected 42, got %v", result["number"])
	}
}

func TestFlexibleCDR_TypeConversions(t *testing.T) {
	cdr := FlexibleCDR{
		RawData: map[string]interface{}{
			"string-field": "hello",
			"int-field":    float64(42), // JSON numbers are float64
			"bool-true":    true,
			"bool-string":  "true",
			"float-field":  3.14,
		},
	}

	// Test string conversion
	if cdr.GetString("int-field") != "42" {
		t.Errorf("Expected '42', got '%s'", cdr.GetString("int-field"))
	}

	// Test int conversion
	if cdr.GetInt("int-field") != 42 {
		t.Errorf("Expected 42, got %d", cdr.GetInt("int-field"))
	}

	// Test bool conversions
	if !cdr.GetBool("bool-true") {
		t.Error("Expected true for bool-true")
	}

	if !cdr.GetBool("bool-string") {
		t.Error("Expected true for bool-string")
	}

	// Test float conversion
	if cdr.GetFloat("float-field") != 3.14 {
		t.Errorf("Expected 3.14, got %f", cdr.GetFloat("float-field"))
	}
}
