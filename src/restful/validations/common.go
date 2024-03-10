package validations

import (
	pbStruct "github.com/golang/protobuf/ptypes/struct"
	"gitlab.zixel.cn/go/framework/logger"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"messenger/configs"
	"messenger/services"
)

var log = logger.Get()

// CreateError : This function creates a new error
func createError(code codes.Code, errorCode int, args ...map[string]string) error {
	message := configs.ErrorCodes[errorCode]
	err := status.Newf(
		code,
		message,
	)

	var arguments *structpb.Struct
	if args != nil {
		arguments = convertMapToStruct(args[0])
	}
	err, wde := err.WithDetails(
		&services.ErrorResponse{
			Error: &services.Error{
				Code:    int64(errorCode),
				Message: message,
				Service: &services.Service{
					Name: "Messenger Service",
					Uuid: "Development",
				},
				Args: arguments,
			},
		})
	if wde != nil {
		return wde
	}
	return err.Err()
}

// convertMapToStruct : This function converts map[string]string to struct
func convertMapToStruct(stringMap map[string]string) *pbStruct.Struct {
	fields := make(map[string]*pbStruct.Value, len(stringMap))
	for k, v := range stringMap {
		fields[k] = &pbStruct.Value{
			Kind: &pbStruct.Value_StringValue{
				StringValue: v,
			},
		}
	}
	return &pbStruct.Struct{
		Fields: fields,
	}

}
