package main

import (
	"messenger/configs"
	"messenger/restful/controllers"
	"messenger/restful/routes"
	"messenger/restful/services"
	pb "messenger/services"

	"gitlab.zixel.cn/go/framework"
)

func main() {
	services.ServerInstance = services.InitializeServer()
	print("Main")
	framework.LoadRoute(routes.LoadProbeRoutes)
	framework.LoadServiceRoute(routes.LoadServerRoutes)
	framework.LoadServiceRoute(routes.LoadConfigurationRoutes)
	framework.LoadServiceRoute(routes.LoadTemplateRoutes)
	framework.LoadServiceRoute(routes.LoadMessageRoutes)

	for code, msg := range configs.ErrorCodes {
		framework.SetErrorTips(code, msg)
	}

	s := framework.GetGrpcServer()
	pb.RegisterTemplateServiceServer(s, &controllers.TemplateServer{})
	pb.RegisterConfigurationServiceServer(s, &controllers.ConfigurationServer{})
	pb.RegisterMessageServiceServer(s, &controllers.MessageServer{})
	pb.RegisterSceneServiceServer(s, &controllers.SceneServer{})
	pb.RegisterSceneRelationServiceServer(s, &controllers.SceneRelationServer{})

	go framework.Run()
	if configs.IsReleaseMode() {
		services.SubscribeThreePartMessage()
	} else {
		select {}
	}
}
