package framework

var (
	ERR_SYS_SERVER       = 1000
	ERR_SYS_DATABASE     = 1001
	ERR_SYS_PARAMETER    = 1002
	ERR_SYS_AUTH         = 1003
	ERR_SYS_PARSE_BODY   = 1004
	ERR_SYS_PARSE_URL    = 1005
	ERR_SYS_PARSE_PARAMS = 1006
	ERR_SYS_PARSE_HEADER = 1007
)

type ErrorCode struct {
	Code    int    //错误码
	Message string //错误信息
}
