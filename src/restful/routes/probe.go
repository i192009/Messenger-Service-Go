package routes

import (
	"github.com/gin-gonic/gin"
	"messenger/restful/controllers"
)

// LoadProbeRoutes : Set probe routes
func LoadProbeRoutes(app *gin.RouterGroup) {
	app.GET("/available", controllers.Available)
	app.GET("/health", controllers.Health)
}
