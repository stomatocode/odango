// handlers/web.go
// Updated version with correct phone number fields and enhanced validation

package handlers

import (
	"encoding/json"
	"fmt"
	"log" // logging line
	"net/http"
	"o-dan-go/services"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ShowSPA serves the single page application
func ShowSPA(c *gin.Context) {
	c.HTML(http.StatusOK, "spa.html", gin.H{
		"title": "O Dan Go - CDR Discovery",
	})
}

// ShowWelcomePage displays the main welcome page
func ShowWelcomePage(c *gin.Context) {
	c.HTML(http.StatusOK, "welcome.html", gin.H{
		"title":   "O Dan Go - NetSapiens CDR Discovery",
		"version": "1.0.0",
	})
}

// ShowSearchForm displays the CDR search form
func ShowSearchForm(c *gin.Context) {
	c.HTML(http.StatusOK, "search.html", gin.H{
		"title": "CDR Search - O Dan Go",
	})
}

// ProcessSearchForm handles search form submission with enhanced validation, with API credentials
func ProcessSearchForm(cdrService *services.CDRDiscoveryService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get API credentials from form
		apiURL := c.PostForm("api_url")
		apiToken := c.PostForm("api_token")

		// Validate API credentials
		if apiURL == "" || apiToken == "" {
			c.HTML(http.StatusBadRequest, "error.html", gin.H{
				"title": "Authentication Error - O Dan Go",
				"error": "API URL and Bearer Token are required",
			})
			return
		}

		// Create CDR service with user-provided credentials
		userCDRService := services.NewCDRDiscoveryService(apiURL, apiToken)

		// Get form data with UPDATED field names
		domain := c.PostForm("domain")
		user := c.PostForm("user")
		site := c.PostForm("site")
		callID := c.PostForm("call_id")

		// NEW: Get phone number fields with correct names
		originatingNumber := c.PostForm("originating_number")
		terminatingNumber := c.PostForm("terminating_number")
		anyPhoneNumber := c.PostForm("any_phone_number")

		startDate := c.PostForm("start_date")
		endDate := c.PostForm("end_date")
		limitStr := c.DefaultPostForm("limit", "100")

		// Parse limit safely
		limit, err := strconv.Atoi(limitStr)
		if err != nil {
			limit = 100 // Default fallback
		}

		// **** Validation
		// logging
		log.Printf("[Web Handler] Processing search request")
		log.Printf("[Web Handler] Domain: %s, User: %s, Site: %s", domain, user, site)
		validationErrors := validateSearchCriteria(domain, user, site, callID,
			originatingNumber, terminatingNumber, anyPhoneNumber, startDate, endDate)

		if len(validationErrors) > 0 {
			c.HTML(http.StatusBadRequest, "error.html", gin.H{
				"title": "Validation Error - O Dan Go",
				"error": fmt.Sprintf("Search validation failed: %s", validationErrors[0]),
			})
			return
		}

		// Create search criteria with UPDATED field names
		criteria := services.CDRSearchCriteria{
			Domain:            domain,
			User:              user,
			Site:              site,
			CallID:            callID,
			Limit:             limit,
			OriginatingNumber: originatingNumber,
			TerminatingNumber: terminatingNumber,
			AnyPhoneNumber:    anyPhoneNumber,
		}

		// Parse dates if provided
		if startDate != "" {
			if parsedDate, err := time.Parse("2006-01-02", startDate); err == nil {
				criteria.StartDate = &parsedDate
			}
		}
		if endDate != "" {
			if parsedDate, err := time.Parse("2006-01-02", endDate); err == nil {
				criteria.EndDate = &parsedDate
			}
		}

		// log to console
		log.Printf("[Web Handler] Starting CDR discovery with user-provided credentials...")

		// Use the user-provided CDR service instead of the default one
		result, err := userCDRService.GetComprehensiveCDRs(criteria)

		if err != nil {
			log.Printf("[Web Handler] ERROR: CDR search failed: %v", err) // logging

			c.HTML(http.StatusInternalServerError, "error.html", gin.H{
				"title": "Search Error - O Dan Go",
				"error": fmt.Sprintf("CDR search failed: %v", err),
			})
			return
		}

		// logging
		log.Printf("[Web Handler] Search completed successfully")
		log.Printf("[Web Handler] Session ID: %s", result.SessionID)
		log.Printf("[Web Handler] Total CDRs: %d, Unique: %d", result.TotalCDRs, result.UniqueCDRs)

		services.GlobalResultsStore.Store(result.SessionID, result)

		// Redirect to results page with session ID
		c.Redirect(http.StatusFound, "/web/results/"+result.SessionID)
	}
}

// NEW: Enhanced search validation function
func validateSearchCriteria(domain, user, site, callID, originatingNumber, terminatingNumber, anyPhoneNumber, startDate, endDate string) []string {
	var errors []string

	// Check that at least one search criterion is provided
	hasSearchCriteria := domain != "" || user != "" || site != "" || callID != "" ||
		originatingNumber != "" || terminatingNumber != "" || anyPhoneNumber != "" ||
		startDate != "" || endDate != ""

	if !hasSearchCriteria {
		errors = append(errors, "At least one search criterion is required")
		return errors // Return early if no criteria at all
	}

	// Validate phone number formats if provided
	phoneValidationRules := map[string]string{
		"Originating Number": originatingNumber,
		"Terminating Number": terminatingNumber,
		"Any Phone Number":   anyPhoneNumber,
	}

	for fieldName, phoneNumber := range phoneValidationRules {
		if phoneNumber != "" && !isValidPhoneNumber(phoneNumber) {
			errors = append(errors, fmt.Sprintf("%s has invalid format. Use digits, +, spaces, parentheses, or dashes", fieldName))
		}
	}

	// Validate phone number exclusivity (prevent conflicting searches)
	phoneFieldCount := 0
	if originatingNumber != "" {
		phoneFieldCount++
	}
	if terminatingNumber != "" {
		phoneFieldCount++
	}
	if anyPhoneNumber != "" {
		phoneFieldCount++
	}

	if phoneFieldCount > 1 {
		errors = append(errors, "Use either 'Any Phone Number' OR specific 'Originating/Terminating' numbers, not both")
	}

	// Validate Call ID exclusivity (Call ID should be used alone for precise searches)
	if callID != "" && (originatingNumber != "" || terminatingNumber != "" || anyPhoneNumber != "") {
		errors = append(errors, "Call ID searches should be used alone for best results")
	}

	// Validate date range logic
	if startDate != "" && endDate != "" {
		start, err1 := time.Parse("2006-01-02", startDate)
		end, err2 := time.Parse("2006-01-02", endDate)

		if err1 != nil {
			errors = append(errors, "Invalid start date format. Use YYYY-MM-DD")
		}
		if err2 != nil {
			errors = append(errors, "Invalid end date format. Use YYYY-MM-DD")
		}

		if err1 == nil && err2 == nil {
			if start.After(end) {
				errors = append(errors, "Start date must be before or equal to end date")
			}

			// Check for reasonable date ranges (prevent overly broad searches)
			daysDiff := end.Sub(start).Hours() / 24
			if daysDiff > 365 {
				errors = append(errors, "Date range longer than 1 year may return excessive data. Consider narrowing the range")
			}
		}
	}

	// Validate user/site requires domain context
	if (user != "" || site != "") && domain == "" {
		errors = append(errors, "User or Site searches require a Domain to be specified")
	}

	return errors
}

// NEW: Phone number validation helper
func isValidPhoneNumber(phone string) bool {
	if phone == "" {
		return true // Empty is valid (optional field)
	}

	// Allow digits, +, spaces, parentheses, and dashes
	// Minimum 7 characters for a valid phone number
	phoneRegex := regexp.MustCompile(`^[\+]?[\d\s\(\)-]{7,}$`)
	return phoneRegex.MatchString(phone)
}

// ShowResults displays search results
func ShowResults(c *gin.Context) {
	sessionID := c.Param("session_id")

	// Try to get results from memory store
	result, exists := services.GlobalResultsStore.Get(sessionID)

	if exists {
		// Calculate query time
		queryTime := result.EndTime.Sub(result.StartTime).Seconds()

		c.HTML(http.StatusOK, "results.html", gin.H{
			"title":     "Search Results - O Dan Go",
			"sessionID": sessionID,
			"message": fmt.Sprintf("Found %d unique CDRs from %d total CDRs across %d endpoints",
				result.UniqueCDRs, result.TotalCDRs, len(result.EndpointResults)),
			"totalCDRs":     result.TotalCDRs,
			"uniqueCDRs":    result.UniqueCDRs,
			"endpointCount": len(result.EndpointResults),
			"queryTime":     fmt.Sprintf("%.2f", queryTime),
			"endpoints":     result.EndpointResults,
		})
	} else {
		c.HTML(http.StatusOK, "results.html", gin.H{
			"title":     "Search Results - O Dan Go",
			"sessionID": sessionID,
			"message":   "Session not found or expired. Results are stored for 1 hour.",
		})
	}
}

// HealthCheck provides API health status
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"service":   "O Dan Go CDR Discovery",
		"version":   "1.0.0",
		"timestamp": time.Now().UTC(),
	})
}

// Add these functions to your handlers/web.go file

// ExportCDRs handles export requests for CDR data
func ExportCDRs(c *gin.Context) {
	sessionID := c.Param("session_id")
	format := c.DefaultQuery("format", "csv")

	// Retrieve results from store
	result, exists := services.GlobalResultsStore.Get(sessionID)
	if !exists {
		c.HTML(http.StatusNotFound, "error.html", gin.H{
			"title": "Export Error",
			"error": "Session not found or expired",
		})
		return
	}

	switch format {
	case "csv":
		exportCSV(c, result)
	case "json":
		exportJSON(c, result)
	default:
		c.HTML(http.StatusBadRequest, "error.html", gin.H{
			"title": "Export Error",
			"error": "Unsupported export format: " + format,
		})
	}
}

// exportCSV exports CDR data as CSV
func exportCSV(c *gin.Context, result *services.CDRDiscoveryResult) {
	// Set headers for CSV download
	filename := fmt.Sprintf("cdrs_%s.csv", result.SessionID)
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	// Write CSV header - using common CDR fields
	csvHeader := []string{
		"call_id",
		"domain",
		"user",
		"orig_number",
		"term_number",
		"start_time",
		"end_time",
		"duration",
		"call_type",
		"direction",
		"disposition",
		"session_id",
	}

	c.Writer.Write([]byte(strings.Join(csvHeader, ",") + "\n"))

	// Write CDR data
	for _, cdr := range result.AllCDRs {
		row := []string{
			escapeCSV(cdr.GetString("call-id")),
			escapeCSV(cdr.GetDomain()),
			escapeCSV(cdr.GetString("user")),
			escapeCSV(cdr.GetString("orig-number")),
			escapeCSV(cdr.GetString("term-number")),
			escapeCSV(cdr.GetString("start-time")),
			escapeCSV(cdr.GetString("end-time")),
			escapeCSV(fmt.Sprintf("%d", cdr.GetInt("duration"))),
			escapeCSV(cdr.GetString("call-type")),
			escapeCSV(cdr.GetString("direction")),
			escapeCSV(cdr.GetString("disposition")),
			escapeCSV(result.SessionID),
		}
		c.Writer.Write([]byte(strings.Join(row, ",") + "\n"))
	}
}

// exportJSON exports CDR data as JSON
func exportJSON(c *gin.Context, result *services.CDRDiscoveryResult) {
	// Set headers for JSON download
	filename := fmt.Sprintf("cdrs_%s.json", result.SessionID)
	c.Header("Content-Type", "application/json")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	// Create export structure
	export := map[string]interface{}{
		"session_id":      result.SessionID,
		"search_criteria": result.SearchCriteria,
		"query_time":      result.EndTime.Sub(result.StartTime).Seconds(),
		"total_cdrs":      result.TotalCDRs,
		"unique_cdrs":     result.UniqueCDRs,
		"export_time":     time.Now().UTC(),
		"cdrs":            result.AllCDRs,
	}

	// Pretty print JSON
	encoder := json.NewEncoder(c.Writer)
	encoder.SetIndent("", "  ")
	encoder.Encode(export)
}

// escapeCSV escapes special characters in CSV fields
func escapeCSV(field string) string {
	// If field contains comma, quote, or newline, wrap in quotes
	if strings.ContainsAny(field, ",\"\n\r") {
		// Escape quotes by doubling them
		field = strings.ReplaceAll(field, "\"", "\"\"")
		return fmt.Sprintf("\"%s\"", field)
	}
	return field
}

// GetCDRsAPI returns CDR data as JSON for AJAX requests
func GetCDRsAPI(c *gin.Context) {
	sessionID := c.Param("session_id")
	limitStr := c.DefaultQuery("limit", "10")
	limit, _ := strconv.Atoi(limitStr)

	log.Printf("[GetCDRsAPI] Fetching CDRs for session: %s, limit: %d", sessionID, limit)

	// Retrieve results from store
	result, exists := services.GlobalResultsStore.Get(sessionID)
	if !exists {
		log.Printf("[GetCDRsAPI] Session not found: %s", sessionID)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Session not found or expired",
		})
		return
	}

	log.Printf("[GetCDRsAPI] Found session with %d CDRs", len(result.AllCDRs))

	// Prepare CDR data for preview
	var previewCDRs []map[string]interface{}
	count := 0
	for _, cdr := range result.AllCDRs {
		if count >= limit {
			break
		}

		// Extract common fields for preview
		previewCDRs = append(previewCDRs, map[string]interface{}{
			"call_id":     cdr.GetID(),                          // Use GetID() method
			"domain":      cdr.GetDomain(),                      // Use GetDomain() method
			"orig_number": cdr.GetString("call-orig-caller-id"), // Correct field name
			"term_number": cdr.GetString("call-term-caller-id"), // Correct field name
			"start_time":  cdr.GetString("call-start-datetime"), // Correct field name
			"duration":    cdr.GetInt("call-duration"),          // Correct field name
		})
		count++
	}

	log.Printf("[GetCDRsAPI] Returning %d CDRs", len(previewCDRs))

	c.JSON(http.StatusOK, gin.H{
		"session_id": sessionID,
		"total":      len(result.AllCDRs),
		"limit":      limit,
		"cdrs":       previewCDRs,
	})
}
