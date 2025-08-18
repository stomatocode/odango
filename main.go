package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"o-dan-go/config"
	"o-dan-go/handlers"
	"o-dan-go/services"

	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration first
	cfg := config.LoadConfig()

	// TEST command
	if len(os.Args) > 1 && os.Args[1] == "test-cdr" {
		testCDREndpoints(cfg)
		return
	}

	// Set Gin mode based on environment
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize CDR Discovery Service
	cdrService := services.NewCDRDiscoveryService(
		cfg.NetsapiensBaseURL,
		cfg.NetsapiensToken,
	)

	// Initialize Web Responder Service
	// TODO: Add SessionSecret to config
	wrService := services.NewWebResponderService("temporary-secret-change-me")
	wrHandler := handlers.NewWebResponderHandler(wrService)

	// Create a Gin router with default middleware
	r := gin.Default()

	// Load HTML templates for web interface
	r.LoadHTMLGlob("templates/*")

	// Serve static files
	r.Static("/static", "./static")

	// Print ASCII Art Banner
	fmt.Println(`
    ___       ____                 ____       
   / _ \     |  _ \  __ _ _ __    / ___| ___  
  | | | |____| | | |/ _` + "`" + ` | '_ \  | |  _ / _ \ 
  | |_| |____| |_| | (_| | | | | | |_| | (_) |
   \___/     |____/ \__,_|_| |_|  \____|\___/ 
                                              
  `)
	fmt.Printf("🍡 O Dan Go - NetSapiens CDR Discovery Platform\n")
	fmt.Printf("Version 1.0.0 | Environment: %s\n", cfg.AppEnv)
	fmt.Println("=" + strings.Repeat("=", 45))

	// API Routes (existing functionality)
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Welcome to O Dan Go!",
			"status":  "running",
			"env":     cfg.AppEnv,
			"features": gin.H{
				"cdr_discovery": "active",
				"web_responder": "active",
			},
		})
	})

	// Web Interface Routes (existing CDR functionality)
	r.GET("/web", handlers.ShowWelcomePage)
	r.GET("/web/search", handlers.ShowSearchForm)
	r.POST("/web/search", handlers.ProcessSearchForm(cdrService))
	r.GET("/web/results/:session_id", handlers.ShowResults)
	r.GET("/spa", handlers.ShowSPA)

	// Web Responder Routes (NEW)
	wr := r.Group("/wr")
	{
		// Weather IVR endpoint
		wr.GET("/weather", wrHandler.HandleWeatherIVR)
		wr.POST("/weather", wrHandler.HandleWeatherIVR)

		// Future endpoints can be added here
		// wr.GET("/menu", wrHandler.HandleMainMenu)
		// wr.GET("/cdr-lookup", wrHandler.HandleCDRLookup)
	}

	// API routes group
	api := r.Group("/api/v1")
	{
		api.GET("/health", handlers.HealthCheck)
		// Future API endpoints
		// api.GET("/cdrs", ...)
		// api.GET("/wr/status", ...)
	}

	// Start server
	fmt.Printf("\n📡 Starting O Dan Go server on port %s\n", cfg.AppPort)
	fmt.Printf("🌐 Web Interface: http://localhost:%s/web\n", cfg.AppPort)
	fmt.Printf("📞 Web Responder: http://localhost:%s/wr/weather\n", cfg.AppPort)
	fmt.Printf("🔗 API Endpoint: http://localhost:%s/\n", cfg.AppPort)
	fmt.Println("\nPress Ctrl+C to stop the server")

	r.Run(":" + cfg.AppPort)
}

func testCDREndpoints(cfg *config.Config) {
	fmt.Println("Testing CDR Discovery Service...")
	fmt.Printf("🔗 Base URL: %s\n", cfg.NetsapiensBaseURL)

	// Safe token display
	token := cfg.NetsapiensToken
	tokenLen := len(token)

	if tokenLen == 0 {
		fmt.Printf("🔑 Token: [EMPTY TOKEN]\n")
	} else if tokenLen <= 8 {
		fmt.Printf("🔑 Token: %s...\n", token[:min(4, tokenLen)])
	} else {
		start := min(8, tokenLen)
		end := max(0, tokenLen-4)
		fmt.Printf("🔑 Token: %s...%s\n", token[:start], token[end:])
	}

	fmt.Printf("🌍 Environment: %s\n\n", cfg.AppEnv)

	// Initialize service
	cdrService := services.NewCDRDiscoveryService(
		cfg.NetsapiensBaseURL,
		cfg.NetsapiensToken,
	)

	fmt.Println("🔍 Testing CDR Discovery with comprehensive search...")

	// Test with minimal criteria (discovers all endpoints)
	criteria := services.CDRSearchCriteria{
		Limit: 5, // Small limit for testing
	}

	startTime := time.Now()
	result, err := cdrService.GetComprehensiveCDRs(criteria)
	elapsed := time.Since(startTime)

	if err != nil {
		fmt.Printf("❌ Error during discovery: %v\n", err)
		return
	}

	fmt.Printf("\n✅ Discovery completed in %v\n", elapsed)
	fmt.Printf("📊 Results Summary:\n")
	fmt.Printf("   - Total CDRs found: %d\n", result.TotalCDRs)
	fmt.Printf("   - Unique CDRs: %d\n", result.UniqueCDRs)
	fmt.Printf("   - Duplicates removed: %d\n", result.Duplicates)
	fmt.Printf("   - Session ID: %s\n", result.SessionID)

	// Show endpoint breakdown
	fmt.Printf("\n📡 Endpoint Results:\n")
	for endpoint, endpointResult := range result.EndpointResults {
		fmt.Printf("   %s:\n", endpoint)
		fmt.Printf("      CDRs: %d\n", endpointResult.Count)
		fmt.Printf("      Success: %v\n", endpointResult.Success)
		if endpointResult.Error != "" {
			fmt.Printf("      Error: %s\n", endpointResult.Error)
		}
		fmt.Println()
	}

	// Show any global errors
	if len(result.Errors) > 0 {
		fmt.Printf("⚠️  Global Errors:\n")
		for _, err := range result.Errors {
			fmt.Printf("   - %s\n", err)
		}
	}

	// Test with domain-specific criteria if we have CDRs
	if result.UniqueCDRs > 0 {
		fmt.Println("\n🎯 Testing domain-specific query...")

		// Get a domain from the first CDR for testing
		firstCDR := result.AllCDRs[0]
		testDomain := firstCDR.GetDomain()

		if testDomain != "" {
			domainCriteria := services.CDRSearchCriteria{
				Domain: testDomain,
				Limit:  3,
			}

			domainResult, err := cdrService.GetComprehensiveCDRs(domainCriteria)
			if err != nil {
				fmt.Printf("❌ Domain query failed: %v\n", err)
			} else {
				fmt.Printf("✅ Domain query for '%s' found %d CDRs\n", testDomain, domainResult.UniqueCDRs)
			}
		}
	}

	// Show sample CDR data if available
	if len(result.AllCDRs) > 0 {
		fmt.Println("\n📋 Sample CDR Data:")
		sampleCDR := result.AllCDRs[0]
		fmt.Printf("   - ID: %s\n", sampleCDR.GetID())
		fmt.Printf("   - Domain: %s\n", sampleCDR.GetDomain())
		fmt.Printf("   - Direction: %d\n", sampleCDR.GetCallDirection())
		fmt.Printf("   - Duration: %d seconds\n", sampleCDR.GetCallDuration())
		fmt.Printf("   - Origin User: %s\n", sampleCDR.GetOrigUser())
		fmt.Printf("   - Term User: %s\n", sampleCDR.GetTermUser())
		fmt.Printf("   - Field Count: %d\n", len(sampleCDR.GetFieldNames()))
	}

	fmt.Println("\n🎉 CDR Discovery Service test completed!")
}

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
