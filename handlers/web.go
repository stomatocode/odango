package handlers

import (
	"net/http"
	"strconv"
	"time"

	"o-dan-go/services"

	"github.com/gin-gonic/gin"
)

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

// ProcessSearchForm handles search form submission
// This returns a gin.HandlerFunc, which is what Gin expects
func ProcessSearchForm(cdrService *services.CDRDiscoveryService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get form data
		domain := c.PostForm("domain")
		user := c.PostForm("user")
		callID := c.PostForm("call_id")
		number := c.PostForm("number")
		startDate := c.PostForm("start_date")
		endDate := c.PostForm("end_date")
		limitStr := c.DefaultPostForm("limit", "100")

		// Parse limit safely
		limit, err := strconv.Atoi(limitStr)
		if err != nil {
			limit = 100 // Default fallback
		}

		// Create search criteria
		criteria := services.CDRSearchCriteria{
			Domain: domain,
			User:   user,
			CallID: callID,
			Number: number,
			Limit:  limit,
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

		// Perform comprehensive search
		result, err := cdrService.GetComprehensiveCDRs(criteria)
		if err != nil {
			c.HTML(http.StatusInternalServerError, "error.html", gin.H{
				"title": "Search Error - O Dan Go",
				"error": err.Error(),
			})
			return
		}

		// Redirect to results page with session ID
		c.Redirect(http.StatusFound, "/web/results/"+result.SessionID)
	}
}

// ShowResults displays search results
func ShowResults(c *gin.Context) {
	sessionID := c.Param("session_id")

	// TODO: In future iterations, retrieve actual results from database
	// For now, show basic results page
	c.HTML(http.StatusOK, "results.html", gin.H{
		"title":     "Search Results - O Dan Go",
		"sessionID": sessionID,
		"message":   "CDR search completed. Results processing coming soon...",
	})
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
