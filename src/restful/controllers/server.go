package controllers

import (
	"github.com/gin-gonic/gin"
	"messenger/restful/services"
)

// CreateWebSocketConnectionForApplication : Create WebSocket Connection for redirecting messages to application
func CreateWebSocketConnectionForApplication(c *gin.Context) {
	// Returning error and logging if failed to create WebSocket connection for application
	if err := services.CreateWebSocketConnectionForApplication(c); err != nil {
		return
	}

	// Returning success and logging if succeeded to create WebSocket connection for application
	log.Info("WebSocket Connection Created")
}

// CreateWebSocketConnectionForInstance : Create WebSocket Connection for redirecting messages to instance
func CreateWebSocketConnectionForInstance(c *gin.Context) {
	// Returning error and logging if failed to create WebSocket connection for application
	if err := services.CreateWebSocketConnectionForInstance(c); err != nil {
		return
	}

	// Returning success and logging if succeeded to create WebSocket connection for application
	log.Info("WebSocket Connection Created")
}
