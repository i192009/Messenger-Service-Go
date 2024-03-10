package configs

// Import packages
import (
	"github.com/joho/godotenv"
	"gitlab.zixel.cn/go/framework/config"
	"os"
	"strings"
)

// Loading env file
var (
	_         = godotenv.Load()
	Env       = config.GetString("env", "offline")
	Namespace = config.GetString("namespace", "local")
	Hostname  = os.Getenv("HOSTNAME")

	EmailAccount  = config.GetString("email.account", "")
	EmailPassword = config.GetString("email.password", "")

	HuaweiApiAddress = config.GetString("huawei_sms_api.api_address", "")
	HuaweiAppKey     = config.GetString("huawei_sms_api.app_key", "")
	HuaweiAppSecret  = config.GetString("huawei_sms_api.app_secret", "")
	HuaweiChannel    = config.GetString("huawei_sms_api.channel", "")
	HuaweiSignature  = config.GetString("huawei_sms_api.signature", "")
	HuaweiTemplateId = config.GetString("huawei_sms_api.template_id", "")

	FeishuAppId     = config.GetString("feishu_api."+Namespace+".app_id", "")
	FeishuAppSecret = config.GetString("feishu_api."+Namespace+".app_secret", "")
	FeishuAppType   = config.GetString("feishu_api."+Namespace+".app_type", "")

	DingTalkSuiteId     = config.GetString("dingtalk_api."+Namespace+".suite_id", "")
	DingTalkAppId       = config.GetString("dingtalk_api."+Namespace+".app_id", "")
	DingTalkMiniAppId   = config.GetString("dingtalk_api."+Namespace+".mini_app_id", "")
	DingTalkSuiteKey    = config.GetString("dingtalk_api."+Namespace+".suite_key", "")
	DingTalkSuiteSecret = config.GetString("dingtalk_api."+Namespace+".suite_secret", "")
	DingTalkTemplateId  = config.GetString("dingtalk_api."+Namespace+".template_id", "")
	DingTalkAgentId     = config.GetString("dingtalk_api."+Namespace+".agent_id", "")

	SecretKey               = config.GetString("secret_key", "")
	UserServiceAddr         = config.GetString("user_service.grpc", "")
	ZixelBusServiceAddr     = config.GetString("zixel_bus_service.grpc", "")
	ThirdAdapterServiceAddr = config.GetString("third_adapter_service.grpc", "")
)

func IsReleaseMode() bool {
	return strings.ToLower(Env) == "online"
}

func IsDebugMode() bool {
	return strings.ToLower(Env) != "online"
}
