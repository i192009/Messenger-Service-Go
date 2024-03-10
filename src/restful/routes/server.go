package routes

import (
	"github.com/gin-gonic/gin"
	"messenger/restful/controllers"
)

// LoadServerRoutes : Set server routes
func LoadServerRoutes(api *gin.RouterGroup) {
	backend := api.Group("/server")
	backend.GET("/ws", controllers.CreateWebSocketConnectionForApplication)
	backend.GET("/instance/ws", controllers.CreateWebSocketConnectionForInstance)
}
