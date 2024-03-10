package controllers

// importing packages
import (
	"context"
	"messenger/restful/models"
	"messenger/restful/services"
	pb "messenger/services"
)

// ConfigurationServer : Implementation of ConfigurationServer grpc
type ConfigurationServer struct {
	pb.ConfigurationServiceServer
}

// CreateConfiguration : Call the service to create a new message class configuration
func (s *ConfigurationServer) CreateConfiguration(ctx context.Context, req *pb.CreateConfigurationRequest) (*pb.Configuration, error) {
	// Converting actions from proto to struct
	print("cc1")
	actions := make([]models.ActionModel, len(req.Actions))
	for i, action := range req.Actions {
		trigger := models.ActionTrigger{
			Type:   action.GetAction().GetType(),
			URL:    action.GetAction().GetType(),
			Method: action.GetAction().GetMethod(),
			Query:  action.GetAction().GetQuery(),
			Body:   action.GetAction().GetBody(),
		}
		actions[i] = models.ActionModel{
			Name:       action.GetName(),
			Tips:       action.GetTips(),
			Action:     trigger,
			NextAction: action.GetNextAction(),
		}
	}

	// Setting createConfigurationRequest struct
	createConfigurationRequest := models.CreateConfigurationRequest{
		ClassId:  req.GetClassId(),
		ParentId: req.GetParentId(),
		AppId:    req.GetAppId(),
		Name:     req.GetName(),
		Persist:  req.GetPersist(),
		States:   req.GetStates(),
		Type:     req.GetType(),
		Mode:     req.GetMode(),
		StatusCallbacks: models.Policy{
			Kafka: req.GetStatusCallback().GetKafka(),
			Http:  req.GetStatusCallback().GetHttp(),
		},
		Actions: actions,
		CustomEmailConfiguration: models.CustomEmailConfiguration{
			EmailAccount:  req.GetCustomEmailConfiguration().GetEmailAccount(),
			EmailPassword: req.GetCustomEmailConfiguration().GetEmailPassword(),
		},
		CustomSmsConfiguration: models.CustomSmsConfiguration{
			Huawei: models.Huawei{
				Channel:    req.GetCustomSmsConfiguration().GetHuawei().GetSender(),
				Signature:  req.GetCustomSmsConfiguration().GetHuawei().GetSignature(),
				TemplateId: req.GetCustomSmsConfiguration().GetHuawei().GetTemplateId(),
			},
		},
		CustomFeishuConfiguration: models.CustomFeishuConfiguration{
			AppId:     req.GetCustomFeishuConfiguration().GetAppId(),
			AppSecret: req.GetCustomFeishuConfiguration().GetAppSecret(),
			Type:      "store",
		},
		CustomDingTalkConfiguration: models.CustomDingTalkConfiguration{
			SuiteId:     req.GetCustomDingTalkConfiguration().GetSuiteId(),
			AppId:       req.GetCustomDingTalkConfiguration().GetAppId(),
			MiniAppId:   req.GetCustomDingTalkConfiguration().GetMiniAppId(),
			SuiteKey:    req.GetCustomDingTalkConfiguration().GetSuiteKey(),
			SuiteSecret: req.GetCustomDingTalkConfiguration().GetSuiteSecret(),
			TemplateId:  req.GetCustomDingTalkConfiguration().GetTemplateId(),
			AgentId:     req.GetCustomDingTalkConfiguration().GetAgentId(),
		},
	}

	// Returning error and logging if failed to create createConfigurationRequest
	if configuration, err := services.CreateConfiguration(ctx, &createConfigurationRequest); err != nil {
		return nil, err
	} else {
		// Returning success and logging if createConfigurationRequest created successfully
		log.Info(configuration)

		// Converting actions from struct to proto
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

		// Returning createConfigurationRequest in proto format
		return &pb.Configuration{
			Id:       configuration.Id.Hex(),
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
			Actions: responseActions,
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
			CreatedAt: configuration.CreatedAt.String(),
			UpdatedAt: configuration.UpdatedAt.String(),
		}, nil
	}
}

// UpdateConfiguration : Call the service to update an existing message class configuration
func (s *ConfigurationServer) UpdateConfiguration(ctx context.Context, req *pb.UpdateConfigurationRequest) (*pb.Configuration, error) {
	// Converting actions from proto to struct
	actions := make([]models.ActionModel, len(req.Actions))
	for i, action := range req.Actions {
		trigger := models.ActionTrigger{
			Type:   action.GetAction().GetType(),
			URL:    action.GetAction().GetType(),
			Method: action.GetAction().GetMethod(),
			Query:  action.GetAction().GetQuery(),
			Body:   action.GetAction().GetBody(),
		}
		actions[i] = models.ActionModel{
			Name:       action.GetName(),
			Tips:       action.GetTips(),
			Action:     trigger,
			NextAction: action.GetNextAction(),
		}
	}

	// Setting request struct
	request := models.UpdateConfigurationRequest{
		ClassId: req.GetClassId(),
		AppId:   req.GetAppId(),
		Name:    req.GetName(),
		Persist: req.GetPersist(),
		States:  req.GetStates(),
		Type:    req.GetType(),
		Mode:    req.GetMode(),
		StatusCallbacks: models.Policy{
			Kafka: req.GetStatusCallback().GetKafka(),
			Http:  req.GetStatusCallback().GetHttp(),
		},
		Actions: actions,
	}

	// Getting request id
	id := req.GetClassId()

	// Returning error and logging if failed to update request
	if configuration, err := services.UpdateConfiguration(ctx, &request, id); err != nil {
		return nil, err
	} else {
		// Returning success and logging if request updated successfully
		log.Info(configuration)

		// Converting actions from struct to proto
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

		return &pb.Configuration{
			Id:       configuration.Id.Hex(),
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
			Actions: responseActions,
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
			CreatedAt: configuration.CreatedAt.String(),
			UpdatedAt: configuration.UpdatedAt.String(),
		}, nil
	}
}

// DeleteConfiguration : Call the service to delete an existing message class configuration
func (s *ConfigurationServer) DeleteConfiguration(ctx context.Context, req *pb.DeleteConfigurationRequest) (*pb.DeleteConfigurationResponse, error) {
	// Getting configuration id
	id := req.GetId()

	// Returning error and logging if failed to delete configuration
	if err := services.DeleteConfiguration(ctx, id); err != nil {
		return nil, err
	}

	// Returning success and logging if configuration deleted successfully
	log.Info("Id: " + id + " deleted successfully")
	return &pb.DeleteConfigurationResponse{Message: "Configuration deleted successfully!"}, nil
}

// QueryConfigurations : Call the service to query message class configurations
func (s *ConfigurationServer) QueryConfigurations(ctx context.Context, req *pb.QueryConfigurationRequest) (*pb.QueryConfigurationResponse, error) {
	// Converting query from proto to struct
	query := models.QueryConfigurationsParams{
		Name:     req.GetName(),
		Type:     req.GetType(),
		Mode:     req.GetMode(),
		AppId:    req.GetAppId(),
		ParentId: req.GetParentId(),
		Page:     req.GetPage(),
		Limit:    req.GetLimit(),
		Order:    req.GetOrder(),
		Sort:     req.GetSort(),
	}

	// Returning error and logging if failed to query configurations
	configurations, err := services.QueryConfigurations(ctx, &query)
	if err != nil {
		return nil, err
	}

	// Returning success and logging if configurations queried successfully
	log.Info(configurations)
	return configurations, nil
}

// GetConfiguration : Call the service to get a message class configuration
func (s *ConfigurationServer) GetConfiguration(ctx context.Context, req *pb.GetConfigurationRequest) (*pb.Configuration, error) {
	// Getting configuration id
	id := req.GetId()

	// Returning error and logging if failed to get configuration
	configuration, err := services.GetConfiguration(ctx, id)
	if err != nil {
		return nil, err
	}

	// Returning success and logging if configuration got successfully
	log.Info(configuration)
	return configuration, nil
}
