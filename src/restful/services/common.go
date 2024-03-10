package services

// Importing packages
import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"messenger/restful/models"
	"reflect"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/rlog"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	pbStruct "github.com/golang/protobuf/ptypes/struct"
	"github.com/gorilla/websocket"
	"gitlab.zixel.cn/go/framework/database"
	"gitlab.zixel.cn/go/framework/k8smanager"
	"gitlab.zixel.cn/go/framework/logger"

	"io"
	"messenger/configs"
	"messenger/services"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	rocketMqPrimitive "github.com/apache/rocketmq-client-go/v2/primitive"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/exp/slices"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

// Initializing MongoDB collections

var Log = logger.Get()

var configurationCollection = database.GetCollection("configurations")
var templateCollection = database.GetCollection("templates")
var messageCollection = database.GetCollection("messages")
var sceneCollection = database.GetCollection("scenes")
var sceneRelationCollection = database.GetCollection("sceneRelations")

// Redis Keys
var redisMessageKey = "message:"
var redisUserMessagesKey = "user-messages:"
var redisUserListKey = "user-list:"
var redisFeishuApplicationsKey = "feishu-applications:"
var redisDingTalkApplicationsKey = "dingtalk-applications:"
var phoneRegex = "^\\s*(?:\\+?(\\d{1,3}))?[-. (]*(\\d{3})[-. )]*(\\d{3})[-. ]*(\\d{4})(?: *x(\\d+))?\\s*$"

var manager = k8smanager.NewK8sClientManager()

const (
	AddConfigurationAction    = "add-configuration"
	DeleteConfigurationAction = "delete-configuration"
	AddToBlacklistAction      = "add-to-blacklist"
	RemoveFromBlacklistAction = "remove-from-blacklist"
	AddToWhitelistAction      = "add-to-whitelist"
	RemoveFromWhitelistAction = "remove-from-whitelist"
	ChangeFilteringModeAction = "change-filtering-mode"
	ErrorAction               = "error"
	SuccessAction             = "success"
)

// Initializing the websocket upgrader
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Variables for websocket
var (
	newline = []byte{'\n'}
)

// Constants for websocket
const (
	// Max wait time when writing message to peer
	writeWait = 10 * time.Second

	// Max time till next pong from peer
	pongWait = 60 * time.Second

	// Send ping interval, must be less than pong wait time
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 10000
)

// Initializing validator to validate request body
var validate = validator.New()

// ValidateJSON : This function validates the request body against the given schema
func ValidateJSON(model interface{}) error {
	// Returning error and logging if request body is not valid
	if err := validate.Struct(model); err != nil {
		Log.Error(err.Error())
		split := strings.Split(err.Error(), "\n")
		validationErrors := make(map[string]string)
		for _, s := range split {
			subSplit := strings.Split(s, " Error:")
			key := strings.Replace(strings.Split(subSplit[0], ".")[1], "'", "", -1)
			val := subSplit[1]
			validationErrors[key] = val
		}

		return CreateError(codes.InvalidArgument, 100002, validationErrors)
	}
	return nil
}

// BindJSON : This function binds the request body to the given model
func BindJSON(c *gin.Context, model interface{}) error {
	// Returning error and logging if request body is not valid
	if err := c.BindJSON(model); err != nil {
		Log.Error(err.Error())
		return CreateError(codes.InvalidArgument, 100001)
	}
	return nil
}

// MarshalBinary : This function is used to marshal the message struct to binary
func MarshalBinary(any interface{}) ([]byte, error) {
	return json.Marshal(any)
}

var infiniteDuration = time.Duration(0)

// ConvertToObjectId : This function converts string to ObjectID
func ConvertToObjectId(id string) (primitive.ObjectID, error) {
	// Convert id string to ObjectId
	objectId, err := primitive.ObjectIDFromHex(id)

	// Returning error and logging if id is not valid
	if err != nil {
		Log.Error(err.Error())
		return [12]byte{}, CreateError(codes.InvalidArgument, 100003)
	}
	return objectId, nil
}

// ConvertMapToStruct : This function converts map[string]string to struct
func ConvertMapToStruct(stringMap map[string]string) *pbStruct.Struct {
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

// CreateError : This function creates a new error
func CreateError(code codes.Code, errorCode int, args ...map[string]string) error {
	message := configs.ErrorCodes[errorCode]
	err := status.Newf(
		code,
		message,
	)

	var arguments *structpb.Struct
	if args != nil {
		arguments = ConvertMapToStruct(args[0])
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

func contains(array []string, element string) bool {
	return slices.Contains(array, element)
}

func getUserContact(ctx context.Context, uid string) (*services.UserContactReply, error) {
	if configs.IsReleaseMode() {
		userInfo, err := services.UserBasicService.GetUserContact(ctx, &services.GetUserContactRequest{Uid: uid})
		if err != nil {
			Log.Error(err.Error())
			return &services.UserContactReply{}, err
		} else {
			return userInfo, nil
		}
	} else {
		return &services.UserContactReply{}, fmt.Errorf(" UserBasicService is not available in development mode")
	}
}

func getUserInfoV2(ctx context.Context, openId string, infoSelector int32) (*services.S2C_UserInfoGetV2_Rpn, error) {
	if configs.IsReleaseMode() {
		userInfo, err := services.UserServiceV2GRPC.GetUserInfo(ctx, &services.C2S_UserInfoGetV2_Req{
			Id:           openId,
			InfoType:     3,
			InfoSelector: infoSelector,
		})
		if err != nil {
			Log.Error(err.Error())
			return &services.S2C_UserInfoGetV2_Rpn{}, err
		} else {
			return userInfo, nil
		}
	} else {
		return &services.S2C_UserInfoGetV2_Rpn{}, fmt.Errorf(" UserBasicService is not available in development mode")
	}
}

//加密过程：
//  1、处理数据，对数据进行填充，采用PKCS7（当密钥长度不够时，缺几位补几个几）的方式。
//  2、对数据进行加密，采用AES加密方法中CBC加密模式
//  3、对得到的加密数据，进行base64加密，得到字符串
// 解密过程相反

//16,24,32位字符串的话，分别对应AES-128，AES-192，AES-256 加密方法

// pkcs7Padding 填充
func pkcs7Padding(data []byte, blockSize int) []byte {
	//判断缺少几位长度。最少1，最多 blockSize
	padding := blockSize - len(data)%blockSize
	//补足位数。把切片[]byte{byte(padding)}复制padding个
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padText...)
}

// pkcs7UnPadding 填充的反向操作
func pkcs7UnPadding(data []byte) ([]byte, error) {
	length := len(data)
	if length == 0 {
		return nil, errors.New("加密字符串错误！")
	}
	//获取填充的个数
	unPadding := int(data[length-1])
	return data[:(length - unPadding)], nil
}

// AesEncrypt 加密
func AES_Encrypt(data []byte, key []byte) ([]byte, error) {
	//创建加密实例
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	//判断加密快的大小
	blockSize := block.BlockSize()
	//填充
	encryptBytes := pkcs7Padding(data, blockSize)
	//初始化加密数据接收切片
	crypted := make([]byte, len(encryptBytes))
	//使用cbc加密模式
	blockMode := cipher.NewCBCEncrypter(block, key[:blockSize])
	//执行加密
	blockMode.CryptBlocks(crypted, encryptBytes)
	return crypted, nil
}

// AesDecrypt 解密
func AES_Decrypt(data []byte, key []byte) (res []byte, err error) {
	defer func() {
		if e := recover(); e != nil {
			print(err)
			err = errors.New("decrypt failed")
		}
	}()

	//创建实例
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	//获取块的大小
	blockSize := block.BlockSize()
	//使用cbc
	blockMode := cipher.NewCBCDecrypter(block, key[:blockSize])
	//初始化解密数据接收切片
	crypted := make([]byte, len(data))

	//执行解密
	blockMode.CryptBlocks(crypted, data)
	//去除填充
	crypted, err = pkcs7UnPadding(crypted)
	if err != nil {
		return nil, err
	}
	return crypted, nil
}

// AESEncryptBase64 Aes加密 后 base64 再加
func AES_EncryptBase64(data []byte, key []byte) (string, error) {
	res, err := AES_Encrypt(data, key)
	if err != nil {
		Log.Error(err.Error())
		return "", CreateError(codes.Internal, 1000)
	}

	return base64.StdEncoding.EncodeToString(res), nil
}

// AESDecryptBase64 base64 解码后再 Aes 解密
func AES_DecryptBase64(data string, key []byte) ([]byte, error) {
	dataByte, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		Log.Error(err.Error())
		return nil, CreateError(codes.Internal, 1000)
	}

	return AES_Decrypt(dataByte, key)
}

func GetDingTalkAuthInfo(suiteKey, suiteSecret, suiteTicket, authCorpId string) (string, error) {

	// Create Signature String
	timestamp := time.Now().UnixNano() / 1e6
	signatureString := fmt.Sprintf("%d\n%s", timestamp, suiteTicket)
	// Create Signature
	h := hmac.New(sha256.New, []byte(suiteSecret))
	h.Write([]byte(signatureString))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))

	query := url.Values{}
	query.Add("accessKey", suiteKey)
	query.Add("timestamp", strconv.FormatInt(timestamp, 10))
	query.Add("suiteTicket", suiteTicket)
	query.Add("signature", signature)

	// Get DingTalk AppAccessToken
	reqUrl := url.URL{
		Scheme:   "https",
		Host:     "oapi.dingtalk.com",
		Path:     "service/get_auth_info",
		RawQuery: query.Encode(),
	}

	// Request body
	body, _ := json.Marshal(
		gin.H{
			"auth_corpid": authCorpId,
			"suite_key":   suiteKey,
		})
	requestBody := bytes.NewBuffer(body)

	// Create Request
	req, err := http.NewRequest("POST", reqUrl.String(), requestBody)
	if err != nil {
		Log.Error(err.Error())
		return "", CreateError(codes.Internal, 1000, map[string]string{
			"error": err.Error(),
		})
	}

	// Send Request
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			Log.Fatal(err)
		}
	}(resp.Body)

	//Read the response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		Log.Error(err.Error())
		return "", CreateError(codes.Internal, 1000, map[string]string{
			"error": err.Error(),
		})
	}

	// Unmarshal the response body
	var data map[string]interface{}
	err = json.Unmarshal(respBody, &data)

	if err != nil {
		Log.Error(err.Error())
		return "", CreateError(codes.Internal, 1000, map[string]string{
			"error": err.Error(),
		})
	}

	if data["errcode"] != nil && data["errcode"].(float64) != 0 {
		Log.Error(data["errmsg"].(string))
		return "", CreateError(codes.Internal, 1000, map[string]string{
			"error": data["errmsg"].(string),
		})
	} else {
		if AuthInfo, ok := data["auth_info"]; ok {
			// convert interface to map
			AuthInfoMap := AuthInfo.(map[string]interface{})
			if agent, ok := AuthInfoMap["agent"]; ok {
				agentMap := agent.([]interface{})
				for _, v := range agentMap {
					if agentId, ok := v.(map[string]interface{})["agentid"]; ok {
						// convert agentId from float64 to string
						agentId := strconv.FormatFloat(agentId.(float64), 'f', -1, 64)
						Log.Info("AgentId retrieved successfully")
						return agentId, nil
					}
				}
			}
		} else {
			Log.Error(data)
			return "", CreateError(codes.Internal, 1000)
		}
	}
	return "", CreateError(codes.Internal, 1000)
}

func GetDingTalkUserIdByUnionId(accessToken, unionId string) (string, error) {

	query := url.Values{}
	query.Add("access_token", accessToken)
	// Get DingTalk AppAccessToken
	reqUrl := url.URL{
		Scheme:   "https",
		Host:     "oapi.dingtalk.com",
		Path:     "/topapi/user/getbyunionid",
		RawQuery: query.Encode(),
	}

	// Request body
	body, _ := json.Marshal(
		gin.H{
			"unionid": unionId,
		})
	requestBody := bytes.NewBuffer(body)

	// Create Request
	req, err := http.NewRequest("POST", reqUrl.String(), requestBody)
	if err != nil {
		Log.Error(err.Error())
		return "", CreateError(codes.Internal, 1000)
	}

	// Send Request
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			Log.Fatal(err)
		}
	}(resp.Body)

	//Read the response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		Log.Error(err.Error())
		return "", CreateError(codes.Internal, 1000)
	}

	// Unmarshal the response body
	var data map[string]interface{}
	err = json.Unmarshal(respBody, &data)

	if err != nil {
		Log.Error(err.Error())
		return "", CreateError(codes.Internal, 1000)
	}

	Log.Info(data)
	if data["errcode"] != nil && data["errcode"].(float64) != 0 {
		return "", CreateError(codes.Internal, 1000)
	} else {
		if result, ok := data["result"]; ok {
			if userId, ok := result.(map[string]interface{})["userid"]; ok {
				Log.Debug(userId.(string))
				return userId.(string), nil
			} else {
				Log.Error(data)
				return "", CreateError(codes.Internal, 1000)
			}
		} else {
			Log.Error(data)
			return "", CreateError(codes.Internal, 1000)
		}
	}
}

func SubscribeThreePartMessage() {

	rlog.SetLogLevel("ERROR")
	Log.Infof("SubscribeThreePartMessage  init-------------\n")

	// 订阅主题、消费
	endPoint := []string{"http://rocketmq-svc:9876"}

	// 创建一个consumer实例
	c, err := rocketmq.NewPushConsumer(consumer.WithNsResolver(rocketMqPrimitive.NewPassthroughResolver(endPoint)),
		consumer.WithConsumerModel(consumer.Clustering),
		consumer.WithGroupName("messengerConsumer"),
	)

	if err != nil {
		Log.Errorf("rocketmq.NewPushConsumer is error.err : %v \n", err)
		return
	}
	// 订阅topic
	err = c.Subscribe("ThreePart", consumer.MessageSelector{},
		func(ctx context.Context, messages ...*rocketMqPrimitive.MessageExt) (consumer.ConsumeResult, error) {
			for _, msg := range messages {
				var data map[string]interface{}
				err := json.Unmarshal(msg.Body, &data)
				if err != nil {
					Log.Error(err.Error())
					return consumer.ConsumeRetryLater, nil
				} else {
					if from, ok := data["from"]; ok {
						if reflect.TypeOf(from) == nil {
							Log.Error("from is nil")
							return consumer.ConsumeSuccess, nil
						}
						if data["from"].(string) == "feishu" {
							if eventType, ok := data["eventType"]; ok {
								if reflect.TypeOf(eventType) == nil {
									Log.Error("eventType is nil")
									return consumer.ConsumeSuccess, nil
								}
								if data["eventType"].(string) == "app_ticket" {
									if appId, ok := data["appId"]; !ok {
										Log.Error("appId is empty")
										return consumer.ConsumeSuccess, nil
									} else {
										if reflect.TypeOf(appId) == nil {
											Log.Error("appId is nil")
											return consumer.ConsumeSuccess, nil
										}
									}

									val, err := database.RedisGetString(redisFeishuApplicationsKey + data["appId"].(string))
									if err != nil {
										Log.Error(err.Error())
										return consumer.ConsumeSuccess, nil
									} else {
										var feishuConfigData models.FeishuConfigData
										err = json.Unmarshal([]byte(*val), &feishuConfigData)
										if err != nil {
											Log.Error(err.Error())
											return consumer.ConsumeRetryLater, nil
										} else {
											appTicket, ok := data["appTicket"]
											if !ok {
												Log.Error("appTicket is empty")
												return consumer.ConsumeSuccess, nil
											}
											if reflect.TypeOf(appTicket) == nil {
												Log.Error("appTicket is nil")
												return consumer.ConsumeSuccess, nil
											}
											if feishuConfigData.AppTicket == data["appTicket"].(string) {
												Log.Error("appTicket is same as before")
												return consumer.ConsumeSuccess, nil
											}
											feishuConfigData.AppTicket = data["appTicket"].(string)
											binary, err := MarshalBinary(feishuConfigData)
											if err != nil {
												Log.Error(err.Error())
												return consumer.ConsumeRetryLater, nil
											} else {
												err := database.RedisSet(redisFeishuApplicationsKey+feishuConfigData.AppId, binary, &infiniteDuration)
												if err != nil {
													Log.Error(err.Error())
													return consumer.ConsumeRetryLater, nil
												}
											}
										}
									}
								} else if data["eventType"].(string) == "app_open" || data["eventType"].(string) == "app_status_change" {
									if appId, ok := data["app_id"]; !ok {
										Log.Error("appId is not exist")
										return consumer.ConsumeSuccess, nil
									} else {
										if reflect.TypeOf(appId) == nil {
											Log.Error("appId is nil")
											return consumer.ConsumeSuccess, nil
										}
									}

									val, err := database.RedisGetString(redisFeishuApplicationsKey + data["app_id"].(string))
									if err != nil {
										Log.Error(err.Error())
										return consumer.ConsumeSuccess, nil
									} else {
										var feishuConfigData models.FeishuConfigData
										err = json.Unmarshal([]byte(*val), &feishuConfigData)
										if err != nil {
											Log.Error(err.Error())
											return consumer.ConsumeRetryLater, nil
										} else {
											if companyId, ok := data["companyId"]; !ok {
												Log.Error("companyId is not exist")
												return consumer.ConsumeSuccess, nil
											} else {
												if reflect.TypeOf(companyId) == nil {
													Log.Error("companyId is nil")
													return consumer.ConsumeSuccess, nil
												}
											}

											tenantKey, ok := data["companyId"].(string)
											if ok {
												feishuConfigData.TenantKey = tenantKey

												binary, err := MarshalBinary(feishuConfigData)
												if err != nil {
													Log.Error(err.Error())
													return consumer.ConsumeRetryLater, nil
												} else {
													err := database.RedisSet(redisFeishuApplicationsKey+feishuConfigData.AppId, binary, &infiniteDuration)
													if err != nil {
														Log.Error(err.Error())
														return consumer.ConsumeRetryLater, nil
													}
												}
											} else {
												return consumer.ConsumeSuccess, nil
											}
										}
									}
								}
							}
						} else if data["from"].(string) == "dingtalk" {
							if eventType, ok := data["eventType"]; ok {
								if reflect.TypeOf(eventType) == nil {
									Log.Error("eventType is nil")
									return consumer.ConsumeSuccess, nil
								}
								if data["eventType"].(string) == "app_ticket" {
									var dingTalkConfigData models.DingTalkConfigData
									if appId, ok := data["appId"]; !ok {
										Log.Error("appId is not exist")
										return consumer.ConsumeSuccess, nil
									} else {
										if reflect.TypeOf(appId) == nil {
											Log.Error("appId is nil")
											return consumer.ConsumeSuccess, nil
										}
										val, err := database.RedisGetString(redisDingTalkApplicationsKey + data["appId"].(string))
										if err != nil {
											Log.Error(err.Error())
											return consumer.ConsumeSuccess, nil
										}
										err = json.Unmarshal([]byte(*val), &dingTalkConfigData)
										if err != nil {
											Log.Error(err.Error())
											return consumer.ConsumeRetryLater, nil
										}
									}
									if suiteTicket, ok := data["suiteTicket"]; !ok {
										Log.Error("suiteTicket is missing")
										return consumer.ConsumeSuccess, nil
									} else {
										if reflect.TypeOf(suiteTicket) == nil {
											Log.Error("suiteTicket is nil")
											return consumer.ConsumeSuccess, nil
										}
										if data["suiteTicket"].(string) != dingTalkConfigData.SuiteTicket {
											dingTalkConfigData.SuiteTicket = data["suiteTicket"].(string)
											binary, err := MarshalBinary(dingTalkConfigData)
											if err != nil {
												Log.Error(err.Error())
												return consumer.ConsumeRetryLater, nil
											} else {
												err := database.RedisSet(redisDingTalkApplicationsKey+dingTalkConfigData.AppId, binary, &infiniteDuration)
												if err != nil {
													Log.Error(err.Error())
													return consumer.ConsumeRetryLater, nil
												}
											}
										}
									}

									if companyId, ok := data["companyId"]; !ok {
										Log.Error("companyId is not exist")
										return consumer.ConsumeSuccess, nil
									} else {
										if reflect.TypeOf(companyId) == nil {
											Log.Error("companyId is nil")
											return consumer.ConsumeSuccess, nil
										}
										if data["companyId"].(string) != dingTalkConfigData.AuthCorpId {
											dingTalkConfigData.AuthCorpId = data["companyId"].(string)
											binary, err := MarshalBinary(dingTalkConfigData)
											if err != nil {
												Log.Error(err.Error())
												return consumer.ConsumeRetryLater, nil
											} else {
												err := database.RedisSet(redisDingTalkApplicationsKey+dingTalkConfigData.AppId, binary, &infiniteDuration)
												if err != nil {
													Log.Error(err.Error())
													return consumer.ConsumeRetryLater, nil
												}
											}
										}
									}
									if corpToken, ok := data["corpToken"]; !ok {
										Log.Error("corpToken is not exist")
										return consumer.ConsumeSuccess, nil
									} else {
										if reflect.TypeOf(corpToken) == nil {
											Log.Error("corpToken is nil")
											return consumer.ConsumeSuccess, nil
										}
										if data["corpToken"].(string) != dingTalkConfigData.AccessToken {
											dingTalkConfigData.AccessToken = data["corpToken"].(string)
											binary, err := MarshalBinary(dingTalkConfigData)
											if err != nil {
												Log.Error(err.Error())
												return consumer.ConsumeRetryLater, nil
											} else {
												err := database.RedisSet(redisDingTalkApplicationsKey+dingTalkConfigData.AppId, binary, &infiniteDuration)
												if err != nil {
													Log.Error(err.Error())
													return consumer.ConsumeRetryLater, nil
												}
											}
										}
									}

									if dingTalkConfigData.AuthCorpId != "" && dingTalkConfigData.AccessToken != "" {
										agentId, err := GetDingTalkAuthInfo(dingTalkConfigData.SuiteKey, dingTalkConfigData.SuiteSecret, dingTalkConfigData.SuiteTicket, dingTalkConfigData.AuthCorpId)
										if err != nil {
											Log.Error(err.Error())
											return consumer.ConsumeSuccess, nil
										} else {
											dingTalkConfigData.AgentId = agentId

											binary, err := MarshalBinary(dingTalkConfigData)
											if err != nil {
												Log.Error(err.Error())
												return consumer.ConsumeRetryLater, nil
											} else {
												err := database.RedisSet(redisDingTalkApplicationsKey+dingTalkConfigData.AppId, binary, &infiniteDuration)
												if err != nil {
													Log.Error(err.Error())
													return consumer.ConsumeRetryLater, nil
												}
											}
										}
									}
								}
							}
						} else {
							return consumer.ConsumeRetryLater, nil
						}
					} else {
						return consumer.ConsumeRetryLater, nil
					}
				}
			}

			return consumer.ConsumeSuccess, nil
		})

	if err != nil {
		Log.Errorf("subscribe message error: %s\n", err.Error())
	}

	// 启动consumer
	err = c.Start()
	Log.Debug("Subcribe Message  start-------------\n")
	if err != nil {
		Log.Errorf("subscribe Start error: %s\n", err.Error())
	}
	defer func(c rocketmq.PushConsumer) {
		err := c.Shutdown()
		if err != nil {
			Log.Errorf("subscribe Shutdown error: %s\n", err.Error())
		}
	}(c)
	select {}
}
