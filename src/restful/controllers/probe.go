package controllers

import (
	"github.com/gin-gonic/gin"
	"messenger/restful/services"
)

func Health(c *gin.Context) {
	if err := services.Health(c); err != nil {
		c.JSON(500, gin.H{
			"Message": "Internal Server Error",
		})
		return
	} else {
		c.JSON(200, gin.H{
			"Message": "OK",
		})
	}
}

func Available(c *gin.Context) {
	if err := services.Available(c); err != nil {
		c.JSON(500, gin.H{
			"Message": "Internal Server Error",
		})
		return
	} else {
		c.JSON(200, gin.H{
			"Message": "OK",
		})
	}
}
