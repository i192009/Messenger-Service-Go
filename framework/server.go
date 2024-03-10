package framework

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	grpcStatus "google.golang.org/grpc/status"

	"gitlab.zixel.cn/go/framework/bus"
	"gitlab.zixel.cn/go/framework/config"
	"gitlab.zixel.cn/go/framework/xutil"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type Server struct {
	bus *bus.AsyncEventBus
}

type Service interface {
	Start() error
	Wait(timeout time.Duration) error
	Stop() error
}

type WebServer struct {
	Server
	eng *gin.Engine  /// http server
	web *http.Server /// web server
}

type RpcServer struct {
	Server
	rpc *grpc.Server /// rpc server
}

type RouteConfigItem struct {
	Method     string   `json:"method,string"`
	Path       string   `json:"path,string"`
	Handler    string   `json:"handler,string"`
	MiddleWare []string `json:"middleWare,omitempty"`
}

type responseRecorder struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

type grpcstruct struct {
	TypeURL string `protobuf:"bytes,1,opt,name=type_url,json=typeUrl,proto3" json:"type_url,omitempty"`
	Value   string `protobuf:"bytes,2,opt,name=value,proto3" json:"value,omitempty"`
	Version string `protobuf:"bytes,3,opt,name=version,proto3" json:"version,omitempty"`
}

var serverIp, _ = GetLocalIP()

// ----------------------------------------------------------
//
//	Web Server
//
// ----------------------------------------------------------
func NewWebServer(bus *bus.AsyncEventBus) *WebServer {
	var srv WebServer
	srv.web = nil
	srv.bus = bus
	//Creating the Gin applicationlication
	srv.eng = gin.New()

	// Create an atomic counter

	srv.eng.Use(CustomHttpServerInterceptor())

	srv.eng.Use(gin.Recovery())
	srv.eng.Use(ErrorHandler())

	return &srv
}

func handleCustomHttpServerLog(c *gin.Context, t time.Time, counter *atomic.Int64, responseRecorder *responseRecorder, params, body string) {

	zixelInvokerLevel := c.GetHeader("Zixel-Log-InvokerLevel")
	zixelRequestId := c.GetHeader("Zixel-Log-RequestId")
	zixelLogResponseIgnored := c.GetHeader("Zixel-Log-ResponseIgnored")

	if zixelInvokerLevel == "" {
		zixelInvokerLevel = "01"
	}

	// After request
	counter.Add(1)
	latency := float64(time.Since(t)) / float64(time.Millisecond)
	status := c.Writer.Status()

	var response string
	if zixelLogResponseIgnored != "true" {
		response = responseRecorder.body.String()
	} else {
		response = "##ignored##"
	}

	responseLevel := zixelInvokerLevel + "." + fmt.Sprintf("%02d", counter.Load())

	// fmt.Println(c.Request.Header)
	logLine := fmt.Sprintf("[receive-log]%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%d|%s|%s\n",
		zixelRequestId,                      // RequestId
		t.Format("2006-01-02 15:04:05.000"), // Timestamp
		"http",                              // Caller service name
		c.Request.Method,                    // Calling method
		c.HandlerName(),                     // Calling method name
		fmt.Sprintf("{\"params\":\"%s\",\"body\":%s}", params, body), // Caller params
		c.ClientIP(),                         // Caller IP
		zixelInvokerLevel,                    // Caller level number
		config.GetString("service.name", ""), // Responder service name
		c.FullPath(),                         // Responder method
		response,                             // Return parameters
		serverIp,                             // Responder IP
		responseLevel,                        // Responder level number
		status,                               // System return code
		http.StatusText(status),              // System return message
		fmt.Sprintf("%.2fms", latency),       // Response time
	)

	fmt.Print(logLine)
}

func CustomHttpServerInterceptor() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Before Request
		t := time.Now()
		counter := atomic.Int64{}
		counter.Store(0)
		zixelInvokerLevel := c.GetHeader("Zixel-Log-InvokerLevel")
		zixelResponseLevel := c.GetHeader("Zixel-Log-ResponseLevel")
		zixelRequestId := c.GetHeader("Zixel-Log-RequestId")
		zixelLogProtocol := "http-" + c.Request.Method
		zixelLogRequestIgnored := c.GetHeader("Zixel-Log-RequestIgnored")
		zixelLogResponseIgnored := c.GetHeader("Zixel-Log-ResponseIgnored")

		if zixelInvokerLevel == "" {
			zixelInvokerLevel = "01"
		}
		zixelResponseLevel = zixelInvokerLevel

		c.Request.Header.Set("Zixel-Log-InvokerLevel", zixelInvokerLevel)
		c.Request.Header.Set("Zixel-Log-ResponseLevel", zixelResponseLevel)
		c.Request.Header.Set("Zixel-Log-RequestId", zixelRequestId)
		c.Request.Header.Set("Zixel-Log-Protocol", zixelLogProtocol)
		c.Request.Header.Set("Zixel-Log-RequestIgnored", zixelLogRequestIgnored)
		c.Request.Header.Set("Zixel-Log-ResponseIgnored", zixelLogResponseIgnored)
		var params string
		var body string

		if zixelLogRequestIgnored != "true" {
			// Copy the request body so we can read it without consuming it
			bodyCopy := new(bytes.Buffer)
			_, err := io.Copy(bodyCopy, c.Request.Body)
			if err != nil {
				log.Printf("Failed to copy body: %v", err)
			}
			c.Request.Body = io.NopCloser(bodyCopy)
			params = c.Request.URL.Query().Encode()
			body = bodyCopy.String()
		} else {
			params = "##ignored##"
			body = "##ignored##"
		}
		writer := c.Writer
		responseRecorder := &responseRecorder{ResponseWriter: writer}
		c.Writer = responseRecorder
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), "counter", &counter))
		c.Next()
		go handleCustomHttpServerLog(c, t, &counter, responseRecorder, params, body)
	}
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	if r.body == nil {
		r.body = bytes.NewBufferString("")
	}
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}

func (s *WebServer) GetEngine() *gin.Engine {
	return s.eng
}

func (s *WebServer) Start() error {
	TrustedProxies := config.GetArray("web.trusted_proxies")
	if len(TrustedProxies) > 0 {
		if err := s.eng.SetTrustedProxies(xutil.Slice[string](TrustedProxies)); err != nil {
			return err
		}
	}

	listenPort := config.GetString("web.listen_port", "80")
	listenAddr := config.GetString("web.listen_addr", "localhost")

	s.web = &http.Server{
		Addr:           listenAddr + ":" + listenPort,
		Handler:        s.eng.Handler(),
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	/// 订阅退出事件，并调用Web server的退出函数
	exit := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := s.web.Shutdown(ctx); err != nil {
			s.bus.Publish("log.error", err.Error())
		}

		<-ctx.Done()
	}

	s.bus.Subscribe("ExitWebServer", exit)
	s.bus.Subscribe("Quit", exit)

	/// 启动web server
	go func() {
		var err error = nil
		s.bus.Publish("log.info", "Listening and serving HTTP on listener what's bind with address ", s.web.Addr)
		if err = s.web.ListenAndServe(); err != nil {
			s.bus.Publish("log.error", err.Error())
		}

		s.bus.Publish("log.info", "web server is closed.")
		s.bus.Publish("WebServerExited")
	}()

	engine := s.GetEngine()

	// Load fallback route, fallback from higher version to lower version
	engine.NoRoute(func(c *gin.Context) {
		re := regexp.MustCompile("/v([1-9]+)/")
		matches := re.FindStringSubmatch(c.Request.URL.Path)

		// If no version found, return
		if matches == nil {
			return
		}

		// Extract version number from path
		currentVersion, _ := strconv.Atoi(matches[1])
		currentVersion--

		for ; currentVersion >= 0; currentVersion-- {
			newPath := ""
			version := ""
			if currentVersion == 0 {
				newPath = strings.Replace(c.Request.URL.Path, matches[0], "/", 1)
				version = "no-version"
			} else {
				newPath = strings.Replace(c.Request.URL.Path, matches[0], "/v"+strconv.Itoa(currentVersion)+"/", 1)
				version = "v" + strconv.Itoa(currentVersion)
			}
			routeInfo := engine.Routes()

			for _, info := range routeInfo {
				if pathMatch(info.Path, newPath) {
					c.Request.URL.Path = newPath
					c.Writer.Header().Set("api-version", version)
					engine.HandleContext(c)
					return
				}
			}
		}
	})

	return nil
}

func (s *WebServer) Wait() error {
	q := make(chan int)
	/// 等待退出事件
	s.bus.Subscribe("WebServerExited", func() {
		q <- 0
	})

	exitCode := <-q
	s.bus.Publish("log.info", "Web server is closed. exit code = ", exitCode)

	return nil
}

func (s *WebServer) Stop() {
	s.bus.Publish("ExitWebServer")
}

// ----------------------------------------------------------
//
//	Rpc Server
//
// ----------------------------------------------------------

func NewRpcServer(bus *bus.AsyncEventBus) *RpcServer {
	var srv RpcServer
	srv.bus = bus
	srv.rpc = grpc.NewServer(
		grpc.UnaryInterceptor(CustomRpcServerInterceptor()),
	)
	if srv.rpc == nil {
		return nil
	}
	reflection.Register(srv.rpc)

	return &srv
}

func CustomRpcServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Before Request
		t := time.Now()

		counter := atomic.Int64{}
		counter.Store(0)
		ctx = context.WithValue(ctx, "counter", &counter)

		md, _ := metadata.FromIncomingContext(ctx)
		zixelInvokerLevel := md.Get("Zixel-Log-InvokerLevel")
		zixelResponseLevel := md.Get("Zixel-Log-ResponseLevel")
		zixelRequestId := md.Get("Zixel-Log-RequestId")
		zixelLogProtocol := "grpc"
		zixelLogRequestIgnored := md.Get("Zixel-Log-RequestIgnored")
		zixelLogResponseIgnored := md.Get("Zixel-Log-ResponseIgnored")

		var requestId string
		if zixelRequestId != nil {
			requestId = zixelRequestId[0]
		}

		var request interface{}
		if zixelLogRequestIgnored != nil {
			if zixelLogRequestIgnored[0] == "true" {
				// Convert request to base64 string
				request = "##ignored##"
			} else {
				requestBytes, err := proto.Marshal(req.(proto.Message))
				if err != nil {
					log.Fatal("Failed to marshal the gRPC request:", err)
				}
				base64String := base64.StdEncoding.EncodeToString(requestBytes)
				requestStruct := grpcstruct{
					TypeURL: info.FullMethod + "/" + reflect.TypeOf(req).String(),
					Value:   base64String,
					Version: "1",
				}
				request = requestStruct
			}
		} else {
			requestBytes, err := proto.Marshal(req.(proto.Message))
			if err != nil {
				log.Fatal("Failed to marshal the gRPC request:", err)
			}
			base64String := base64.StdEncoding.EncodeToString(requestBytes)
			requestStruct := grpcstruct{
				TypeURL: info.FullMethod + "/" + reflect.TypeOf(req).String(),
				Value:   base64String,
				Version: "1",
			}
			request = requestStruct
		}

		if zixelInvokerLevel == nil {
			zixelInvokerLevel = []string{"01"}
		}

		var keys []string
		var values []string

		// Update the level number for the next service
		if zixelInvokerLevel != nil {
			keys = append(keys, "Zixel-Log-InvokerLevel")
			values = append(values, zixelInvokerLevel[0])
		}

		if zixelResponseLevel != nil {
			keys = append(keys, "Zixel-Log-ResponseLevel")
			values = append(values, zixelResponseLevel[0])
		}

		keys = append(keys, "Zixel-Log-Protocol")
		values = append(values, zixelLogProtocol)

		if zixelRequestId != nil {
			keys = append(keys, "Zixel-Log-RequestId")
			values = append(values, zixelRequestId[0])
		}

		if zixelLogRequestIgnored != nil {
			keys = append(keys, "Zixel-Log-RequestIgnored")
			values = append(values, zixelLogRequestIgnored[0])
		}

		if zixelLogResponseIgnored != nil {
			keys = append(keys, "Zixel-Log-ResponseIgnored")
			values = append(values, zixelLogResponseIgnored[0])
		}

		var headers []string
		for i := 0; i < len(keys); i++ {
			headers = append(headers, keys[i], values[i])
		}
		pairs := metadata.Pairs(headers...)

		ctx = metadata.NewIncomingContext(ctx, pairs)

		resp, err := handler(ctx, req)

		if err != nil {
			// errCode = error code
			// errMessage = error message

			//{
			//    "error": {
			//        "service": {
			//            "name": "xdm",
			//            "uuid": "NoServiceUUID"
			//        },
			//        "code": 1000301,
			//        "message": "create new structure node failed:create structure node failed: rpc error: code = Internal desc = jumeaux-exception:ErrorDetail{code=80102, message='同层级下同类型已存在同名节点', serviceInfo=ServiceInfo{name='JumeauxStructure', uuid='UUID_from_k8s'}}"
			//    }
			//}

			// Transform into structure

		}
		// After request
		counter.Add(1)
		latency := float64(time.Since(t)) / float64(time.Millisecond)
		p, _ := peer.FromContext(ctx)
		callerIP := p.Addr.String()

		// Get the gRPC status code and message
		st, _ := grpcStatus.FromError(err)
		grpcCode := st.Code()
		grpcMessage := st.Message()
		if grpcMessage == "" {
			grpcMessage = "OK"
		}

		var response interface{}
		if zixelLogResponseIgnored != nil {
			if zixelLogResponseIgnored[0] == "true" {
				response = "##ignored##"
			} else {
				if respStringer, ok := resp.(fmt.Stringer); ok {
					reply := respStringer.String()
					base64String := base64.StdEncoding.EncodeToString([]byte(reply))
					requestStruct := grpcstruct{
						TypeURL: info.FullMethod + "/" + reflect.TypeOf(resp).String(),
						Value:   base64String,
						Version: "1",
					}
					response = requestStruct
				}
			}
		} else {
			if respStringer, ok := resp.(fmt.Stringer); ok {
				reply := respStringer.String()
				base64String := base64.StdEncoding.EncodeToString([]byte(reply))
				requestStruct := grpcstruct{
					TypeURL: info.FullMethod + "/" + reflect.TypeOf(resp).String(),
					Value:   base64String,
					Version: "1",
				}
				response = requestStruct
			}
		}

		fullMethod := info.FullMethod
		if strings.HasPrefix(fullMethod, "/") {
			fullMethod = fullMethod[1:]
		}
		// split fullMethod
		methods := strings.Split(fullMethod, "/")
		protoService := methods[0]
		protoMethod := methods[1]

		responseLevel := zixelInvokerLevel[0] + "." + fmt.Sprintf("%02d", counter.Load())

		logLine := fmt.Sprintf("[receive-log]%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%d|%s|%s\n",
			requestId,                            // RequestId
			t.Format("2006-01-02 15:04:05.000"),  // Timestamp
			"grpc",                               // Caller service name
			protoService,                         // Calling method
			protoMethod,                          // Calling method name
			fmt.Sprintf("%v", request),           // Caller params
			callerIP,                             // Caller IP
			zixelInvokerLevel[0],                 // Caller level number
			config.GetString("service.name", ""), // Responder service name
			info.FullMethod,                      // Responder method
			response,                             // Return parameters
			serverIp,                             // Responder IP
			responseLevel,                        // Responder level number
			grpcCode,                             // System return code
			grpcMessage,                          // System return message
			fmt.Sprintf("%.2fms", latency),       // Latency
		)

		fmt.Print(logLine)
		return resp, err
	}
}

func CustomRpcClientInterceptor(address string) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		// Before Request
		t := time.Now()

		counter := int64(1)
		if ctx.Value("counter") != nil {
			(*(ctx.Value("counter").(*atomic.Int64))).Add(1)
			counter = ctx.Value("counter").(*atomic.Int64).Load()
		}

		md, _ := metadata.FromIncomingContext(ctx)

		zixelInvokerLevel := md.Get("Zixel-Log-InvokerLevel")
		zixelResponseLevel := md.Get("Zixel-Log-ResponseLevel")
		zixelRequestId := md.Get("Zixel-Log-RequestId")
		zixelLogProtocol := "grpc"
		zixelLogRequestIgnored := md.Get("Zixel-Log-RequestIgnored")
		zixelLogResponseIgnored := md.Get("Zixel-Log-ResponseIgnored")

		var requestId string
		if zixelRequestId != nil {
			requestId = zixelRequestId[0]
		}

		var request interface{}
		if zixelLogRequestIgnored != nil {
			if zixelLogRequestIgnored[0] == "true" {
				request = "##ignored##"
			} else {
				// Convert the gRPC request to bytes
				requestBytes, err := proto.Marshal(req.(proto.Message))
				if err != nil {
					log.Fatal("Failed to marshal the gRPC request:", err)
				}
				base64String := base64.StdEncoding.EncodeToString(requestBytes)
				requestStruct := grpcstruct{
					TypeURL: method + "/" + reflect.TypeOf(req).String(),
					Value:   base64String,
					Version: "1",
				}
				request = requestStruct
			}
		} else { // Convert the gRPC request to bytes
			requestBytes, err := proto.Marshal(req.(proto.Message))
			if err != nil {
				log.Fatal("Failed to marshal the gRPC request:", err)
			}
			base64String := base64.StdEncoding.EncodeToString(requestBytes)
			requestStruct := grpcstruct{
				TypeURL: method + "/" + reflect.TypeOf(req).String(),
				Value:   base64String,
				Version: "1",
			}
			request = requestStruct
		}

		var callerLevel string
		var responseLevel string
		if zixelInvokerLevel != nil {
			callerLevel = zixelInvokerLevel[0]
			responseLevel = zixelInvokerLevel[0] + "." + fmt.Sprintf("%02d", counter)
			zixelInvokerLevel[0] = responseLevel
		} else {
			zixelInvokerLevel = []string{responseLevel}
		}
		if zixelResponseLevel != nil {
			zixelResponseLevel[0] = zixelInvokerLevel[0]
		} else {
			zixelResponseLevel = []string{zixelInvokerLevel[0]}
		}

		var keys []string
		var values []string

		// Update the level number for the next service
		if zixelInvokerLevel != nil {
			keys = append(keys, "Zixel-Log-InvokerLevel")
			values = append(values, zixelInvokerLevel[0])
		}

		if zixelResponseLevel != nil {
			keys = append(keys, "Zixel-Log-ResponseLevel")
			values = append(values, zixelResponseLevel[0])
		}

		keys = append(keys, "Zixel-Log-Protocol")
		values = append(values, zixelLogProtocol)

		if zixelRequestId != nil {
			keys = append(keys, "Zixel-Log-RequestId")
			values = append(values, zixelRequestId[0])
		}

		if zixelLogRequestIgnored != nil {
			keys = append(keys, "Zixel-Log-RequestIgnored")
			values = append(values, zixelLogRequestIgnored[0])
		}

		if zixelLogResponseIgnored != nil {
			keys = append(keys, "Zixel-Log-ResponseIgnored")
			values = append(values, zixelLogResponseIgnored[0])
		}

		var headers []string
		for i := 0; i < len(keys); i++ {
			headers = append(headers, keys[i], values[i])
		}

		pairs := metadata.Pairs(headers...)
		ctx = metadata.NewOutgoingContext(ctx, pairs)

		err := invoker(ctx, method, req, reply, cc, opts...)

		// After request
		latency := float64(time.Since(t)) / float64(time.Millisecond)

		// Get the gRPC status code and message
		st, _ := grpcStatus.FromError(err)
		grpcCode := st.Code()
		grpcMessage := st.Message()
		if grpcMessage == "" {
			grpcMessage = "OK"
		}
		var response interface{}
		if zixelLogResponseIgnored != nil {
			if zixelLogResponseIgnored[0] == "true" {
				response = "##ignored##"
			} else {
				if respStringer, ok := reply.(fmt.Stringer); ok {
					resp := respStringer.String()
					base64String := base64.StdEncoding.EncodeToString([]byte(resp))
					requestStruct := grpcstruct{
						TypeURL: method + "/" + reflect.TypeOf(reply).String(),
						Value:   base64String,
						Version: "1",
					}
					response = requestStruct
				}
			}
		} else {
			if respStringer, ok := reply.(fmt.Stringer); ok {
				resp := respStringer.String()
				base64String := base64.StdEncoding.EncodeToString([]byte(resp))
				requestStruct := grpcstruct{
					TypeURL: method + "/" + reflect.TypeOf(reply).String(),
					Value:   base64String,
					Version: "1",
				}
				response = requestStruct
			}
		}

		fullMethod := method
		if strings.HasPrefix(fullMethod, "/") {
			fullMethod = fullMethod[1:]
		}
		// split fullMethod
		methods := strings.Split(fullMethod, "/")
		protoService := methods[0]
		protoMethod := methods[1]

		logLine := fmt.Sprintf("[trace-log]%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%d|%s|%s\n",
			requestId,                            // RequestId
			t.Format("2006-01-02 15:04:05.000"),  // Timestamp
			"grpc",                               // Caller service name
			protoService,                         // Calling method
			protoMethod,                          // Calling method name
			fmt.Sprintf("%v", request),           // Caller params
			serverIp,                             // Caller IP
			callerLevel,                          // Caller level number
			config.GetString("service.name", ""), // Responder service name
			method,                               // Responder method
			fmt.Sprintf("%v", response),          // Return parameters
			address,                              // Responder IP
			responseLevel,                        // Responder level number
			grpcCode,                             // System return code
			grpcMessage,                          // System return message
			fmt.Sprintf("%.2fms", latency),       // Response time
		)

		fmt.Print(logLine)
		return err
	}
}

func (s *RpcServer) Start() error {
	// ----------------- Net (for gRPC) ----------------- //
	listenAddr := config.GetString("grpc.listen_addr", "localhost")
	listenPort := config.GetString("grpc.listen_port", "9400")

	//Creating the Server
	lis, err := net.Listen("tcp", listenAddr+":"+listenPort)
	if err != nil {
		s.bus.Publish("log.fatal", "failed to listen: ", err.Error())
	}

	exit := func() {
		s.bus.Publish("log.info", "grpc server is closeing...")
		s.rpc.GracefulStop()
		s.bus.Publish("log.info", "grpc server is closed.")
	}

	s.bus.Subscribe("ExitRpcServer", exit)
	s.bus.Subscribe("Quit", exit)

	go func() {
		if err := s.rpc.Serve(lis); err != nil {
			s.bus.Publish("log.fatal", "failed to listen: ", err.Error())
		}

		s.bus.Publish("RpcServerExited")
	}()

	return nil
}

func (s *RpcServer) Wait() error {
	q := make(chan int)
	/// 等待退出事件
	s.bus.Subscribe("RpcServerExited", func() {
		q <- 0
	})

	exitCode := <-q
	s.bus.Publish("log.info", "Rpc server is closed. exit code = ", exitCode)

	return nil
}

func (s *RpcServer) Stop() {
	s.bus.Publish("ExitRpcServer")
}

func pathMatch(pattern, path string) bool {
	re := regexp.MustCompile(":[^/]+")
	pattern = re.ReplaceAllString(pattern, "[^/]+")
	re = regexp.MustCompile(pattern)
	return re.MatchString(path)
}

func GetLocalIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP.String(), nil
}
