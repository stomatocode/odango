/*

                                                       -*
                                                     =@ *@
                                            @@@@@   @  @
                                         #@----::*@# @-
                                        :@:-------:%@
                                       +@=----------:@
                                    @*   @%:--------:#@
                                   @       .#::-----:@
                                  :@        +@#:---:@+             .
                               #@#+@@          %@%@%             @+ @
                              @*--::#@=         @       #@@%.  @% %@
                              @--::::-@@       .@    #@=:::::%@ +@
                           @@@@*--::::--%@%:.*@=    @*::::::::-@@
                         @#:---%@---::::--@@       #@:::::::::::+@
                        @=:---:--@@-::::--@.    @*  #@+:-:-:::::-@
                        @::------:+@#+-=%@    -@       =-:-:::---@
                        @=:::::::::--%@       @         @@--:--=@
                         @#:-::::::--@     @@#%@           ##%@-
                        :@=@*::::::#@    @@-:--*@.          @
                       @  @- @@%@@%      @-:::::-@@        @
                     @= @#            @@#@#-::::::=@@+   *@
                   @* %@            @#:--:##-::::::-:*@
                 @% +@             @#:----:-@@--:::::%@
               #@ :@               @--------:=@*=--+@%
             -@  @                 @*:--::----:-@@
            @. @=                   @*:-:------:@
           @.@%                    @-#@+----::*@
                                 @= @% .@@@@@%
                               @# %@
                             @@ +@
                           %@ :@
                         =@  @=
                        @  @+
                      @= @%
                     ##%@


   O Dan Go!
   VoIP Management System
   NetSapiens API Integration
   MIT License
*/

package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"o-dan-go/config"
	"o-dan-go/handlers" // handlers for web interface
	"o-dan-go/services"

	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration first
	cfg := config.LoadConfig()

	// TEST command
	if len(os.Args) > 1 && os.Args[1] == "test-cdr" {
		testCDREndpoints(cfg) // Pass config to test function
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

	// Create a Gin router with default middleware (logger and recovery)
	r := gin.Default()

	// Load HTML templates for web interface
	r.LoadHTMLGlob("templates/*")

	// Serve static files (CSS, JS, images)
	r.Static("/static", "./static")

	// API Routes (existing functionality)
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Welcome to O Dan Go!",
			"status":  "running",
			"env":     cfg.AppEnv,
		})
	})

	// Web Interface Routes (new functionality)
	r.GET("/web", handlers.ShowWelcomePage)
	r.GET("/web/search", handlers.ShowSearchForm)
	r.POST("/web/search", handlers.ProcessSearchForm(cdrService))
	r.GET("/web/results/:session_id", handlers.ShowResults)

	// API routes group for future expansion
	api := r.Group("/api/v1")
	{
		api.GET("/health", handlers.HealthCheck)
		// Future API endpoints can go here
	}

	// Start server on configured port
	fmt.Printf("Starting O Dan Go server on port %s in %s mode\n", cfg.AppPort, cfg.AppEnv)
	fmt.Printf("🌐 Web Interface: http://localhost:%s/web\n", cfg.AppPort)
	fmt.Printf("🔗 API Endpoint: http://localhost:%s/\n", cfg.AppPort)
	r.Run(":" + cfg.AppPort)
}

// Keep your existing testCDREndpoints function unchanged
func testCDREndpoints(cfg *config.Config) {
	fmt.Println("Testing CDR Discovery Service...")
	fmt.Printf("🔗 Base URL: %s\n", cfg.NetsapiensBaseURL)
	fmt.Printf("🔑 Token: %s...%s\n",
		cfg.NetsapiensToken[:min(8, len(cfg.NetsapiensToken))],
		cfg.NetsapiensToken[max(0, len(cfg.NetsapiensToken)-4):])

	// Initialize service using configuration (correct 2-parameter constructor)
	cdrService := services.NewCDRDiscoveryService(
		cfg.NetsapiensBaseURL,
		cfg.NetsapiensToken,
	)

	fmt.Println("✅ CDR Discovery Service initialized successfully")

	// Test endpoint configuration
	endpoints := cdrService.GetSupportedEndpoints()
	fmt.Printf("📋 Found %d supported endpoints:\n", len(endpoints))

	for _, endpoint := range endpoints {
		fmt.Printf("   - %s: %s\n", endpoint.Name, endpoint.Description)
		fmt.Printf("     URL Template: %s\n", endpoint.URLTemplate)
		if len(endpoint.RequiredParams) > 0 {
			fmt.Printf("     Required: %v\n", endpoint.RequiredParams)
		}
	}

	// Test basic connectivity with minimal criteria
	fmt.Println("\n🔍 Testing basic CDR query...")

	criteria := services.CDRSearchCriteria{
		Limit: 5, // Just get a few records for testing
	}

	start := time.Now()
	result, err := cdrService.GetComprehensiveCDRs(criteria)
	duration := time.Since(start)

	if err != nil {
		fmt.Printf("❌ Error during CDR query: %v\n", err)
		return
	}

	// Print comprehensive results
	fmt.Printf("✅ Query completed in %v\n", duration)
	fmt.Printf("📊 Results Summary:\n")
	fmt.Printf("   - Session ID: %s\n", result.SessionID)
	fmt.Printf("   - Total CDRs found: %d\n", result.TotalCDRs)
	fmt.Printf("   - Unique CDRs: %d\n", result.UniqueCDRs)
	fmt.Printf("   - Endpoints queried: %d\n", len(result.EndpointResults))

	// Show detailed endpoint results
	fmt.Println("\n📡 Endpoint Results:")
	for _, endpointResult := range result.EndpointResults {
		status := "❌ FAILED"
		if endpointResult.Success {
			status = "✅ SUCCESS"
		}
		fmt.Printf("   %s %s\n", status, endpointResult.EndpointName)
		fmt.Printf("      URL: %s\n", endpointResult.URL)
		fmt.Printf("      Records: %d\n", endpointResult.RecordCount)
		fmt.Printf("      Response Time: %v\n", endpointResult.QueryTime)
		fmt.Printf("      HTTP Status: %d\n", endpointResult.HTTPStatus)

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

// Keep your existing helper functions unchanged
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
