// handlers/web.go
// Updated version with correct phone number fields and enhanced validation

package handlers

import (
	"fmt"
	"log" // logging line
	"net/http"
	"o-dan-go/main"
	"o-dan-go/services"
	"regexp"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// ShowSPA serves the single page application
func ShowSPA(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{
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

// ProcessSearchForm handles search form submission with enhanced validation
func ProcessSearchForm(cdrService *services.CDRDiscoveryService) gin.HandlerFunc {
	return func(c *gin.Context) {
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

		// Handle Call ID searches with priority (synchronous processing)
		if criteria.CallID != "" {
			result, err := cdrService.GetComprehensiveCDRs(criteria)
			if err != nil {
				c.HTML(http.StatusInternalServerError, "error.html", gin.H{
					"title": "Call ID Search Error - O Dan Go",
					"error": fmt.Sprintf("Call ID search failed: %v", err),
				})
				return
			}

			// For Call ID searches, redirect directly to results with immediate data
			c.Redirect(http.StatusFound, "/web/results/"+result.SessionID)
			return
		}
		// log to console
		log.Printf("[Web Handler] Starting CDR discovery...")

		// For other searches, perform comprehensive search
		result, err := cdrService.GetComprehensiveCDRs(criteria)

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

		main.GlobalResultsStore.Store(result.SessionID, result)

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
	result, exists := main.GlobalResultsStore.Get(sessionID)

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
