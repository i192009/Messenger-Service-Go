package framework

import (
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"syscall"

	"google.golang.org/grpc"

	"github.com/gin-gonic/gin"
	"gitlab.zixel.cn/go/framework/bus"
	"gitlab.zixel.cn/go/framework/config"
	"gitlab.zixel.cn/go/framework/logger"
)

var log = logger.Get()

// public
var webServer *WebServer
var rpcServer *RpcServer
var busServer *bus.AsyncEventBus

func init() {
	// 初始化事件总线
	busServer = bus.NewAsyncEventBus()

	// 初始化日志系统
	if err := logger.InitLogger(
		config.GetString("log.level", "debug"),
		filepath.Join(
			config.GetString("log.path", ""),
			config.GetString("log.file", "log.log"),
		),
	); err != nil {
		panic(err)
	}

	busServer.Subscribe("log.debug", log.Debug)
	busServer.Subscribe("log.error", log.Error)
	busServer.Subscribe("log.fatal", log.Fatal)
	busServer.Subscribe("log.info", log.Info)

	log.Info("framework initialize ...")

	// 监听应用退出事件
	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	/// 将信号转为事件
	go func() {
		sig := <-quit
		busServer.Publish("ExitWebServer")
		busServer.Publish("ExitRpcServer")
		busServer.Publish("Quit", sig)
	}()

	// 设置服务详情
	SetServiceDetail(
		config.GetString("service.name", "NoServiceName"),
		config.GetString("service.uuid", "NoServiceUUID"),
	)

	// 初始化WEB服务
	webServer = NewWebServer(busServer)
	if webServer == nil {
		log.Error("create web service error.")
		panic("web service create failed")
	}

	// 初始化RPC服务
	rpcServer = NewRpcServer(busServer)
	if rpcServer == nil {
		log.Error("create rpc service error.")
		panic("rpc service create failed")
	}

	// 连接RPC服务
	if err := ConnectGrpcServers(); err != nil {
		log.Error("connect rpc service error.")
		panic(err)
	}

	// 初始化消息总线

}

func LoadRoute(loader func(c *gin.RouterGroup)) {
	engine := webServer.GetEngine()
	if engine != nil {
		loader(&engine.RouterGroup)
	}
}

func LoadServiceRoute(loader func(c *gin.RouterGroup), version ...string) {
	engine := webServer.GetEngine()
	println("for confiqbf I am here")
	name := config.GetString("service.name", "")
	if name == "" {
		panic("service.name not found")
	}

	if len(version) > 0 {
		//panic if version is incorrect
		if !regexp.MustCompile("^v[0-9]+$").MatchString(version[0]) {
			panic("invalid route version: " + version[0])
		}
		name = name + "/" + version[0]
	}

	prefix := config.GetString("web.prefix_path", name)
	print("Prefix Path in app.go")

	loader(engine.Group(prefix))
}

func GetGrpcServer() *grpc.Server {
	return rpcServer.rpc
}

func Run() {
	webServer.Start()
	rpcServer.Start()

	webServer.Wait()
	rpcServer.Wait()
	config.IsRunning = false
}

func IsRunning() bool {
	return config.IsRunning
}

func IsDebug() bool {
	return config.IsDebug
}
