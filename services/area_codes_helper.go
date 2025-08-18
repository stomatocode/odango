package services

import (
	"fmt"
	"strings"
)

// CompleteAreaCodes - Basic area code database (can be expanded)
var CompleteAreaCodes = map[string]Location{
	"415": {City: "San Francisco", State: "CA", Lat: 37.7749, Lon: -122.4194, Timezone: "America/Los_Angeles"},
	"212": {City: "New York", State: "NY", Lat: 40.7128, Lon: -74.0060, Timezone: "America/New_York"},
	"312": {City: "Chicago", State: "IL", Lat: 41.8781, Lon: -87.6298, Timezone: "America/Chicago"},
	"305": {City: "Miami", State: "FL", Lat: 25.7617, Lon: -80.1918, Timezone: "America/New_York"},
	"206": {City: "Seattle", State: "WA", Lat: 47.6062, Lon: -122.3321, Timezone: "America/Los_Angeles"},
	"617": {City: "Boston", State: "MA", Lat: 42.3601, Lon: -71.0589, Timezone: "America/New_York"},
	"404": {City: "Atlanta", State: "GA", Lat: 33.7490, Lon: -84.3880, Timezone: "America/New_York"},
	"512": {City: "Austin", State: "TX", Lat: 30.2672, Lon: -97.7431, Timezone: "America/Chicago"},
	"602": {City: "Phoenix", State: "AZ", Lat: 33.4484, Lon: -112.0740, Timezone: "America/Phoenix"},
	"702": {City: "Las Vegas", State: "NV", Lat: 36.1699, Lon: -115.1398, Timezone: "America/Los_Angeles"},
}

// GetAreaCodeStats returns statistics about the area code database
func GetAreaCodeStats() map[string]int {
	stats := make(map[string]int)

	usCount := 0
	canadaCount := 0
	territoryCount := 0

	for _, location := range CompleteAreaCodes {
		switch {
		case location.State == "PR" || location.State == "VI" ||
			location.State == "MP" || location.State == "GU" || location.State == "AS":
			territoryCount++
		case location.State == "ON" || location.State == "BC" || location.State == "AB" ||
			location.State == "MB" || location.State == "SK" || location.State == "QC" ||
			location.State == "NB" || location.State == "NS" || location.State == "NL" ||
			location.State == "NT" || location.State == "PE" || location.State == "YT" ||
			location.State == "NU":
			canadaCount++
		default:
			usCount++
		}
	}

	stats["total"] = len(CompleteAreaCodes)
	stats["us"] = usCount
	stats["canada"] = canadaCount
	stats["territories"] = territoryCount

	return stats
}

// GetAreaCodeByState returns all area codes for a given state/province
func GetAreaCodesByState(state string) []string {
	var codes []string
	upperState := strings.ToUpper(state)

	for code, location := range CompleteAreaCodes {
		if strings.ToUpper(location.State) == upperState {
			codes = append(codes, code)
		}
	}

	return codes
}

// GetAreaCodesByCity returns all area codes for a given city
func GetAreaCodesByCity(city string) []string {
	var codes []string
	lowerCity := strings.ToLower(city)

	for code, location := range CompleteAreaCodes {
		if strings.ToLower(location.City) == lowerCity {
			codes = append(codes, code)
		}
	}

	return codes
}

// IsValidAreaCode checks if an area code exists in our database
func IsValidAreaCode(areaCode string) bool {
	_, exists := CompleteAreaCodes[areaCode]
	return exists
}

// GetNearbyAreaCodes returns area codes within approximately 100 miles
func GetNearbyAreaCodes(areaCode string, maxDistance float64) []string {
	var nearby []string

	origin, exists := CompleteAreaCodes[areaCode]
	if !exists {
		return nearby
	}

	for code, location := range CompleteAreaCodes {
		if code == areaCode {
			continue
		}

		// Simple distance calculation (not perfectly accurate but good enough)
		latDiff := origin.Lat - location.Lat
		lonDiff := origin.Lon - location.Lon
		distance := latDiff*latDiff + lonDiff*lonDiff

		// Rough approximation: 1 degree ≈ 69 miles
		if distance < (maxDistance/69)*(maxDistance/69) {
			nearby = append(nearby, code)
		}
	}

	return nearby
}

// GetTimeZoneForAreaCode returns the timezone for a given area code
func GetTimeZoneForAreaCode(areaCode string) (string, error) {
	location, exists := CompleteAreaCodes[areaCode]
	if !exists {
		return "", fmt.Errorf("area code %s not found", areaCode)
	}
	return location.Timezone, nil
}

// GetLocationString returns a formatted location string
func GetLocationString(areaCode string) string {
	location, exists := CompleteAreaCodes[areaCode]
	if !exists {
		return "Unknown Location"
	}

	// Handle special cases
	if location.State == "DC" {
		return fmt.Sprintf("%s, DC", location.City)
	}

	// Canadian provinces
	canadianProvinces := map[string]string{
		"ON": "Ontario",
		"BC": "British Columbia",
		"AB": "Alberta",
		"MB": "Manitoba",
		"SK": "Saskatchewan",
		"QC": "Quebec",
		"NB": "New Brunswick",
		"NS": "Nova Scotia",
		"NL": "Newfoundland",
		"NT": "Northwest Territories",
		"PE": "Prince Edward Island",
		"YT": "Yukon",
		"NU": "Nunavut",
	}

	if fullName, isCanadian := canadianProvinces[location.State]; isCanadian {
		return fmt.Sprintf("%s, %s, Canada", location.City, fullName)
	}

	// US Territories
	territories := map[string]string{
		"PR": "Puerto Rico",
		"VI": "US Virgin Islands",
		"MP": "Northern Mariana Islands",
		"GU": "Guam",
		"AS": "American Samoa",
	}

	if fullName, isTerritory := territories[location.State]; isTerritory {
		return fmt.Sprintf("%s, %s", location.City, fullName)
	}

	// Regular US states
	return fmt.Sprintf("%s, %s", location.City, location.State)
}

// GetCountryForAreaCode returns the country for a given area code
func GetCountryForAreaCode(areaCode string) string {
	location, exists := CompleteAreaCodes[areaCode]
	if !exists {
		return "Unknown"
	}

	// Check if it's Canada
	canadianProvinces := []string{"ON", "BC", "AB", "MB", "SK", "QC", "NB", "NS", "NL", "NT", "PE", "YT", "NU"}
	for _, province := range canadianProvinces {
		if location.State == province {
			return "Canada"
		}
	}

	// Check if it's a US territory
	territories := []string{"PR", "VI", "MP", "GU", "AS"}
	for _, territory := range territories {
		if location.State == territory {
			return "US Territory"
		}
	}

	// Default to USA
	return "USA"
}
