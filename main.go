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
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
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
