package services

import (
	"context"
	"github.com/google/uuid"
	"gitlab.zixel.cn/go/framework/database"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc/codes"
	"messenger/configs"
	"messenger/restful/models"
	pb "messenger/services"
	"net/http"
	"strings"
	"time"
)

// Validation arrays
var messageTypes = []string{"shared", "broadcast", "exclusive", "linkage"}
var messageModes = []string{"push", "pull", "third-party", "email", "sms", "feishu", "dingtalk", "listening"}
var httpMethods = []string{"get", "post", "put", "delete"}
var actionTypes = []string{"http"}

// Configuration : Model of message class configuration
type Configuration struct {
	Id                          primitive.ObjectID                 `json:"_id" bson:"_id,omitempty"`
	ClassId                     string                             `json:"classId" validate:"required" bson:"classId"`
	ParentId                    string                             `json:"parentId" validate:"required" bson:"parentId"`
	AppId                       string                             `json:"appId" validate:"required" bson:"appId"`
	Name                        string                             `json:"name" validate:"required" bson:"name"`
	Persist                     bool                               `json:"persist" bson:"persist"`
	States                      []string                           `json:"states" validate:"required" bson:"states"`
	Type                        string                             `json:"type" validate:"required,lowercase" bson:"type"`
	Mode                        string                             `json:"mode" validate:"required,lowercase" bson:"mode"`
	StatusCallbacks             models.Policy                      `json:"statusCallbacks" validate:"required" bson:"statusCallbacks"`
	Actions                     []models.ActionModel               `json:"actions,omitempty" validate:"dive" bson:"actions,omitempty"`
	CustomEmailConfiguration    models.CustomEmailConfiguration    `json:"customEmailConfiguration,omitempty" bson:"customEmailConfiguration,omitempty"`
	CustomSmsConfiguration      models.CustomSmsConfiguration      `json:"customSmsConfiguration,omitempty" bson:"customSmsConfiguration,omitempty"`
	CustomFeishuConfiguration   models.CustomFeishuConfiguration   `json:"customFeishuConfiguration,omitempty" bson:"customFeishuConfiguration,omitempty"`
	CustomDingTalkConfiguration models.CustomDingTalkConfiguration `json:"customDingTalkConfiguration,omitempty" bson:"customDingTalkConfiguration,omitempty"`
	CreatedAt                   time.Time                          `json:"createdAt" bson:"createdAt"`
	UpdatedAt                   time.Time                          `json:"updatedAt" bson:"updatedAt"`
}

func (configuration *Configuration) NewConfiguration(request *models.CreateConfigurationRequest) {
	// Setting default values for request
	configuration.Id = primitive.NewObjectID()
	configuration.ClassId = request.ClassId
	configuration.ParentId = request.ParentId
	configuration.AppId = request.AppId
	configuration.Name = request.Name
	configuration.Persist = request.Persist
	configuration.States = request.States
	configuration.Type = request.Type
	configuration.Mode = request.Mode
	configuration.StatusCallbacks = request.StatusCallbacks
	configuration.Actions = request.Actions
	configuration.CustomEmailConfiguration = request.CustomEmailConfiguration
	configuration.CustomSmsConfiguration = request.CustomSmsConfiguration
	configuration.CustomFeishuConfiguration = request.CustomFeishuConfiguration
	configuration.CustomDingTalkConfiguration = request.CustomDingTalkConfiguration
	configuration.CreatedAt = time.Now()
	configuration.UpdatedAt = time.Now()
}

// CreateConfiguration : Calls required functions to create a message class configuration
func CreateConfiguration(ctx context.Context, request *models.CreateConfigurationRequest) (*Configuration, error) {
	var configuration Configuration
	// Returning error and logging if failed to validate request JSON
	if err := ValidateJSON(request); err != nil {
		return nil, err
	}

	// Returning error and logging if failed to validate request body
	if err := validateCreateConfiguration(ctx, request, &configuration); err != nil {
		return nil, err
	}

	// Returning error and logging if failed to create configuration
	if err := createConfiguration(ctx, &configuration); err != nil {
		return nil, err
	}

	// Returning nil if configuration is created successfully
	return &configuration, nil
}

// UpdateConfiguration : Calls required functions to update a message class configuration
func UpdateConfiguration(ctx context.Context, request *models.UpdateConfigurationRequest, id string) (*Configuration, error) {
	// Initializing request model
	var configuration Configuration

	// Returning error and logging if failed to get request
	if err := getConfigurationByClassId(ctx, id, &configuration); err != nil {
		return nil, err
	}

	// Returning error and logging if failed to validate request JSON
	if err := ValidateJSON(request); err != nil {
		return nil, err
	}

	// Returning error and logging if failed to validate request body
	if err := validateUpdateConfiguration(ctx, request, &configuration); err != nil {
		return nil, err
	}

	// Returning error and logging if failed to update request
	if err := updateConfiguration(ctx, &configuration); err != nil {
		return nil, err
	}

	// Returning nil if request is updated successfully
	return &configuration, nil
}

// DeleteConfiguration : Calls required functions to delete a message class configuration
func DeleteConfiguration(ctx context.Context, id string) error {
	// Initializing configuration model
	var find Configuration

	// Returning error and logging if failed to get configuration
	if err := getConfigurationByClassId(ctx, id, &find); err != nil {
		return err
	}

	// Returning error and logging if failed to delete configuration
	if err := deleteConfiguration(ctx, &find); err != nil {
		return err
	}

	// Returning nil if configuration is deleted successfully
	return nil
}

// QueryConfigurations : Calls required functions to query message class configurations
func QueryConfigurations(ctx context.Context, query *models.QueryConfigurationsParams) (*pb.QueryConfigurationResponse, error) {
	// Returning error and logging if failed to query queryConfigurationResponse
	queryConfigurationResponse, err := queryConfigurations(ctx, query)
	if err != nil {
		return nil, err
	}

	// Converting queryConfigurationResponse to protobuf message
	response := make([]*pb.Configuration, len(queryConfigurationResponse.Results.([]Configuration)))
	for i, configuration := range queryConfigurationResponse.Results.([]Configuration) {
		responseActions := make([]*pb.Action, len(configuration.Actions))
		for i, action := range configuration.Actions {
			responseActions[i] = &pb.Action{
				Name: action.Name,
				Tips: action.Tips,
				Action: &pb.ActionTrigger{
					Type:   action.Action.Type,
					Url:    action.Action.URL,
					Method: action.Action.Method,
					Body:   action.Action.Body,
					Query:  action.Action.Query,
				},
				NextAction: action.NextAction,
			}
		}
		response[i] = &pb.Configuration{
			ClassId:  configuration.ClassId,
			ParentId: configuration.ParentId,
			AppId:    configuration.AppId,
			Name:     configuration.Name,
			Persist:  configuration.Persist,
			States:   configuration.States,
			Type:     configuration.Type,
			Mode:     configuration.Mode,
			StatusCallback: &pb.Policy{
				Kafka: configuration.StatusCallbacks.Kafka,
				Http:  configuration.StatusCallbacks.Http,
			},
			Actions:   responseActions,
			CreatedAt: configuration.CreatedAt.Format(time.RFC3339),
			UpdatedAt: configuration.UpdatedAt.Format(time.RFC3339),
		}
	}

	// Returning queryConfigurationResponse if successfully queried
	return &pb.QueryConfigurationResponse{
		Page:    query.Page,
		Limit:   query.Limit,
		Total:   queryConfigurationResponse.Total,
		Results: response,
	}, nil
}

// GetConfiguration : Calls required functions to get a message class configuration
func GetConfiguration(ctx context.Context, id string) (*pb.Configuration, error) {
	// Initializing configuration model
	var find Configuration

	// Returning error and logging if failed to get configuration
	if err := validateGetConfiguration(ctx, id, &find); err != nil {
		return nil, err
	}

	// Returning error and logging if failed to get configuration
	configuration, err := getConfiguration(ctx, id, &find)
	if err != nil {
		return nil, err
	}

	// Returning configuration if successfully queried
	return configuration, nil
}

// validateCreateConfiguration : Validate create message class configuration request
func validateCreateConfiguration(ctx context.Context, request *models.CreateConfigurationRequest, configuration *Configuration) error {
	var parent Configuration

	// Returning error and logging if classId already exists in the database
	if err := checkClassIdExists(ctx, request.ClassId); err != nil {
		return err
	}

	// Validate parent request if parentId is not 0
	if request.ParentId != "0" {
		// Initializing Parent Message Class Model

		// Returning error and logging if message class parent does not exist in the database
		if err := getConfigurationParentId(ctx, request.ParentId, &parent); err != nil {
			return err
		}

		if request.AppId == "" {
			request.AppId = parent.AppId
		}

		if request.Name == "" {
			request.Name = parent.Name
		}

		if request.States == nil {
			request.States = parent.States
		} else {
			// Returning error and logging if failed to validate message class states
			if err := validateStatesContainingDeleted(request.States); err != nil {
				return err
			}
		}

		if request.Type == "" {
			request.Type = parent.Type
		} else {
			// Returning error and logging if failed to validate message type
			if err := validateMessageType(request.Type); err != nil {
				return err
			}
		}

		if request.Mode == "" {
			request.Mode = parent.Mode
		} else {
			// Returning error and logging if failed to validate message class mode
			if err := validateMessageMode(request.Mode); err != nil {
				return err
			}
		}

		if request.StatusCallbacks.Kafka == "" || request.StatusCallbacks.Http == "" {
			request.StatusCallbacks = parent.StatusCallbacks
		}

		if request.Actions == nil {
			request.Actions = parent.Actions
		} else {
			// Returning error and logging if failed to validate message class type
			if err := validateActions(request.Actions); err != nil {
				return err
			}
		}

		if request.Mode == "email" {
			if request.CustomEmailConfiguration.EmailAccount == "" || request.CustomEmailConfiguration.EmailPassword == "" {
				request.CustomEmailConfiguration = parent.CustomEmailConfiguration
			} else {
				emailPassword, err := AES_EncryptBase64([]byte(request.CustomEmailConfiguration.EmailPassword), []byte(configs.SecretKey))
				if err != nil {
					return err
				} else {
					request.CustomEmailConfiguration.EmailPassword = emailPassword
				}
			}
			request.CustomSmsConfiguration = models.CustomSmsConfiguration{}
			request.CustomFeishuConfiguration = models.CustomFeishuConfiguration{}
			request.CustomDingTalkConfiguration = models.CustomDingTalkConfiguration{}
		} else if request.Mode == "sms" {
			if request.CustomSmsConfiguration.Huawei.Channel == "" || request.CustomSmsConfiguration.Huawei.Signature == "" || request.CustomSmsConfiguration.Huawei.TemplateId == "" {
				request.CustomSmsConfiguration = parent.CustomSmsConfiguration
			} else {

			}
			request.CustomEmailConfiguration = models.CustomEmailConfiguration{}
			request.CustomFeishuConfiguration = models.CustomFeishuConfiguration{}
			request.CustomDingTalkConfiguration = models.CustomDingTalkConfiguration{}

		} else if request.Mode == "third-party" {
			// Feishu Logic
			appId := request.CustomFeishuConfiguration.AppId
			appSecret := request.CustomFeishuConfiguration.AppSecret
			if appId == "" || appSecret == "" {
				request.CustomFeishuConfiguration = parent.CustomFeishuConfiguration
			} else {
				feishuAppId, err := AES_EncryptBase64([]byte(request.CustomFeishuConfiguration.AppId), []byte(configs.SecretKey))
				if err != nil {
					return err
				} else {
					request.CustomFeishuConfiguration.AppId = feishuAppId
				}

				feishuAppSecret, err := AES_EncryptBase64([]byte(request.CustomFeishuConfiguration.AppSecret), []byte(configs.SecretKey))
				if err != nil {
					return err
				} else {
					request.CustomFeishuConfiguration.AppSecret = feishuAppSecret
				}
			}

			var feishuConfig models.FeishuConfigData
			feishuConfig.AppId = appId
			feishuConfig.AppSecret = appSecret
			feishuConfig.AppType = request.CustomFeishuConfiguration.Type
			if feishuConfig.AppType == "" {
				feishuConfig.AppType = "store"
			}

			_, err := database.RedisGetString(redisFeishuApplicationsKey + feishuConfig.AppId)
			if err != nil {
				if err.Error() == "redis: nil" {
					binary, err := MarshalBinary(feishuConfig)
					if err != nil {
						Log.Error(err.Error())
						return CreateError(codes.Internal, 1000)
					} else {
						err := database.RedisSet(redisFeishuApplicationsKey+feishuConfig.AppId, binary, &infiniteDuration)
						if err != nil {
							Log.Error(err.Error())
							return CreateError(codes.Internal, 1001)
						}
					}
				}
			}

			// DingTalk Logic
			suiteKey := request.CustomDingTalkConfiguration.SuiteKey
			suiteSecret := request.CustomDingTalkConfiguration.SuiteSecret
			if request.CustomDingTalkConfiguration.SuiteId == "" ||
				request.CustomDingTalkConfiguration.SuiteKey == "" ||
				request.CustomDingTalkConfiguration.SuiteSecret == "" ||
				request.CustomDingTalkConfiguration.AppId == "" ||
				request.CustomDingTalkConfiguration.MiniAppId == "" ||
				request.CustomDingTalkConfiguration.TemplateId == "" ||
				request.CustomDingTalkConfiguration.AgentId == "" {
				request.CustomDingTalkConfiguration = parent.CustomDingTalkConfiguration
			} else {
				dingTalkSuiteKey, err := AES_EncryptBase64([]byte(request.CustomDingTalkConfiguration.SuiteKey), []byte(configs.SecretKey))
				if err != nil {
					return err
				} else {
					request.CustomDingTalkConfiguration.SuiteKey = dingTalkSuiteKey
				}

				dingTalkSuiteSecret, err := AES_EncryptBase64([]byte(request.CustomDingTalkConfiguration.SuiteSecret), []byte(configs.SecretKey))
				if err != nil {
					return err
				} else {
					request.CustomDingTalkConfiguration.SuiteSecret = dingTalkSuiteSecret
				}
			}
			var dingtalkConfigData models.DingTalkConfigData
			dingtalkConfigData.AppId = request.CustomDingTalkConfiguration.SuiteId
			dingtalkConfigData.SuiteKey = suiteKey
			dingtalkConfigData.SuiteSecret = suiteSecret
			dingtalkConfigData.SuiteId = request.CustomDingTalkConfiguration.SuiteId

			_, err = database.RedisGetString(redisDingTalkApplicationsKey + dingtalkConfigData.AppId)
			if err != nil {
				if err.Error() == "redis: nil" {
					binary, err := MarshalBinary(dingtalkConfigData)
					if err != nil {
						Log.Error(err.Error())
						return CreateError(codes.Internal, 1000)
					} else {
						err := database.RedisSet(redisDingTalkApplicationsKey+dingtalkConfigData.AppId, binary, &infiniteDuration)
						if err != nil {
							Log.Error(err.Error())
							return CreateError(codes.Internal, 1001)
						}
					}
				}
			}
			request.CustomSmsConfiguration = models.CustomSmsConfiguration{}
			request.CustomEmailConfiguration = models.CustomEmailConfiguration{}
		} else if request.Mode == "feishu" {
			appId := request.CustomFeishuConfiguration.AppId
			appSecret := request.CustomFeishuConfiguration.AppSecret
			if appId == "" || appSecret == "" {
				request.CustomFeishuConfiguration = parent.CustomFeishuConfiguration
			} else {
				feishuAppId, err := AES_EncryptBase64([]byte(request.CustomFeishuConfiguration.AppId), []byte(configs.SecretKey))
				if err != nil {
					return err
				} else {
					request.CustomFeishuConfiguration.AppId = feishuAppId
				}

				feishuAppSecret, err := AES_EncryptBase64([]byte(request.CustomFeishuConfiguration.AppSecret), []byte(configs.SecretKey))
				if err != nil {
					return err
				} else {
					request.CustomFeishuConfiguration.AppSecret = feishuAppSecret
				}
			}

			var feishuConfig models.FeishuConfigData
			feishuConfig.AppId = appId
			feishuConfig.AppSecret = appSecret
			feishuConfig.AppType = request.CustomFeishuConfiguration.Type
			if feishuConfig.AppType == "" {
				feishuConfig.AppType = "store"
			}

			_, err := database.RedisGetString(redisFeishuApplicationsKey + feishuConfig.AppId)
			if err != nil {
				if err.Error() == "redis: nil" {
					binary, err := MarshalBinary(feishuConfig)
					if err != nil {
						Log.Error(err.Error())
						return CreateError(codes.Internal, 1000)
					} else {
						err := database.RedisSet(redisFeishuApplicationsKey+feishuConfig.AppId, binary, &infiniteDuration)
						if err != nil {
							Log.Error(err.Error())
							return CreateError(codes.Internal, 1001)
						}
					}
				}
			}
			request.CustomSmsConfiguration = models.CustomSmsConfiguration{}
			request.CustomEmailConfiguration = models.CustomEmailConfiguration{}
			request.CustomDingTalkConfiguration = models.CustomDingTalkConfiguration{}

		} else if request.Mode == "dingtalk" {
			suiteKey := request.CustomDingTalkConfiguration.SuiteKey
			suiteSecret := request.CustomDingTalkConfiguration.SuiteSecret
			if request.CustomDingTalkConfiguration.SuiteId == "" ||
				request.CustomDingTalkConfiguration.SuiteKey == "" ||
				request.CustomDingTalkConfiguration.SuiteSecret == "" ||
				request.CustomDingTalkConfiguration.AppId == "" ||
				request.CustomDingTalkConfiguration.MiniAppId == "" ||
				request.CustomDingTalkConfiguration.TemplateId == "" ||
				request.CustomDingTalkConfiguration.AgentId == "" {
				request.CustomDingTalkConfiguration = parent.CustomDingTalkConfiguration
			} else {
				dingTalkSuiteKey, err := AES_EncryptBase64([]byte(request.CustomDingTalkConfiguration.SuiteKey), []byte(configs.SecretKey))
				if err != nil {
					return err
				} else {
					request.CustomDingTalkConfiguration.SuiteKey = dingTalkSuiteKey
				}

				dingTalkSuiteSecret, err := AES_EncryptBase64([]byte(request.CustomDingTalkConfiguration.SuiteSecret), []byte(configs.SecretKey))
				if err != nil {
					return err
				} else {
					request.CustomDingTalkConfiguration.SuiteSecret = dingTalkSuiteSecret
				}
			}
			var dingtalkConfigData models.DingTalkConfigData
			dingtalkConfigData.AppId = request.CustomDingTalkConfiguration.SuiteId
			dingtalkConfigData.SuiteKey = suiteKey
			dingtalkConfigData.SuiteSecret = suiteSecret
			dingtalkConfigData.SuiteId = request.CustomDingTalkConfiguration.SuiteId

			_, err := database.RedisGetString(redisDingTalkApplicationsKey + dingtalkConfigData.AppId)
			if err != nil {
				if err.Error() == "redis: nil" {
					binary, err := MarshalBinary(dingtalkConfigData)
					if err != nil {
						Log.Error(err.Error())
						return CreateError(codes.Internal, 1000)
					} else {
						err := database.RedisSet(redisDingTalkApplicationsKey+dingtalkConfigData.AppId, binary, &infiniteDuration)
						if err != nil {
							Log.Error(err.Error())
							return CreateError(codes.Internal, 1001)
						}
					}
				}
			}

			request.CustomSmsConfiguration = models.CustomSmsConfiguration{}
			request.CustomEmailConfiguration = models.CustomEmailConfiguration{}
			request.CustomFeishuConfiguration = models.CustomFeishuConfiguration{}
		} else {
			request.CustomSmsConfiguration = models.CustomSmsConfiguration{}
			request.CustomEmailConfiguration = models.CustomEmailConfiguration{}
			request.CustomFeishuConfiguration = models.CustomFeishuConfiguration{}
			request.CustomDingTalkConfiguration = models.CustomDingTalkConfiguration{}
		}

	} else {
		// Returning error and logging if failed to validate message class states
		if err := validateStatesContainingDeleted(request.States); err != nil {
			return err
		}

		// Returning error and logging if failed to validate message type
		if err := validateMessageType(request.Type); err != nil {
			return err
		}

		// Returning error and logging if failed to validate message class mode
		if err := validateMessageMode(request.Mode); err != nil {
			return err
		}

		// Returning error and logging if failed to validate message class type
		if err := validateActions(request.Actions); err != nil {
			return err
		}

		if request.Mode == "email" {
			if request.CustomEmailConfiguration.EmailAccount == "" || request.CustomEmailConfiguration.EmailPassword == "" {
				request.CustomEmailConfiguration.EmailAccount = configs.EmailAccount
				request.CustomEmailConfiguration.EmailPassword = configs.EmailPassword
			}

			emailPassword, err := AES_EncryptBase64([]byte(request.CustomEmailConfiguration.EmailPassword), []byte(configs.SecretKey))
			if err != nil {
				return err
			} else {
				request.CustomEmailConfiguration.EmailPassword = emailPassword
			}
			request.CustomSmsConfiguration = models.CustomSmsConfiguration{}
			request.CustomFeishuConfiguration = models.CustomFeishuConfiguration{}
			request.CustomDingTalkConfiguration = models.CustomDingTalkConfiguration{}
		} else if request.Mode == "sms" {
			if request.CustomSmsConfiguration.Huawei.Channel == "" || request.CustomSmsConfiguration.Huawei.Signature == "" || request.CustomSmsConfiguration.Huawei.TemplateId == "" {
				request.CustomSmsConfiguration.Huawei.Channel = configs.HuaweiChannel
				request.CustomSmsConfiguration.Huawei.Signature = configs.HuaweiSignature
				request.CustomSmsConfiguration.Huawei.TemplateId = configs.HuaweiTemplateId
			}
			request.CustomEmailConfiguration = models.CustomEmailConfiguration{}
			request.CustomFeishuConfiguration = models.CustomFeishuConfiguration{}
			request.CustomDingTalkConfiguration = models.CustomDingTalkConfiguration{}

		} else if request.Mode == "third-party" {
			// Feishu Logic
			if request.CustomFeishuConfiguration.AppId == "" || request.CustomFeishuConfiguration.AppSecret == "" {
				request.CustomFeishuConfiguration.AppId = configs.FeishuAppId
				request.CustomFeishuConfiguration.AppSecret = configs.FeishuAppSecret
				request.CustomFeishuConfiguration.Type = configs.FeishuAppType
			}
			appId := request.CustomFeishuConfiguration.AppId
			appSecret := request.CustomFeishuConfiguration.AppSecret
			feishuAppId, err := AES_EncryptBase64([]byte(request.CustomFeishuConfiguration.AppId), []byte(configs.SecretKey))
			if err != nil {
				return err
			} else {
				request.CustomFeishuConfiguration.AppId = feishuAppId
			}

			feishuAppSecret, err := AES_EncryptBase64([]byte(request.CustomFeishuConfiguration.AppSecret), []byte(configs.SecretKey))
			if err != nil {
				return err
			} else {
				request.CustomFeishuConfiguration.AppSecret = feishuAppSecret
			}
			var feishuConfig models.FeishuConfigData
			feishuConfig.AppId = appId
			feishuConfig.AppSecret = appSecret
			feishuConfig.AppType = request.CustomFeishuConfiguration.Type

			if feishuConfig.AppType == "" {
				feishuConfig.AppType = "store"
			}

			_, err = database.RedisGetString(redisFeishuApplicationsKey + feishuConfig.AppId)
			if err != nil {
				if err.Error() == "redis: nil" {
					binary, err := MarshalBinary(feishuConfig)
					if err != nil {
						Log.Error(err.Error())
						return CreateError(codes.Internal, 1000)
					} else {
						err := database.RedisSet(redisFeishuApplicationsKey+feishuConfig.AppId, binary, &infiniteDuration)
						if err != nil {
							Log.Error(err.Error())
							return CreateError(codes.Internal, 1001)
						}
					}
				}
			}

			// DingTalk Logic
			if request.CustomDingTalkConfiguration.SuiteId == "" ||
				request.CustomDingTalkConfiguration.SuiteKey == "" ||
				request.CustomDingTalkConfiguration.SuiteSecret == "" ||
				request.CustomDingTalkConfiguration.AppId == "" ||
				request.CustomDingTalkConfiguration.MiniAppId == "" ||
				request.CustomDingTalkConfiguration.TemplateId == "" ||
				request.CustomDingTalkConfiguration.AgentId == "" {
				request.CustomDingTalkConfiguration.SuiteId = configs.DingTalkSuiteId
				request.CustomDingTalkConfiguration.SuiteKey = configs.DingTalkSuiteKey
				request.CustomDingTalkConfiguration.SuiteSecret = configs.DingTalkSuiteSecret
				request.CustomDingTalkConfiguration.AppId = configs.DingTalkAppId
				request.CustomDingTalkConfiguration.MiniAppId = configs.DingTalkMiniAppId
				request.CustomDingTalkConfiguration.TemplateId = configs.DingTalkTemplateId
				request.CustomDingTalkConfiguration.AgentId = configs.DingTalkAgentId
			}
			suiteKey := request.CustomDingTalkConfiguration.SuiteKey
			suiteSecret := request.CustomDingTalkConfiguration.SuiteSecret

			dingTalkSuiteKey, err := AES_EncryptBase64([]byte(request.CustomDingTalkConfiguration.SuiteKey), []byte(configs.SecretKey))
			if err != nil {
				return err
			} else {
				request.CustomDingTalkConfiguration.SuiteKey = dingTalkSuiteKey
			}

			dingTalkSuiteSecret, err := AES_EncryptBase64([]byte(request.CustomDingTalkConfiguration.SuiteSecret), []byte(configs.SecretKey))
			if err != nil {
				return err
			} else {
				request.CustomDingTalkConfiguration.SuiteSecret = dingTalkSuiteSecret
			}

			var dingtalkConfigData models.DingTalkConfigData
			dingtalkConfigData.AppId = request.CustomDingTalkConfiguration.AppId
			dingtalkConfigData.SuiteKey = suiteKey
			dingtalkConfigData.SuiteSecret = suiteSecret
			dingtalkConfigData.SuiteId = request.CustomDingTalkConfiguration.SuiteId

			_, err = database.RedisGetString(redisDingTalkApplicationsKey + dingtalkConfigData.AppId)
			if err != nil {
				if err.Error() == "redis: nil" {
					binary, err := MarshalBinary(dingtalkConfigData)
					if err != nil {
						Log.Error(err.Error())
						return CreateError(codes.Internal, 1000)
					} else {
						err := database.RedisSet(redisDingTalkApplicationsKey+dingtalkConfigData.AppId, binary, &infiniteDuration)
						if err != nil {
							Log.Error(err.Error())
							return CreateError(codes.Internal, 1001)
						}
					}
				}
			}

			request.CustomSmsConfiguration = models.CustomSmsConfiguration{}
			request.CustomEmailConfiguration = models.CustomEmailConfiguration{}
		} else if request.Mode == "feishu" {
			if request.CustomFeishuConfiguration.AppId == "" || request.CustomFeishuConfiguration.AppSecret == "" {
				request.CustomFeishuConfiguration.AppId = configs.FeishuAppId
				request.CustomFeishuConfiguration.AppSecret = configs.FeishuAppSecret
				request.CustomFeishuConfiguration.Type = configs.FeishuAppType
			}
			appId := request.CustomFeishuConfiguration.AppId
			appSecret := request.CustomFeishuConfiguration.AppSecret
			feishuAppId, err := AES_EncryptBase64([]byte(request.CustomFeishuConfiguration.AppId), []byte(configs.SecretKey))
			if err != nil {
				return err
			} else {
				request.CustomFeishuConfiguration.AppId = feishuAppId
			}

			feishuAppSecret, err := AES_EncryptBase64([]byte(request.CustomFeishuConfiguration.AppSecret), []byte(configs.SecretKey))
			if err != nil {
				return err
			} else {
				request.CustomFeishuConfiguration.AppSecret = feishuAppSecret
			}
			var feishuConfig models.FeishuConfigData
			feishuConfig.AppId = appId
			feishuConfig.AppSecret = appSecret
			feishuConfig.AppType = request.CustomFeishuConfiguration.Type

			if feishuConfig.AppType == "" {
				feishuConfig.AppType = "store"
			}

			_, err = database.RedisGetString(redisFeishuApplicationsKey + feishuConfig.AppId)
			if err != nil {
				if err.Error() == "redis: nil" {
					binary, err := MarshalBinary(feishuConfig)
					if err != nil {
						Log.Error(err.Error())
						return CreateError(codes.Internal, 1000)
					} else {
						err := database.RedisSet(redisFeishuApplicationsKey+feishuConfig.AppId, binary, &infiniteDuration)
						if err != nil {
							Log.Error(err.Error())
							return CreateError(codes.Internal, 1001)
						}
					}
				}
			}

			request.CustomSmsConfiguration = models.CustomSmsConfiguration{}
			request.CustomEmailConfiguration = models.CustomEmailConfiguration{}
			request.CustomDingTalkConfiguration = models.CustomDingTalkConfiguration{}

		} else if request.Mode == "dingtalk" {
			if request.CustomDingTalkConfiguration.SuiteId == "" ||
				request.CustomDingTalkConfiguration.SuiteKey == "" ||
				request.CustomDingTalkConfiguration.SuiteSecret == "" ||
				request.CustomDingTalkConfiguration.AppId == "" ||
				request.CustomDingTalkConfiguration.MiniAppId == "" ||
				request.CustomDingTalkConfiguration.TemplateId == "" ||
				request.CustomDingTalkConfiguration.AgentId == "" {
				request.CustomDingTalkConfiguration.SuiteId = configs.DingTalkSuiteId
				request.CustomDingTalkConfiguration.SuiteKey = configs.DingTalkSuiteKey
				request.CustomDingTalkConfiguration.SuiteSecret = configs.DingTalkSuiteSecret
				request.CustomDingTalkConfiguration.AppId = configs.DingTalkAppId
				request.CustomDingTalkConfiguration.MiniAppId = configs.DingTalkMiniAppId
				request.CustomDingTalkConfiguration.TemplateId = configs.DingTalkTemplateId
				request.CustomDingTalkConfiguration.AgentId = configs.DingTalkAgentId
			}
			suiteKey := request.CustomDingTalkConfiguration.SuiteKey
			suiteSecret := request.CustomDingTalkConfiguration.SuiteSecret

			dingTalkSuiteKey, err := AES_EncryptBase64([]byte(request.CustomDingTalkConfiguration.SuiteKey), []byte(configs.SecretKey))
			if err != nil {
				return err
			} else {
				request.CustomDingTalkConfiguration.SuiteKey = dingTalkSuiteKey
			}

			dingTalkSuiteSecret, err := AES_EncryptBase64([]byte(request.CustomDingTalkConfiguration.SuiteSecret), []byte(configs.SecretKey))
			if err != nil {
				return err
			} else {
				request.CustomDingTalkConfiguration.SuiteSecret = dingTalkSuiteSecret
			}

			var dingtalkConfigData models.DingTalkConfigData
			dingtalkConfigData.AppId = request.CustomDingTalkConfiguration.AppId
			dingtalkConfigData.SuiteKey = suiteKey
			dingtalkConfigData.SuiteSecret = suiteSecret
			dingtalkConfigData.SuiteId = request.CustomDingTalkConfiguration.SuiteId

			_, err = database.RedisGetString(redisDingTalkApplicationsKey + dingtalkConfigData.AppId)
			if err != nil {
				if err.Error() == "redis: nil" {
					binary, err := MarshalBinary(dingtalkConfigData)
					if err != nil {
						Log.Error(err.Error())
						return CreateError(codes.Internal, 1000)
					} else {
						err := database.RedisSet(redisDingTalkApplicationsKey+dingtalkConfigData.AppId, binary, &infiniteDuration)
						if err != nil {
							Log.Error(err.Error())
							return CreateError(codes.Internal, 1001)
						}
					}
				}
			}

			request.CustomSmsConfiguration = models.CustomSmsConfiguration{}
			request.CustomEmailConfiguration = models.CustomEmailConfiguration{}
			request.CustomFeishuConfiguration = models.CustomFeishuConfiguration{}
		} else {
			request.CustomSmsConfiguration = models.CustomSmsConfiguration{}
			request.CustomEmailConfiguration = models.CustomEmailConfiguration{}
			request.CustomFeishuConfiguration = models.CustomFeishuConfiguration{}
			request.CustomDingTalkConfiguration = models.CustomDingTalkConfiguration{}
		}
	}

	configuration.NewConfiguration(request)
	// Returning nil if validation is successful
	return nil
}

// validateUpdateConfiguration : Validate update message class configuration request
func validateUpdateConfiguration(ctx context.Context, request *models.UpdateConfigurationRequest, configuration *Configuration) error {

	if request.AppId != "" {
		configuration.AppId = request.AppId
	}

	if request.Name != "" {
		configuration.Name = request.Name
	}

	if request.States != nil {
		// Returning error and logging if failed to validate message class states
		if err := validateStatesContainingDeleted(request.States); err != nil {
			return err
		} else {
			configuration.States = request.States
		}
	}

	if request.Type != "" {
		// Returning error and logging if failed to validate message type
		if err := validateMessageType(request.Type); err != nil {
			return err
		} else {
			configuration.Type = request.Type
		}
	}

	if request.Mode != "" {
		// Returning error and logging if failed to validate message class mode
		if err := validateMessageMode(request.Mode); err != nil {
			return err
		} else {
			configuration.Mode = request.Mode
		}
	}

	if request.Actions != nil {
		// Returning error and logging if failed to validate message class type
		if err := validateActions(request.Actions); err != nil {
			return err
		} else {
			configuration.Actions = request.Actions
		}
	}
	if request.Mode == "email" {
		if request.CustomEmailConfiguration.EmailAccount != "" &&
			request.CustomEmailConfiguration.EmailPassword != "" {
			emailPassword, err := AES_EncryptBase64([]byte(request.CustomEmailConfiguration.EmailPassword), []byte(configs.SecretKey))
			if err != nil {
				return err
			} else {
				configuration.CustomEmailConfiguration.EmailAccount = request.CustomEmailConfiguration.EmailAccount
				configuration.CustomEmailConfiguration.EmailPassword = emailPassword
			}
			configuration.CustomSmsConfiguration = models.CustomSmsConfiguration{}
			configuration.CustomFeishuConfiguration = models.CustomFeishuConfiguration{}
			configuration.CustomDingTalkConfiguration = models.CustomDingTalkConfiguration{}
		}
	} else if request.Mode == "sms" {
		if request.CustomSmsConfiguration.Huawei.Channel != "" &&
			request.CustomSmsConfiguration.Huawei.Signature != "" &&
			request.CustomSmsConfiguration.Huawei.TemplateId != "" {
			configuration.CustomSmsConfiguration.Huawei.Channel = request.CustomSmsConfiguration.Huawei.Channel
			configuration.CustomSmsConfiguration.Huawei.Signature = request.CustomSmsConfiguration.Huawei.Signature
			configuration.CustomSmsConfiguration.Huawei.TemplateId = request.CustomSmsConfiguration.Huawei.TemplateId
		}
		request.CustomEmailConfiguration = models.CustomEmailConfiguration{}
		request.CustomFeishuConfiguration = models.CustomFeishuConfiguration{}
		request.CustomDingTalkConfiguration = models.CustomDingTalkConfiguration{}

	} else if request.Mode == "feishu" {
		if request.CustomFeishuConfiguration.AppId != "" &&
			request.CustomFeishuConfiguration.AppSecret != "" {
			feishuAppId, err := AES_EncryptBase64([]byte(request.CustomFeishuConfiguration.AppId), []byte(configs.SecretKey))
			if err != nil {
				return err
			} else {
				configuration.CustomFeishuConfiguration.AppId = feishuAppId
			}

			feishuAppSecret, err := AES_EncryptBase64([]byte(request.CustomFeishuConfiguration.AppSecret), []byte(configs.SecretKey))
			if err != nil {
				return err
			} else {
				configuration.CustomFeishuConfiguration.AppSecret = feishuAppSecret
			}
		}

		request.CustomSmsConfiguration = models.CustomSmsConfiguration{}
		request.CustomEmailConfiguration = models.CustomEmailConfiguration{}
		request.CustomDingTalkConfiguration = models.CustomDingTalkConfiguration{}

	} else if request.Mode == "dingtalk" {
		if request.CustomDingTalkConfiguration.SuiteId == "" &&
			request.CustomDingTalkConfiguration.SuiteKey == "" &&
			request.CustomDingTalkConfiguration.SuiteSecret == "" &&
			request.CustomDingTalkConfiguration.AppId == "" &&
			request.CustomDingTalkConfiguration.MiniAppId == "" &&
			request.CustomDingTalkConfiguration.TemplateId == "" &&
			request.CustomDingTalkConfiguration.AgentId == "" {
			dingTalkSuiteKey, err := AES_EncryptBase64([]byte(request.CustomDingTalkConfiguration.SuiteKey), []byte(configs.SecretKey))
			if err != nil {
				return err
			} else {
				configuration.CustomDingTalkConfiguration.SuiteKey = dingTalkSuiteKey
			}

			dingTalkSuiteSecret, err := AES_EncryptBase64([]byte(request.CustomDingTalkConfiguration.SuiteSecret), []byte(configs.SecretKey))
			if err != nil {
				return err
			} else {
				configuration.CustomDingTalkConfiguration.SuiteSecret = dingTalkSuiteSecret
			}
			configuration.CustomDingTalkConfiguration.SuiteId = request.CustomDingTalkConfiguration.SuiteId
			configuration.CustomDingTalkConfiguration.AppId = request.CustomDingTalkConfiguration.AppId
			configuration.CustomDingTalkConfiguration.MiniAppId = request.CustomDingTalkConfiguration.MiniAppId
			configuration.CustomDingTalkConfiguration.TemplateId = request.CustomDingTalkConfiguration.TemplateId
			configuration.CustomDingTalkConfiguration.AgentId = request.CustomDingTalkConfiguration.AgentId
		}
		request.CustomSmsConfiguration = models.CustomSmsConfiguration{}
		request.CustomEmailConfiguration = models.CustomEmailConfiguration{}
		request.CustomFeishuConfiguration = models.CustomFeishuConfiguration{}
	} else {
		request.CustomSmsConfiguration = models.CustomSmsConfiguration{}
		request.CustomEmailConfiguration = models.CustomEmailConfiguration{}
		request.CustomFeishuConfiguration = models.CustomFeishuConfiguration{}
		request.CustomDingTalkConfiguration = models.CustomDingTalkConfiguration{}
	}

	// Setting default values for configuration
	configuration.UpdatedAt = time.Now()

	// Returning nil if validation is successful
	return nil
}

// validateGetConfiguration : Validate get message class configuration request
func validateGetConfiguration(ctx context.Context, id string, configuration *Configuration) error {
	// Returning error and logging if failed to get configuration
	if err := getConfigurationByClassId(ctx, id, configuration); err != nil {
		return err
	}

	// Returning nil if validation is successful
	return nil
}

// createConfiguration : Create a message class configuration in the database
func createConfiguration(
	ctx context.Context,
	configuration *Configuration) error {
	if configuration.Mode == "dingtalk" {
		/*if err := registerDingtalkApp(ctx, configuration); err != nil {
			return err
		}*/
	} else if configuration.Mode == "listening" {
		if err := registerListeningConfiguration(ctx, configuration); err != nil {
			return err
		}
	}

	// Returning error and logging if failed to insert message class configuration
	if err := insertOneConfiguration(ctx, configuration); err != nil {
		return err
	}

	// Returning false if successfully inserted message class configuration
	return nil
}

// updateConfiguration : Update a message class configuration in the database
func updateConfiguration(
	ctx context.Context,
	configuration *Configuration) error {
	// Returning error and logging if failed to update message class configuration
	if err := updateOneConfiguration(ctx, configuration); err != nil {
		return err
	}

	// Returning false if successfully updated message class configuration
	return nil
}

// deleteConfiguration : Delete a message class configuration in the database
func deleteConfiguration(
	ctx context.Context,
	configuration *Configuration) error {
	// Returning error and logging if failed to find and delete message class configuration
	if err := deleteOneConfiguration(ctx, configuration); err != nil {
		return err
	}

	if configuration.Mode == "listening" {
		if err := unregisterListeningConfiguration(ctx, configuration); err != nil {
			return err
		}
	}

	// Returning false if successfully deleted message class configuration
	return nil
}

// queryConfigurations : Query message class configurations in the database
func queryConfigurations(
	ctx context.Context,
	params *models.QueryConfigurationsParams) (models.QueryConfigurationResponse, error) {
	// Initializing query parameters
	query := bson.M{}

	// Initializing message class configurations array
	var configurations []Configuration

	// Setting name query parameter if name is not empty
	if params.Name != "" {
		query["name"] = params.Name
	}

	// Setting type query parameter if type is not empty and is valid
	if params.Type != "" {
		// Returning error and logging if type is not valid
		if err := validateMessageType(params.Type); err != nil {
			return models.QueryConfigurationResponse{}, err
		} else {
			// Setting type to params
			query["type"] = params.Type
		}
	}

	// Setting mode query parameter if mode is not empty and is valid
	if params.Mode != "" {
		// Returning error and logging if mode is not valid
		if err := validateMessageMode(params.Mode); err != nil {
			return models.QueryConfigurationResponse{}, err
		} else {
			// Setting mode to params
			query["mode"] = params.Mode
		}
	}

	// Setting appId query parameter if appId is not empty
	if params.AppId != "" {
		query["appId"] = params.AppId
	}

	// Setting parentId query parameter if parentId is not empty
	if params.ParentId != "" {
		query["parentId"] = params.ParentId
	}

	if strings.ToLower(params.Sort) != "" && (strings.ToLower(params.Sort) != "asc" && strings.ToLower(params.Sort) != "desc") {
		Log.Error("Sort must be one of the following: asc, desc. Default is asc")
		return models.QueryConfigurationResponse{}, CreateError(http.StatusBadRequest, 100008)
	} else {
		if strings.ToLower(params.Sort) == "" {
			params.Sort = "asc"
		}
	}

	if params.Order == "" {
		params.Order = "updatedAt"
	}

	// Count total messages filtered by query
	total, err := configurationCollection.CountDocuments(ctx, query)
	if err != nil {
		// Returning error if failed to count messages
		Log.Error(err.Error())
		return models.QueryConfigurationResponse{}, CreateError(codes.Internal, 1001)
	}

	// Setting Options to params message templates by pages
	skip := (params.Page - 1) * params.Limit
	limit := params.Limit

	opts := options.Find().SetSkip(skip).SetLimit(limit)

	if params.Sort == "asc" {
		opts.SetSort(bson.D{{params.Order, 1}})
	} else {
		opts.SetSort(bson.D{{params.Order, -1}})
	}

	//opts.SetProjection(bson.M{})

	// Finding message class configuration by params
	cursor, err := configurationCollection.Find(ctx, query, opts)
	if err != nil {
		// Returning error if failed to find message class configurations
		Log.Error(err.Error())
		return models.QueryConfigurationResponse{}, CreateError(codes.Internal, 1001)
	}

	// Iterating through message class configurations
	for cursor.Next(ctx) {
		var configuration Configuration
		// Decoding message class configuration
		err := cursor.Decode(&configuration)
		if err != nil {
			// Returning error if failed to decode message class configuration
			Log.Error(err.Error())
			return models.QueryConfigurationResponse{}, CreateError(codes.Internal, 1000)
		}
		// Appending message class configuration to message class configurations array
		configurations = append(configurations, configuration)
	}

	// Returning error if no message class configurations found
	if len(configurations) == 0 {
		Log.Error("No message class configurations found")
		return models.QueryConfigurationResponse{}, CreateError(codes.NotFound, 100111)
	}

	queryConfigurationResponse := models.QueryConfigurationResponse{
		Page:    params.Page,
		Limit:   params.Limit,
		Total:   total,
		Results: configurations,
	}
	// Returning false and message class configurations if successfully queried message class configurations
	return queryConfigurationResponse, nil
}

// getConfiguration : Get a message class configuration from the database
func getConfiguration(
	_ context.Context,
	_ string,
	configuration *Configuration) (*pb.Configuration, error) {
	// Converting struct configuration to protobuf configuration
	responseActions := make([]*pb.Action, len(configuration.Actions))
	for i, action := range configuration.Actions {
		responseActions[i] = &pb.Action{
			Name: action.Name,
			Tips: action.Tips,
			Action: &pb.ActionTrigger{
				Type:   action.Action.Type,
				Url:    action.Action.URL,
				Method: action.Action.Method,
				Body:   action.Action.Body,
				Query:  action.Action.Query,
			},
			NextAction: action.NextAction,
		}
	}

	protoConfiguration := &pb.Configuration{
		Id:       configuration.Id.Hex(),
		AppId:    configuration.AppId,
		ClassId:  configuration.ClassId,
		ParentId: configuration.ParentId,
		Name:     configuration.Name,
		Persist:  configuration.Persist,
		States:   configuration.States,
		Type:     configuration.Type,
		Mode:     configuration.Mode,
		StatusCallback: &pb.Policy{
			Kafka: configuration.StatusCallbacks.Kafka,
			Http:  configuration.StatusCallbacks.Http,
		},
		Actions:   responseActions,
		CreatedAt: configuration.CreatedAt.Format(time.RFC3339),
		UpdatedAt: configuration.UpdatedAt.Format(time.RFC3339),
		CustomEmailConfiguration: &pb.CustomEmailConfiguration{
			EmailAccount:  configuration.CustomEmailConfiguration.EmailAccount,
			EmailPassword: configuration.CustomEmailConfiguration.EmailPassword,
		},
		CustomSmsConfiguration: &pb.CustomSmsConfiguration{
			Huawei: &pb.Huawei{
				Sender:     configuration.CustomSmsConfiguration.Huawei.Channel,
				Signature:  configuration.CustomSmsConfiguration.Huawei.Signature,
				TemplateId: configuration.CustomSmsConfiguration.Huawei.TemplateId,
			},
		},
		CustomFeishuConfiguration: &pb.CustomFeishuConfiguration{
			AppId:     configuration.CustomFeishuConfiguration.AppId,
			AppSecret: configuration.CustomFeishuConfiguration.AppSecret,
		},
		CustomDingTalkConfiguration: &pb.CustomDingTalkConfiguration{
			SuiteId:     configuration.CustomDingTalkConfiguration.SuiteId,
			AppId:       configuration.CustomDingTalkConfiguration.AppId,
			MiniAppId:   configuration.CustomDingTalkConfiguration.MiniAppId,
			SuiteKey:    configuration.CustomDingTalkConfiguration.SuiteKey,
			SuiteSecret: configuration.CustomDingTalkConfiguration.SuiteSecret,
			TemplateId:  configuration.CustomDingTalkConfiguration.TemplateId,
			AgentId:     configuration.CustomDingTalkConfiguration.AgentId,
		},
	}

	// Returning nil and protobuf configuration if successfully converted struct configuration to protobuf configuration
	return protoConfiguration, nil
}

// validateActions : Validate actions is a function which is used by the API to validate actions
func validateActions(actions []models.ActionModel) error {
	// Looping through actions
	for _, action := range actions {
		// Returning error and logging if failed to validate action
		if err := validateAction(action, actions); err != nil {
			return err
		}
	}

	// Returning false if all actions are valid
	return nil
}

// validateAction : Validate action is a function which is used by the API to validate an action
func validateAction(action models.ActionModel, actions []models.ActionModel) error {
	// Returning error and logging if failed to validate next action
	if action.NextAction != "" {
		if err := validateNextAction(action, actions); err != nil {
			return err
		}
	}

	// Returning error and logging if failed to validate action trigger
	if action.Action.URL != "" {
		if err := validateActionTrigger(action.Action); err != nil {
			return err
		}
	}

	// Returning false if successfully validated action
	return nil
}

// validateActionTrigger : Validate action trigger is a function which is used by the API to validate an action trigger
func validateActionTrigger(action models.ActionTrigger) error {
	// Returning error and logging if failed to validate action trigger type
	if err := validateActionTriggerType(action.Type); err != nil {
		return err
	}

	// Returning error and logging if failed to validate action trigger method
	if err := validateActionTriggerMethod(action.Method); err != nil {
		return err
	}

	// Returning false if successfully validated action trigger
	return nil
}

// checkClassIdExists: Find message class configuration by class id is a function which is used by the API to find a message class configuration by class id
func checkClassIdExists(ctx context.Context, classId string) error {
	var configuration Configuration
	// Setting Filter
	filter := bson.M{"classId": classId}

	// Finding Message Class Configuration
	err := configurationCollection.FindOne(ctx, filter).Decode(&configuration)

	// Returning error and logging if failed to find message configuration
	if err == nil {
		Log.Error("ClassId: ", classId, "Error: ", "Message Class Configuration already exists")
		return CreateError(codes.AlreadyExists, 100101)
	}

	// Returning false if failed to find the configuration
	return nil
}

// getConfigurationByClassId: Check message class configuration by class id is a function which is used by the API to check if a message class configuration by class id exists
func getConfigurationByClassId(ctx context.Context, classId string, class *Configuration) error {
	// Setting Filter
	filter := bson.M{"classId": classId}

	// Finding Message Class Configuration
	err := configurationCollection.FindOne(ctx, filter).Decode(&class)

	// Returning error and logging if failed to find message class configuration
	if err != nil {
		Log.Error("ClassId: ", classId, "Error: ", err.Error())
		return CreateError(codes.NotFound, 100112)
	}

	// Returning false if failed to find the message class configuration
	return nil
}

// getConfigurationParentId: Validate message class configuration parent id is a function which is used by the API to validate if a message class configuration parent id exists
func getConfigurationParentId(ctx context.Context, classId string, class *Configuration) error {
	// Setting Filter
	filter := bson.M{"classId": classId}

	// Finding Message Class Configuration
	err := configurationCollection.FindOne(ctx, filter).Decode(&class)

	// Returning error and logging if failed to find message class configuration
	if err != nil {
		Log.Error("ClassId: ", classId, " Error: ", err.Error())
		return CreateError(codes.NotFound, 100102)
	}

	// Returning false if failed to find the message class configuration
	return nil
}

// queryConfigurationByIds: Query message class configuration by ids is a function which is used by the API to query message class configuration by ids
func queryConfigurationByIds(ctx context.Context, ids []string, configurations *map[string]Configuration) error {
	// Setting Filter
	filter := bson.M{"classId": bson.M{"$in": ids}}

	// Finding Message Class Configuration
	cursor, err := configurationCollection.Find(ctx, filter)

	// Returning error and logging if failed to find message class configuration
	if err != nil {
		Log.Error("Ids: ", ids, "Error: ", err.Error())
		return CreateError(codes.NotFound, 100103)
	}

	cfs := make(map[string]Configuration)
	// Iterating through message class configurations
	for cursor.Next(ctx) {
		var configuration Configuration
		// Decoding message class configuration
		err := cursor.Decode(&configuration)
		if err != nil {
			// Returning error if failed to decode message class configuration
			Log.Error(err.Error())
			continue
		}
		cfs[configuration.ClassId] = configuration
	}

	*configurations = cfs

	// Returning false if failed to find the message class configuration
	return nil
}

// insertOneConfiguration : Insert one message class configuration is a function which is used by the API to insert a message class configuration
func insertOneConfiguration(ctx context.Context, class *Configuration) error {
	// Inserting message class configuration into MongoDB
	_, err := configurationCollection.InsertOne(ctx, class)

	// Returning error and logging if failed to insert message class configuration
	if err != nil {
		Log.Error(err.Error())
		return CreateError(codes.Internal, 1001)
	}

	// Returning false if successfully inserted message class configuration
	return nil
}

// updateOneConfiguration : Update one message class configuration is a function which is used by the API to update a message class configuration
func updateOneConfiguration(ctx context.Context, configuration *Configuration) error {
	// Setting filter to find message template to update
	filter := bson.M{"_id": configuration.Id}

	// Setting updatedAt
	configuration.UpdatedAt = time.Now()

	// Setting update
	update := bson.D{
		{"$set", bson.D{
			{"classId", configuration.ClassId},
			{"parentId", configuration.ParentId},
			{"appId", configuration.AppId},
			{"name", configuration.Name},
			{"persist", configuration.Persist},
			{"states", configuration.States},
			{"type", configuration.Type},
			{"mode", configuration.Mode},
			{"statusCallbacks", configuration.StatusCallbacks},
			{"actions", configuration.Actions},
			{"updatedAt", configuration.UpdatedAt},
		},
		}}

	// Updating message configuration into MongoDB
	_, err := configurationCollection.UpdateOne(ctx, filter, update)

	// Returning error and logging if failed to update message configuration
	if err != nil {
		Log.Error(err.Error())
		return CreateError(codes.Internal, 1001)
	}

	// Returning nil if successfully updated message configuration
	return nil
}

// deleteOneConfiguration : Delete one message class configuration is a function which is used by the API to delete a message class configuration
func deleteOneConfiguration(ctx context.Context, configuration *Configuration) error {
	// Setting filter to find message class configuration to update
	filter := bson.M{"_id": configuration.Id}

	// Deleting message class configuration from MongoDB
	_, err := configurationCollection.DeleteOne(ctx, filter)

	// Returning error and logging if failed to delete message class configuration
	if err != nil {
		Log.Error(err.Error())
		return CreateError(codes.Internal, 1001)
	} else {
		return nil
	}
}

// validateStatesContainingDeleted : Validate states containing deleted is a function which is used by the API to validate states containing deleted
func validateStatesContainingDeleted(states []string) error {
	// Returning error and logging if states does not contain deleted
	if !contains(states, "deleted") {
		Log.Error("Error: ", "States must contain deleted")
		return CreateError(codes.InvalidArgument, 100103)
	} else {
		return nil
	}
}

// validateMessageType : Validate message type is a function which is used by the API to validate message type
func validateMessageType(messageType string) error {
	// Returning error and logging if message type is not valid
	if !contains(messageTypes, messageType) {
		Log.Error("Error: ", "Message type is invalid")
		return CreateError(codes.InvalidArgument, 100104)
	} else {
		return nil
	}
}

// validateMessageMode : Validate message mode is a function which is used by the API to validate message mode
func validateMessageMode(messageMode string) error {
	// Returning error and logging if message mode is not valid
	if !contains(messageModes, messageMode) {
		Log.Error("Error: ", "Message mode is invalid")
		return CreateError(codes.InvalidArgument, 100105)
	} else {
		return nil
	}
}

// validateNextAction : Validate next action is a function which is used by the API to validate next action
func validateNextAction(action models.ActionModel, actions []models.ActionModel) error {
	// Returning error and logging if next action is not valid
	if action.NextAction == action.Name {
		Log.Error("Error: ", "Next action cannot be the same as the current action")
		return CreateError(codes.InvalidArgument, 100106)
	} else {
		count := 0
		check := false

		// Validating Action with other actions
		for _, otherAction := range actions {
			if action.Name == otherAction.Name {
				count++
			}
			if action.NextAction == otherAction.Name {
				check = true
			}
		}

		// Returning error and logging if action name is not unique
		if count > 1 {
			Log.Error("Error: ", "Action name is not unique")
			return CreateError(codes.InvalidArgument, 100107)
		}

		// Returning error and logging if action next status is not found
		if !check {
			Log.Error("Error: ", "Action next status is not found")
			return CreateError(codes.InvalidArgument, 100108)
		}
	}

	// Returning nil if next action is valid
	return nil
}

// validateActionTriggerType : Validate action trigger type is a function which is used by the API to validate action trigger type
func validateActionTriggerType(triggerType string) error {
	// Returning error and logging if action trigger type is not valid
	if !contains(actionTypes, triggerType) {
		Log.Error("Error: ", "Action trigger type is invalid")
		return CreateError(codes.InvalidArgument, 100109)
	} else {
		return nil
	}
}

// validateActionTriggerMethod : Validate action trigger method is a function which is used by the API to validate action trigger method
func validateActionTriggerMethod(method string) error {
	// Returning error and logging if action trigger method is not valid
	if !contains(httpMethods, method) {
		Log.Error("Error: ", "Action trigger method is invalid")
		return CreateError(codes.InvalidArgument, 100110)
	} else {
		return nil
	}
}

func registerListeningConfiguration(ctx context.Context, configuration *Configuration) error {
	users := make(map[string][]string)
	bytes, err := MarshalBinary(users)
	if err != nil {
		Log.Error(err.Error())
		return CreateError(codes.Internal, 1001)
	}
	err = database.RedisSetCtx(ctx, redisUserListKey+configuration.ClassId, bytes, &infiniteDuration)
	if err != nil {
		Log.Error(err.Error())
		return CreateError(codes.Internal, 1001)
	} else {
		return nil
	}
}

func unregisterListeningConfiguration(ctx context.Context, configuration *Configuration) error {
	err := database.RedisMultiDelCtx(ctx, []string{redisUserListKey + configuration.ClassId})
	if err != nil {
		Log.Error(err.Error())
		return CreateError(codes.Internal, 1001)
	} else {
		return nil
	}
}

func GenerateToken() string {
	// Generating token
	token := uuid.New().String()

	// Returning token
	return token
}
