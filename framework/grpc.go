package framework

import (
	"fmt"
	"time"

	"gitlab.zixel.cn/go/framework/config"
	"gitlab.zixel.cn/go/framework/variant"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var grpcConnMap map[string][]*grpc.ClientConn /// grpc client name mapping

type grpcLogger struct {
}

// Info logs to INFO log. Arguments are handled in the manner of fmt.Print.
func (l grpcLogger) Info(args ...interface{}) {
	log.Info(args...)
}

// Infoln logs to INFO log. Arguments are handled in the manner of fmt.Println.
func (l grpcLogger) Infoln(args ...interface{}) {
	log.Infoln(args...)
}

// Infof logs to INFO log. Arguments are handled in the manner of fmt.Printf.
func (l grpcLogger) Infof(format string, args ...interface{}) {
	log.Infof(format, args...)
}

// Warning logs to WARNING log. Arguments are handled in the manner of fmt.Print.
func (l grpcLogger) Warning(args ...interface{}) {
	log.Warning(args...)
}

// Warningln logs to WARNING log. Arguments are handled in the manner of fmt.Println.
func (l grpcLogger) Warningln(args ...interface{}) {
	log.Warningln(args...)
}

// Warningf logs to WARNING log. Arguments are handled in the manner of fmt.Printf.
func (l grpcLogger) Warningf(format string, args ...interface{}) {
	log.Warningf(format, args...)
}

// Error logs to ERROR log. Arguments are handled in the manner of fmt.Print.
func (l grpcLogger) Error(args ...interface{}) {
	log.Error(args...)
}

// Errorln logs to ERROR log. Arguments are handled in the manner of fmt.Println.
func (l grpcLogger) Errorln(args ...interface{}) {
	log.Errorln(args...)
}

// Errorf logs to ERROR log. Arguments are handled in the manner of fmt.Printf.
func (l grpcLogger) Errorf(format string, args ...interface{}) {
	log.Errorf(format, args...)
}

// Fatal logs to ERROR log. Arguments are handled in the manner of fmt.Print.
// gRPC ensures that all Fatal logs will exit with os.Exit(1).
// Implementations may also call os.Exit() with a non-zero exit code.
func (l grpcLogger) Fatal(args ...interface{}) {
	log.Fatal(args...)
}

// Fatalln logs to ERROR log. Arguments are handled in the manner of fmt.Println.
// gRPC ensures that all Fatal logs will exit with os.Exit(1).
// Implementations may also call os.Exit() with a non-zero exit code.
func (l grpcLogger) Fatalln(args ...interface{}) {
	log.Fatalln(args...)
}

// Fatalf logs to ERROR log. Arguments are handled in the manner of fmt.Printf.
// gRPC ensures that all Fatal logs will exit with os.Exit(1).
// Implementations may also call os.Exit() with a non-zero exit code.
func (l grpcLogger) Fatalf(format string, args ...interface{}) {
	log.Fatalf(format, args...)
}

// V reports whether verbosity level l is at least the requested verbose level.
func (l grpcLogger) V(level int) bool {
	return true
}

func GrpcCloseAll() {
	for k, v := range grpcConnMap {
		for _, conn := range v {
			log.Infof("close grpc conn:%v", k)
			conn.Close()
		}
	}
}

func ConnectGrpcServers() error {
	if grpcConnMap == nil {
		grpcConnMap = make(map[string][]*grpc.ClientConn)
	}
	conf := config.GetObject("grpc.connections")
	for k, v := range conf {
		log.Infof("building grpc conn for:%v:%v", k, v)
		val := variant.New(v)
		if val.IsString() {
			conn, err := grpc.Dial(val.ToString(), grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithUnaryInterceptor(CustomRpcClientInterceptor(val.ToString())))
			if err != nil {
				return err
			}

			if arr, ok := grpcConnMap[k]; ok {
				grpcConnMap[k] = append(arr, conn)
			} else {
				grpcConnMap[k] = []*grpc.ClientConn{conn}
			}
		} else if val.IsArray() {
			arr := val.ToArray()
			connArr := make([]*grpc.ClientConn, 0)
			for _, connStr := range arr {
				tmp := variant.New(connStr)
				if tmp.IsString() {
					conn, err := grpc.Dial(val.ToString(), grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithUnaryInterceptor(CustomRpcClientInterceptor(val.ToString())))
					if err != nil {
						return err
					}
					connArr = append(connArr, conn)
				} else {
					return fmt.Errorf("grpc_services has error, the array element is not a string")
				}
			}
			if arr, ok := grpcConnMap[k]; ok {
				grpcConnMap[k] = append(arr, connArr...)
			} else {
				grpcConnMap[k] = connArr
			}
		}
	}
	return nil
}

func GetGrpcConnection(key string) *grpc.ClientConn {
	if arr, ok := grpcConnMap[key]; ok && len(arr) > 0 {
		idx := int(time.Now().Unix()) % len(arr)
		return arr[idx]
	}

	return nil
}
