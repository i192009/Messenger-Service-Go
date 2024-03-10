package routes

import (
	"github.com/gin-gonic/gin"
	"messenger/restful/controllers"
)

// LoadMessageRoutes : Set message routes
func LoadMessageRoutes(api *gin.RouterGroup) {
	api.GET("/ws", controllers.CreateWebSocketConnection)
	backend := api.Group("/message")
	backend.GET("/", controllers.HttpGetMessages)
	backend.GET("/query", controllers.HttpQueryMessages)
	backend.POST("/:messageId/setMessageStatus", controllers.HttpUpdateMessageStatus)
	backend.POST("/setMessageStatus", controllers.HttpUpdateMessagesStatus)
	backend.DELETE("/:messageId", controllers.HttpDeleteMessage)
}
