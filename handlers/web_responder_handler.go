package handlers

import (
	"net/http"
	"o-dan-go/services"

	"github.com/gin-gonic/gin"
)

// WebResponderHandler handles Web Responder routes
type WebResponderHandler struct {
	wrService *services.WebResponderService
}

// NewWebResponderHandler creates a new Web Responder handler
func NewWebResponderHandler(wrService *services.WebResponderService) *WebResponderHandler {
	return &WebResponderHandler{
		wrService: wrService,
	}
}

// HandleWeatherIVR handles weather IVR requests from NetSapiens
func (wrh *WebResponderHandler) HandleWeatherIVR(c *gin.Context) {
	// Get parameters from NetSapiens
	callerNumber := c.Query("NmsAni")
	digits := c.Query("Digits")

	// Get or create session
	session, err := wrh.wrService.GetSession(c.Request, "weather-ivr-session")
	if err != nil {
		c.String(http.StatusInternalServerError, "Session error")
		return
	}

	// Process the IVR request
	xmlResponse, err := wrh.wrService.ProcessWeatherIVR(session, callerNumber, digits)
	if err != nil {
		c.String(http.StatusInternalServerError, "Processing error")
		return
	}

	// Save session
	session.Save(c.Request, c.Writer)

	// Return XML response for NetSapiens
	c.Header("Content-Type", "text/xml")
	c.String(http.StatusOK, xmlResponse)
}
