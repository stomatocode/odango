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
	"o-dan-go/services"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {

	// TEST command
	if len(os.Args) > 1 && os.Args[1] == "test-cdr" {
		testCDREndpoints()
		return
	}

	// Create a Gin router with default middleware (logger and recovery)
	r := gin.Default()

	// Define a simple route
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Welcome to O Dan Go!",
			"status":  "running",
		})
	})

	// Start server on port 8080
	r.Run(":8080")
}

func testCDREndpoints() {
	// Initialize service in test mode
	cdrService := services.NewCDRDiscoveryService(
		"https://core1-iad.dh.nseng.dev/ns-api/v2",                     // Base URL, core1 V2 right now only
		"nss_6dYMzuco6vLpU7gXR2QSXReVBM48Xf2mP2hGnbAba4xic1eiff2a146b", // API token
		true, // Enable test mode for debugging
	)

	fmt.Println("Testing CDR endpoint connectivity...")
	connectivityResult := cdrService.TestEndpointConnectivity()

	// Print results
	for endpointName, result := range connectivityResult.Results {
		status := "❌ FAILED"
		if result.Success {
			status = "✅ SUCCESS"
		}
		fmt.Printf("%s %s - %v - %s\n", status, endpointName, result.ResponseTime, result.URL)
		if result.Error != "" {
			fmt.Printf("   Error: %s\n", result.Error)
		}
	}

	fmt.Println("\nTesting with mock data...")
	criteria := services.CDRSearchCriteria{Domain: "test.com", Limit: 5}
	mockResult, _ := cdrService.MockCDRData(criteria)
	fmt.Printf("Mock test created %d CDRs successfully\n", len(mockResult.AllCDRs))
}
