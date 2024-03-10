package controllers

import (
	"github.com/gin-gonic/gin"
	"gitlab.zixel.cn/go/framework"
	"gitlab.zixel.cn/go/framework/logger"
	"google.golang.org/grpc/status"
	pb "messenger/services"
)

var log = logger.Get()

// SendErrorToJson : Convert error status into http error
func SendErrorToJson(c *gin.Context, err error) {
	errStatus, _ := status.FromError(err)

	// convert interface to map
	errDetails := errStatus.Details()[0].(*pb.ErrorResponse)

	framework.Error(c, framework.NewServiceError(int(errDetails.GetError().GetCode()), ""))
}
