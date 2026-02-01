// ABOUTME: Tests for vault sync helper functions
// ABOUTME: Covers parseTime, fallbackName, and other utility functions
package sync

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestParseTimeEmpty(t *testing.T) {
	result := parseTime("")
	if !result.IsZero() {
		t.Errorf("parseTime empty string should return zero time, got %v", result)
	}
}

func TestParseTimeValid(t *testing.T) {
	result := parseTime("2025-01-15T10:30:00Z")
	expected := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Errorf("parseTime valid RFC3339 = %v, want %v", result, expected)
	}
}

func TestParseTimeInvalid(t *testing.T) {
	result := parseTime("not-a-date")
	if !result.IsZero() {
		t.Errorf("parseTime invalid string should return zero time, got %v", result)
	}
}

func TestParseTimePartialDate(t *testing.T) {
	// RFC3339 requires time component
	result := parseTime("2025-01-15")
	if !result.IsZero() {
		t.Errorf("parseTime partial date should return zero time, got %v", result)
	}
}

func TestParseTimeWithTimezone(t *testing.T) {
	// Test with timezone offset
	result := parseTime("2025-06-15T14:30:00+05:30")
	if result.IsZero() {
		t.Error("expected valid time with timezone offset")
	}

	// Verify UTC conversion
	utc := result.UTC()
	expected := time.Date(2025, 6, 15, 9, 0, 0, 0, time.UTC)
	if !utc.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, utc)
	}
}

func TestFallbackNameNonEmpty(t *testing.T) {
	id := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	result := fallbackName("John Doe", id)
	if result != "John Doe" {
		t.Errorf("fallbackName non-empty = %q, want %q", result, "John Doe")
	}
}

func TestFallbackNameEmpty(t *testing.T) {
	id := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	result := fallbackName("", id)
	expected := "123e4567-e89b-12d3-a456-426614174000"
	if result != expected {
		t.Errorf("fallbackName empty = %q, want %q", result, expected)
	}
}

func TestFallbackNameWhitespaceOnly(t *testing.T) {
	id := uuid.MustParse("abc12345-1234-1234-1234-123456789abc")
	result := fallbackName("   ", id)
	expected := "abc12345-1234-1234-1234-123456789abc"
	if result != expected {
		t.Errorf("fallbackName whitespace = %q, want %q", result, expected)
	}
}

func TestFallbackNameWithTrimming(t *testing.T) {
	id := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	result := fallbackName("  Jane Smith  ", id)
	if result != "Jane Smith" {
		t.Errorf("fallbackName with whitespace = %q, want %q", result, "Jane Smith")
	}
}
