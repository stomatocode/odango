// services/cdr_discovery.go
// Updated to include raw=yes parameter for complete CDR data retrieval

package services

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"o-dan-go/models"
	"strings"
	"time" // add for console logging
)

// CDRDiscoveryService handles comprehensive CDR discovery across multiple endpoints
type CDRDiscoveryService struct {
	client      *http.Client
	baseURL     string
	accessToken string
	debug       bool // console logging

}

// CDRSearchCriteria - flexible search criteria, all fields optional
type CDRSearchCriteria struct {
	Domain            string     `json:"domain,omitempty"`
	User              string     `json:"user,omitempty"`
	Site              string     `json:"site,omitempty"` // Site/location filter
	CallID            string     `json:"call_id,omitempty"`
	StartDate         *time.Time `json:"start_date,omitempty"`
	EndDate           *time.Time `json:"end_date,omitempty"`
	Start             int        `json:"start,omitempty"` // Pagination offset
	Limit             int        `json:"limit,omitempty"` // Max records per endpoint
	Raw               bool       `json:"raw,omitempty"`   // Force raw data (always true for bulk dumps)
	OriginatingNumber string     `json:"originating_number"`
	TerminatingNumber string     `json:"terminating_number"`
	AnyPhoneNumber    string     `json:"any_phone_number"`
}

// CDRDiscoveryResult - comprehensive result from all endpoints
type CDRDiscoveryResult struct {
	SessionID       string                          `json:"session_id"`
	SearchCriteria  CDRSearchCriteria               `json:"search_criteria"`
	StartTime       time.Time                       `json:"start_time"`
	EndTime         time.Time                       `json:"end_time"`
	TotalCDRs       int                             `json:"total_cdrs"`
	UniqueCDRs      int                             `json:"unique_cdrs"`
	EndpointResults []EndpointResult                `json:"endpoint_results"`
	AllCDRs         []models.FlexibleCDR            `json:"all_cdrs"`
	CDRsByEndpoint  map[string][]models.FlexibleCDR `json:"cdrs_by_endpoint"`
	Errors          []string                        `json:"errors,omitempty"`
}

// EndpointResult - result from individual endpoint query
type EndpointResult struct {
	EndpointName   string               `json:"endpoint_name"`
	URL            string               `json:"url"`
	RecordCount    int                  `json:"record_count"`
	Success        bool                 `json:"success"`
	Error          string               `json:"error,omitempty"`
	QueryTime      time.Duration        `json:"query_time"`
	HTTPStatus     int                  `json:"http_status"`
	CDRs           []models.FlexibleCDR `json:"cdrs,omitempty"`
	RawDataUsed    bool                 `json:"raw_data_used"`   // Indicates if raw=yes was used
	DiscoveredData bool                 `json:"discovered_data"` //
}

// CDREndpointConfig - configuration for each CDR endpoint
type CDREndpointConfig struct {
	Name           string   `json:"name"`
	URLTemplate    string   `json:"url_template"`
	RequiredParams []string `json:"required_params"`
	OptionalParams []string `json:"optional_params"`
	SupportsRaw    bool     `json:"supports_raw"` // Indicates if endpoint supports raw=yes
	Description    string   `json:"description"`
}

// NewCDRDiscoveryService creates a new CDR discovery service
func NewCDRDiscoveryService(baseURL, accessToken string) *CDRDiscoveryService {
	return &CDRDiscoveryService{
		client:      &http.Client{Timeout: 30 * time.Second},
		baseURL:     strings.TrimRight(baseURL, "/"),
		accessToken: accessToken,
		debug:       true, // console logging
	}
}

// console logging helper method
func (cds *CDRDiscoveryService) logDebug(format string, args ...interface{}) {
	if cds.debug {
		log.Printf("[CDR Discovery] "+format, args...)
	}
}

// GetSupportedEndpoints returns all available CDR endpoints with raw support info
func (cds *CDRDiscoveryService) GetSupportedEndpoints() []CDREndpointConfig {
	return []CDREndpointConfig{
		{
			Name:           "global_cdrs",
			URLTemplate:    "/ns-api/v2/cdrs",
			RequiredParams: []string{},
			OptionalParams: []string{"start", "limit", "raw"},
			SupportsRaw:    true, // Global CDR endpoint supports raw=yes
			Description:    "All CDRs system-wide (supports raw=yes)",
		},
		{
			Name:           "domain_cdrs",
			URLTemplate:    "/ns-api/v2/domains/{domain}/cdrs",
			RequiredParams: []string{"domain"},
			OptionalParams: []string{"start", "limit", "raw"},
			SupportsRaw:    true, // Domain CDR endpoint supports raw=yes
			Description:    "CDRs for specific domain (supports raw=yes)",
		},
		{
			Name:           "user_cdrs",
			URLTemplate:    "/ns-api/v2/domains/{domain}/users/{user}/cdrs",
			RequiredParams: []string{"domain", "user"},
			OptionalParams: []string{"start", "limit", "raw"},
			SupportsRaw:    true, // User CDR endpoint supports raw=yes
			Description:    "CDRs for specific user (supports raw=yes)",
		},
		{
			Name:           "site_cdrs",
			URLTemplate:    "/ns-api/v2/domains/{domain}/sites/{site}/cdrs",
			RequiredParams: []string{"domain", "site"},
			OptionalParams: []string{"start", "limit", "raw"},
			SupportsRaw:    true, // Site CDR endpoint supports raw=yes
			Description:    "CDRs for specific site (supports raw=yes)",
		},
		{
			Name:           "global_count",
			URLTemplate:    "/ns-api/v2/cdrs/count",
			RequiredParams: []string{},
			OptionalParams: []string{},
			SupportsRaw:    false, // Count endpoints typically don't support raw
			Description:    "Count and sum of all CDRs",
		},
		{
			Name:           "domain_count",
			URLTemplate:    "/ns-api/v2/domains/{domain}/cdrs/count",
			RequiredParams: []string{"domain"},
			OptionalParams: []string{},
			SupportsRaw:    false, // Count endpoints typically don't support raw
			Description:    "Count and sum for domain CDRs",
		},
		{
			Name:           "user_count",
			URLTemplate:    "/ns-api/v2/domains/{domain}/users/{user}/cdrs/count",
			RequiredParams: []string{"domain", "user"},
			OptionalParams: []string{},
			SupportsRaw:    false, // Count endpoints typically don't support raw
			Description:    "Count and sum for user CDRs",
		},
	}
}

// GetComprehensiveCDRs - main function to query all relevant endpoints with raw data
func (cds *CDRDiscoveryService) GetComprehensiveCDRs(criteria CDRSearchCriteria) (*CDRDiscoveryResult, error) {
	startTime := time.Now()
	sessionID := cds.generateSessionID()

	// logging
	cds.logDebug("=== NEW CDR SEARCH SESSION STARTED ===")
	cds.logDebug("Session ID: %s", sessionID)
	cds.logDebug("Search Criteria: %+v", criteria)

	// Set default pagination if not provided
	if criteria.Limit == 0 {
		criteria.Limit = 100 // Default limit per endpoint
	}

	// ************************************************************************
	// IMPORTANT: Always force raw=yes for bulk CDR dumps for complete data
	criteria.Raw = true
	cds.logDebug("Raw data mode: ENABLED") // log raw data mode

	result := &CDRDiscoveryResult{
		SessionID:       sessionID,
		SearchCriteria:  criteria,
		StartTime:       startTime,
		EndpointResults: []EndpointResult{},
		CDRsByEndpoint:  make(map[string][]models.FlexibleCDR),
		Errors:          []string{},
	}

	// Determine which endpoints to query based on available criteria
	endpointsToQuery := cds.selectEndpointsToQuery(criteria)
	// logging:
	cds.logDebug("Endpoints selected for query: %d", len(endpointsToQuery))
	for _, ep := range endpointsToQuery {
		cds.logDebug("  - %s: %s", ep.Name, ep.Description)
	}

	// Query each relevant endpoint
	for _, endpointConfig := range endpointsToQuery {
		cds.logDebug("\n--- Querying endpoint: %s ---", endpointConfig.Name) // logging to console

		endpointResult := cds.queryEndpoint(endpointConfig, criteria)
		result.EndpointResults = append(result.EndpointResults, endpointResult)

		// logging block:
		if endpointResult.Success {
			cds.logDebug("✓ SUCCESS: %s", endpointConfig.Name)
			cds.logDebug("  Records found: %d", endpointResult.RecordCount)
			cds.logDebug("  Query time: %v", endpointResult.QueryTime)
			cds.logDebug("  HTTP status: %d", endpointResult.HTTPStatus)

			if len(endpointResult.CDRs) > 0 {
				result.CDRsByEndpoint[endpointConfig.Name] = endpointResult.CDRs
				result.AllCDRs = append(result.AllCDRs, endpointResult.CDRs...)

				// Log sample CDR
				sampleCDR := endpointResult.CDRs[0]
				cds.logDebug("  Sample CDR ID: %s", sampleCDR.GetID())
				cds.logDebug("  Sample Domain: %s", sampleCDR.GetDomain())
			}
		} else {
			cds.logDebug("✗ FAILED: %s", endpointConfig.Name)
			cds.logDebug("  Error: %s", endpointResult.Error)
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %s", endpointConfig.Name, endpointResult.Error))
		}

		if endpointResult.Success && len(endpointResult.CDRs) > 0 {
			result.CDRsByEndpoint[endpointConfig.Name] = endpointResult.CDRs
			result.AllCDRs = append(result.AllCDRs, endpointResult.CDRs...)
		}

		if !endpointResult.Success {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %s", endpointConfig.Name, endpointResult.Error))
		}
	}

	// logging duplication:
	cds.logDebug("\n--- Deduplication ---")
	cds.logDebug("Total CDRs before deduplication: %d", len(result.AllCDRs))

	// Deduplicate CDRs by ID
	result.AllCDRs = cds.deduplicateCDRs(result.AllCDRs)
	result.UniqueCDRs = len(result.AllCDRs)
	result.TotalCDRs = cds.countTotalCDRs(result.CDRsByEndpoint)
	result.EndTime = time.Now()

	// console logging:
	cds.logDebug("Unique CDRs after deduplication: %d", result.UniqueCDRs)
	cds.logDebug("Duplicates removed: %d", result.TotalCDRs-result.UniqueCDRs)

	// Log final summary
	cds.logDebug("\n=== SEARCH SESSION COMPLETED ===")
	cds.logDebug("Session ID: %s", sessionID)
	cds.logDebug("Total execution time: %v", result.EndTime.Sub(result.StartTime))
	cds.logDebug("Total CDRs found: %d", result.TotalCDRs)
	cds.logDebug("Unique CDRs: %d", result.UniqueCDRs)
	cds.logDebug("Endpoints queried: %d", len(result.EndpointResults))
	cds.logDebug("Errors encountered: %d", len(result.Errors))

	// Log CDR distribution by endpoint
	cds.logDebug("\nCDR Distribution by Endpoint:")
	for endpoint, cdrs := range result.CDRsByEndpoint {
		cds.logDebug("  %s: %d CDRs", endpoint, len(cdrs))
	}

	return result, nil
}

// selectEndpointsToQuery determines which endpoints to query based on criteria
func (cds *CDRDiscoveryService) selectEndpointsToQuery(criteria CDRSearchCriteria) []CDREndpointConfig {
	endpoints := cds.GetSupportedEndpoints()
	var selected []CDREndpointConfig

	for _, endpoint := range endpoints {
		// Skip count endpoints for CDR data collection (focus on data endpoints)
		if strings.Contains(endpoint.Name, "count") {
			continue
		}

		// Check if we have required parameters for this endpoint
		if cds.hasRequiredParams(endpoint, criteria) {
			selected = append(selected, endpoint)
		}
	}

	// Always include global CDRs (no required params) if no other endpoints selected
	if len(selected) == 0 {
		for _, endpoint := range endpoints {
			if endpoint.Name == "global_cdrs" {
				selected = append(selected, endpoint)
				break
			}
		}
	}

	return selected
}

// hasRequiredParams checks if criteria contains required parameters for endpoint
func (cds *CDRDiscoveryService) hasRequiredParams(endpoint CDREndpointConfig, criteria CDRSearchCriteria) bool {
	for _, required := range endpoint.RequiredParams {
		switch required {
		case "domain":
			if criteria.Domain == "" {
				return false
			}
		case "user":
			if criteria.User == "" {
				return false
			}
		case "site":
			if criteria.Site == "" {
				return false
			}
		}
	}
	return true
}

// queryEndpoint queries a single endpoint and returns results
func (cds *CDRDiscoveryService) queryEndpoint(endpointConfig CDREndpointConfig, criteria CDRSearchCriteria) EndpointResult {
	queryStart := time.Now()

	// Initialize result with proper CDRs field
	result := EndpointResult{
		EndpointName:   endpointConfig.Name,
		CDRs:           []models.FlexibleCDR{},
		RawDataUsed:    false, // Will be set to true if raw=yes is used
		DiscoveredData: false, //
	}

	// Build URL with parameters (including raw=yes if supported)
	url, err := cds.buildEndpointURL(endpointConfig, criteria)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("URL build error: %v", err)
		result.QueryTime = time.Since(queryStart)
		return result
	}

	result.URL = url
	result.RawDataUsed = endpointConfig.SupportsRaw && criteria.Raw
	// logging to console:
	cds.logDebug("  URL: %s", url)

	// Make HTTP request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("Request creation error: %v", err)
		result.QueryTime = time.Since(queryStart)
		return result
	}

	// Add authorization header
	req.Header.Set("Authorization", "Bearer "+cds.accessToken)
	req.Header.Set("Accept", "application/json")

	// Execute request
	resp, err := cds.client.Do(req)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("HTTP request error: %v", err)
		result.QueryTime = time.Since(queryStart)
		return result
	}
	defer resp.Body.Close()

	result.HTTPStatus = resp.StatusCode
	result.QueryTime = time.Since(queryStart)

	// Handle non-200 responses
	if resp.StatusCode != http.StatusOK {
		result.Success = false
		result.Error = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, resp.Status)
		return result
	}

	// Parse JSON response
	var apiResponse interface{}
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("JSON decode error: %v", err)
		return result
	}

	// Convert to CDR models
	cdrs, err := cds.convertAPIResponseToCDRs(apiResponse)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("CDR conversion error: %v", err)
		return result
	}

	result.CDRs = cdrs
	result.RecordCount = len(cdrs)
	result.Success = true

	return result
}

// buildEndpointURL builds the complete URL for an endpoint with parameters (including raw=yes)
// Replace the buildEndpointURL method in your services/cdr_discovery.go file
// with this corrected version

// buildEndpointURL builds the complete URL for an endpoint with parameters (including raw=yes)
func (cds *CDRDiscoveryService) buildEndpointURL(endpointConfig CDREndpointConfig, criteria CDRSearchCriteria) (string, error) {
	// Start with URL template
	urlPath := endpointConfig.URLTemplate

	// Replace path parameters
	urlPath = strings.ReplaceAll(urlPath, "{domain}", criteria.Domain)
	urlPath = strings.ReplaceAll(urlPath, "{user}", criteria.User)
	urlPath = strings.ReplaceAll(urlPath, "{site}", criteria.Site)

	// Build query parameters
	params := url.Values{}

	// Add pagination parameters
	if criteria.Start > 0 {
		params.Add("start", fmt.Sprintf("%d", criteria.Start))
	}
	if criteria.Limit > 0 {
		params.Add("limit", fmt.Sprintf("%d", criteria.Limit))
	}

	// CRITICAL: Add raw=yes parameter if endpoint supports it and criteria requests it
	if endpointConfig.SupportsRaw && criteria.Raw {
		params.Add("raw", "yes")
	}

	// Add date parameters if provided
	if criteria.StartDate != nil {
		// Use NetSapiens standard parameter names (start/end, not start_date/end_date)
		params.Add("start", criteria.StartDate.Format("2006-01-02"))
	}
	if criteria.EndDate != nil {
		params.Add("end", criteria.EndDate.Format("2006-01-02"))
	}

	// Add call ID if provided
	if criteria.CallID != "" {
		params.Add("call_id", criteria.CallID)
	}

	// Add phone number parameters with correct field names
	if criteria.OriginatingNumber != "" {
		params.Add("orig_number", criteria.OriginatingNumber)
	}
	if criteria.TerminatingNumber != "" {
		params.Add("term_number", criteria.TerminatingNumber)
	}

	// Build final URL
	fullURL := cds.baseURL + urlPath
	if len(params) > 0 {
		fullURL += "?" + params.Encode()
	}

	return fullURL, nil
}

// convertAPIResponseToCDRs converts API response to FlexibleCDR models
func (cds *CDRDiscoveryService) convertAPIResponseToCDRs(apiResponse interface{}) ([]models.FlexibleCDR, error) {
	var cdrs []models.FlexibleCDR

	// Handle different response formats
	switch response := apiResponse.(type) {
	case []interface{}:
		// Array of CDR objects
		for _, item := range response {
			if cdrData, ok := item.(map[string]interface{}); ok {
				cdr, err := cds.convertMapToFlexibleCDR(cdrData)
				if err != nil {
					continue // Skip invalid CDRs, don't fail entire request
				}
				cdrs = append(cdrs, cdr)
			}
		}
	case map[string]interface{}:
		// Single CDR object or wrapped response
		if data, exists := response["data"]; exists {
			// Response is wrapped, recurse on data
			return cds.convertAPIResponseToCDRs(data)
		} else {
			// Single CDR object
			cdr, err := cds.convertMapToFlexibleCDR(response)
			if err != nil {
				return nil, err
			}
			cdrs = append(cdrs, cdr)
		}
	default:
		return nil, fmt.Errorf("unexpected API response format: %T", response)
	}

	return cdrs, nil
}

// convertMapToFlexibleCDR converts a map to FlexibleCDR
func (cds *CDRDiscoveryService) convertMapToFlexibleCDR(data map[string]interface{}) (models.FlexibleCDR, error) {
	var cdr models.FlexibleCDR

	// Convert map to JSON and then unmarshal into FlexibleCDR
	jsonData, err := json.Marshal(data)
	if err != nil {
		return cdr, err
	}

	err = json.Unmarshal(jsonData, &cdr)
	return cdr, err
}

// deduplicateCDRs removes duplicate CDRs based on ID
func (cds *CDRDiscoveryService) deduplicateCDRs(cdrs []models.FlexibleCDR) []models.FlexibleCDR {
	seen := make(map[string]bool)
	var unique []models.FlexibleCDR

	for _, cdr := range cdrs {
		id := cdr.GetID()
		if id != "" && !seen[id] {
			seen[id] = true
			unique = append(unique, cdr)
		}
	}

	return unique
}

// countTotalCDRs counts total CDRs across all endpoints
func (cds *CDRDiscoveryService) countTotalCDRs(cdrsByEndpoint map[string][]models.FlexibleCDR) int {
	total := 0
	for _, cdrs := range cdrsByEndpoint {
		total += len(cdrs)
	}
	return total
}

// generateSessionID generates a unique session ID
func (cds *CDRDiscoveryService) generateSessionID() string {
	return fmt.Sprintf("cdr_session_%d", time.Now().UnixNano())
}

// GetRawDataSummary returns a summary of which endpoints used raw data
func (cds *CDRDiscoveryService) GetRawDataSummary(result *CDRDiscoveryResult) map[string]bool {
	summary := make(map[string]bool)
	for _, endpointResult := range result.EndpointResults {
		summary[endpointResult.EndpointName] = endpointResult.RawDataUsed
	}
	return summary
}
