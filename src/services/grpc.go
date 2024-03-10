package services

import (
	"google.golang.org/grpc"
	"messenger/configs"
)

var conns = make([]*grpc.ClientConn, 0, 10)

var UserBasicService = connectUserBasicService()
var UserService = connectUserService()
var UserServiceV2GRPC = connectUserServiceV2()
var ThirdAdapterService = connectThirdAdapterService()

func connectUserBasicService() UserBasicServiceClient {
	conn, err := grpc.Dial(configs.UserServiceAddr, grpc.WithInsecure())
	if err != nil {
		return nil
	}

	Service := NewUserBasicServiceClient(conn)

	conns = append(conns, conn)
	return Service

}

func connectUserService() UserServiceClient {
	conn, err := grpc.Dial(configs.UserServiceAddr, grpc.WithInsecure())
	if err != nil {
		return nil
	}

	Service := NewUserServiceClient(conn)

	conns = append(conns, conn)
	return Service

}

func connectUserServiceV2() UserServiceV2Client {
	conn, err := grpc.Dial(configs.UserServiceAddr, grpc.WithInsecure())
	if err != nil {
		return nil
	}

	Service := NewUserServiceV2Client(conn)
	conns = append(conns, conn)
	return Service
}

func connectZixelBusService() ProducerMsgServerClient {
	conn, err := grpc.Dial(configs.ZixelBusServiceAddr, grpc.WithInsecure())
	if err != nil {
		return nil
	}

	Service := NewProducerMsgServerClient(conn)

	conns = append(conns, conn)
	return Service
}

func connectThirdAdapterService() ThirdAdapterServerClient {
	conn, err := grpc.Dial(configs.ThirdAdapterServiceAddr, grpc.WithInsecure())
	if err != nil {
		return nil
	}

	Service := NewThirdAdapterServerClient(conn)

	conns = append(conns, conn)
	return Service
}

func ExitGrpc() {
	for _, conn := range conns {
		err := conn.Close()
		if err != nil {
			return
		}
	}
}
