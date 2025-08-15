package services

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"
	"math/rand"
	"regexp"
	"strings"
	"time"

	"github.com/gorilla/sessions"
)

// WebResponderService handles IVR functionality
type WebResponderService struct {
	store *sessions.CookieStore
}

// NewWebResponderService creates a new Web Responder service
func NewWebResponderService(sessionSecret string) *WebResponderService {
	return &WebResponderService{
		store: sessions.NewCookieStore([]byte(sessionSecret)),
	}
}

// XML Response structures for NetSapiens
type Response struct {
	XMLName xml.Name `xml:"Response"`
	Actions []interface{}
}

type Say struct {
	XMLName  xml.Name `xml:"Say"`
	Voice    string   `xml:"voice,attr,omitempty"`
	Language string   `xml:"language,attr,omitempty"`
	Text     string   `xml:",chardata"`
}

type Gather struct {
	XMLName   xml.Name `xml:"Gather"`
	NumDigits string   `xml:"numDigits,attr"`
	Action    string   `xml:"action,attr"`
	Timeout   string   `xml:"timeout,attr,omitempty"`
	Actions   []interface{}
}

type Wait struct {
	XMLName xml.Name `xml:"Wait"`
	Timeout string   `xml:"timeout,attr"`
}

type Hangup struct {
	XMLName xml.Name `xml:"Hangup"`
}

// Location data structure
type Location struct {
	City     string  `json:"city"`
	State    string  `json:"state"`
	Lat      float64 `json:"lat"`
	Lon      float64 `json:"lon"`
	Timezone string  `json:"timezone"`
}

// WeatherData structure
type WeatherData struct {
	Temperature int `json:"temperature"`
	AQI         int `json:"aqi"`
}

// Area code database - expand as needed
var areaCodes = map[string]Location{
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

// ExtractAreaCode extracts area code from phone number
func (wr *WebResponderService) ExtractAreaCode(phoneNumber string) string {
	// Remove all non-digits
	re := regexp.MustCompile(`[^0-9]`)
	cleaned := re.ReplaceAllString(phoneNumber, "")

	// Remove country code if present
	if len(cleaned) == 11 && strings.HasPrefix(cleaned, "1") {
		cleaned = cleaned[1:]
	}

	// Extract first 3 digits as area code
	if len(cleaned) >= 10 {
		return cleaned[:3]
	}

	return ""
}

// GetLocationFromAreaCode looks up location by area code
func (wr *WebResponderService) GetLocationFromAreaCode(areaCode string) (Location, bool) {
	location, exists := areaCodes[areaCode]
	return location, exists
}

// GetWeatherData fetches weather for location (simulated for now)
func (wr *WebResponderService) GetWeatherData(lat, lon float64) WeatherData {
	// TODO: Replace with actual weather API call
	// For now, return simulated data
	rand.Seed(time.Now().UnixNano())
	return WeatherData{
		Temperature: rand.Intn(40) + 45,  // 45-85°F
		AQI:         rand.Intn(130) + 20, // 20-150
	}
}

// GetLocalTime returns local time for timezone
func (wr *WebResponderService) GetLocalTime(timezone string) string {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		log.Printf("Error loading timezone %s: %v", timezone, err)
		return "unknown"
	}

	now := time.Now().In(loc)
	return now.Format("3:04 PM")
}

// GetAQIDescription returns human-readable AQI description
func (wr *WebResponderService) GetAQIDescription(aqi int) string {
	switch {
	case aqi <= 50:
		return "Good. Air quality is satisfactory."
	case aqi <= 100:
		return "Moderate. Air quality is acceptable for most people."
	case aqi <= 150:
		return "Unhealthy for sensitive groups."
	default:
		return "Unhealthy. Everyone may experience health effects."
	}
}

// GenerateXMLResponse converts Response struct to XML string
func (wr *WebResponderService) GenerateXMLResponse(response Response) string {
	output, err := xml.MarshalIndent(response, "", "  ")
	if err != nil {
		log.Printf("Error marshaling XML: %v", err)
		return ""
	}
	return xml.Header + string(output)
}

// ProcessWeatherIVR processes the main weather IVR logic
func (wr *WebResponderService) ProcessWeatherIVR(session *sessions.Session, callerNumber string, digits string) (string, error) {
	// First call - no digits pressed
	if digits == "" {
		log.Printf("[WR] New call from: %s", callerNumber)

		areaCode := wr.ExtractAreaCode(callerNumber)
		if areaCode == "" {
			log.Printf("[WR] Could not extract area code from: %s", callerNumber)
			response := Response{
				Actions: []interface{}{
					Say{
						Voice:    "female",
						Language: "en-US",
						Text:     "I'm sorry, I couldn't identify your area code. Please try calling from a valid US phone number. Goodbye!",
					},
					Hangup{},
				},
			}
			return wr.GenerateXMLResponse(response), nil
		}

		location, exists := wr.GetLocationFromAreaCode(areaCode)
		if !exists {
			log.Printf("[WR] Area code not found: %s", areaCode)
			response := Response{
				Actions: []interface{}{
					Say{
						Voice:    "female",
						Language: "en-US",
						Text:     fmt.Sprintf("I'm sorry, I couldn't identify the location for area code %s. This service may not be available for your area yet. Goodbye!", areaCode),
					},
					Hangup{},
				},
			}
			return wr.GenerateXMLResponse(response), nil
		}

		log.Printf("[WR] Location identified: %s, %s", location.City, location.State)

		// Store location in session
		locationJSON, _ := json.Marshal(location)
		session.Values["location_json"] = string(locationJSON)
		session.Values["area_code"] = areaCode

		// Build welcome message with menu
		cityState := fmt.Sprintf("%s, %s", location.City, location.State)

		gatherAction := Gather{
			NumDigits: "1",
			Action:    "/wr/weather",
			Timeout:   "10",
			Actions: []interface{}{
				Say{
					Voice:    "female",
					Language: "en-US",
					Text:     fmt.Sprintf("For the current local time in %s, press 1. For the current temperature, press 2. For the air quality index, press 3.", location.City),
				},
			},
		}

		response := Response{
			Actions: []interface{}{
				Say{
					Voice:    "female",
					Language: "en-US",
					Text:     fmt.Sprintf("Welcome! I've detected you're calling from area code %s, which covers %s.", areaCode, cityState),
				},
				gatherAction,
				Say{
					Voice:    "female",
					Language: "en-US",
					Text:     "I didn't receive your selection. Goodbye!",
				},
			},
		}

		return wr.GenerateXMLResponse(response), nil
	}

	// Handle menu selection
	log.Printf("[WR] DTMF received: %s", digits)

	locationJSON, ok := session.Values["location_json"].(string)
	if !ok {
		log.Printf("[WR] No location in session")
		response := Response{
			Actions: []interface{}{
				Say{
					Voice:    "female",
					Language: "en-US",
					Text:     "I'm sorry, there was an error processing your request. Please try again.",
				},
				Hangup{},
			},
		}
		return wr.GenerateXMLResponse(response), nil
	}

	var location Location
	json.Unmarshal([]byte(locationJSON), &location)

	var responseText string

	switch digits {
	case "1":
		log.Printf("[WR] User selected: Local Time")
		localTime := wr.GetLocalTime(location.Timezone)
		responseText = fmt.Sprintf("The current time in %s, %s is %s.",
			location.City, location.State, localTime)

	case "2":
		log.Printf("[WR] User selected: Temperature")
		weather := wr.GetWeatherData(location.Lat, location.Lon)
		responseText = fmt.Sprintf("The current temperature in %s, %s is %d degrees Fahrenheit.",
			location.City, location.State, weather.Temperature)

	case "3":
		log.Printf("[WR] User selected: Air Quality")
		weather := wr.GetWeatherData(location.Lat, location.Lon)
		aqiDescription := wr.GetAQIDescription(weather.AQI)
		responseText = fmt.Sprintf("The current Air Quality Index in %s, %s is %d. This is considered %s",
			location.City, location.State, weather.AQI, aqiDescription)

	default:
		log.Printf("[WR] Invalid selection: %s", digits)
		// Re-present menu
		gatherAction := Gather{
			NumDigits: "1",
			Action:    "/wr/weather",
			Timeout:   "10",
			Actions: []interface{}{
				Say{
					Voice:    "female",
					Language: "en-US",
					Text:     fmt.Sprintf("For the current local time in %s, press 1. For the current temperature, press 2. For the air quality index, press 3.", location.City),
				},
			},
		}

		response := Response{
			Actions: []interface{}{
				Say{
					Voice:    "female",
					Language: "en-US",
					Text:     "Invalid selection. Let me repeat the options.",
				},
				gatherAction,
				Say{
					Voice:    "female",
					Language: "en-US",
					Text:     "I didn't receive your selection. Goodbye!",
				},
				Hangup{},
			},
		}

		return wr.GenerateXMLResponse(response), nil
	}

	// Send response for valid selections
	response := Response{
		Actions: []interface{}{
			Say{
				Voice:    "female",
				Language: "en-US",
				Text:     responseText,
			},
			Wait{Timeout: "1"},
			Say{
				Voice:    "female",
				Language: "en-US",
				Text:     "Thank you for calling. Goodbye!",
			},
			Hangup{},
		},
	}

	log.Printf("[WR] Sending response: %s", responseText)
	return wr.GenerateXMLResponse(response), nil
}
