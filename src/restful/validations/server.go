package validations

import (
	"github.com/gin-gonic/gin"
	"gitlab.zixel.cn/go/framework"
	"net/http"
)

// WebSocketConnectionForApplication : validates the request for websocket connection
func WebSocketConnectionForApplication(c *gin.Context) error {
	// Returning error and logging if failed to parse request
	if err := c.Request.ParseForm(); err != nil {
		return err
	}

	// Returning error and logging if open id is empty
	appId := c.Query("appId")
	if appId == "" {
		log.Error("appId is empty")
		framework.Error(c, framework.NewServiceError(100007, ""))
		return createError(http.StatusBadRequest, 100007)
	}

	// Returning nil if successful
	return nil
}

// WebSocketConnectionForInstance : validates the request for websocket connection
func WebSocketConnectionForInstance(c *gin.Context) error {
	// Returning error and logging if failed to parse request
	if err := c.Request.ParseForm(); err != nil {
		return err
	}

	// Returning nil if successful
	return nil
}
