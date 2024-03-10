package controllers

import (
	"context"
	"messenger/restful/models"
	"messenger/restful/services"
	pb "messenger/services"
	"time"
)

type SceneServer struct {
	pb.SceneServiceServer
}
type SceneRelationServer struct {
	pb.SceneRelationServiceServer
}

func (s *SceneServer) CreateScene(ctx context.Context, req *pb.C2S_CreateSceneRequest) (*pb.Scene, error) {
	// Converting Scene from proto to struct
	scene := services.SceneCreate{
		Name:        req.GetName(),
		Describe:    req.GetDescribe(),
		SupportArgs: req.GetSupportArgs(),
		AppId:       req.GetAppId(),
		InstanceId:  req.GetInstanceId(),
		TenantId:    req.GetTenantId(),
		CategoryId:  req.GetCategoryId(),
		CreatedBy:   req.GetCreatedBy(),
	}

	// Returning error and logging if failed to create scene
	ans, err := services.CreateScene(ctx, &scene)
	if err != nil {
		return nil, err
	}
	// Returning success and logging if succeeded to create scene
	log.Info(scene)
	return &pb.Scene{
		Id:          ans.Id.Hex(),
		Name:        ans.Name,
		Describe:    ans.Describe,
		SupportArgs: ans.SupportArgs,
		AppId:       ans.AppId,
		InstanceId:  ans.InstanceId,
		TenantId:    ans.TenantId,
		CategoryId:  ans.CategoryId,
		CreatedBy:   ans.CreatedBy,
		CreatedAt:   ans.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   ans.UpdatedAt.Format(time.RFC3339),
		UpdateBy:    ans.UpdateBy,
	}, nil
}
func (s *SceneServer) GetScene(ctx context.Context, req *pb.C2S_GetSceneRequest) (*pb.Scene, error) {
	// Converting template from proto to struct
	// Returning error and logging if failed to create template
	ans, err := services.GetScene(ctx, req.GetId())
	if err != nil {
		return nil, err
	}

	// Returning success and logging if succeeded to create template
	log.Info(req.GetId())
	return &pb.Scene{
		Id:          ans.Id.Hex(),
		Name:        ans.Name,
		Describe:    ans.Describe,
		SupportArgs: ans.SupportArgs,
		AppId:       ans.AppId,
		InstanceId:  ans.InstanceId,
		TenantId:    ans.TenantId,
		CategoryId:  ans.CategoryId,
		CreatedBy:   ans.CreatedBy,
		CreatedAt:   ans.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   ans.UpdatedAt.Format(time.RFC3339),
		UpdateBy:    ans.UpdateBy,
	}, nil
}
func (s *SceneServer) UpdateScene(ctx context.Context, req *pb.C2S_UpdateSceneRequest) (*pb.Scene, error) {
	// Converting template from proto to struct
	scene := services.SceneUpdate{
		Id:         req.GetId(),
		Name:       req.Name,
		Describe:   req.Describe,
		AppId:      req.GetAppId(),
		InstanceId: req.GetInstanceId(),
		TenantId:   req.GetTenantId(),
		CategoryId: req.CategoryId,
		UpdateBy:   req.GetUpdateBy(),
	}
	if req.SupportArgs != nil {
		if req.SupportArgs[0] == "__empty__" {
			tmp := make([]string, 0)
			scene.SupportArgs = &tmp
		} else {
			scene.SupportArgs = &req.SupportArgs
		}
	}

	// Returning error and logging if failed to create template
	err := services.UpdateScene(ctx, scene)
	if err != nil {
		return nil, err
	}
	ans, err := services.GetScene(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	// Returning success and logging if succeeded to create template
	log.Info(scene)
	return &pb.Scene{
		Id:          ans.Id.Hex(),
		Name:        ans.Name,
		Describe:    ans.Describe,
		SupportArgs: ans.SupportArgs,
		AppId:       ans.AppId,
		InstanceId:  ans.InstanceId,
		TenantId:    ans.TenantId,
		CategoryId:  ans.CategoryId,
		CreatedBy:   ans.CreatedBy,
		CreatedAt:   ans.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   ans.UpdatedAt.Format(time.RFC3339),
		UpdateBy:    ans.UpdateBy,
	}, nil
}
func (s *SceneServer) DeleteScene(ctx context.Context, req *pb.C2S_DeleteSceneRequest) (*pb.S2C_DeleteSceneResponse, error) {
	// Returning error and logging if failed to create template
	err := services.DeleteScene(ctx, req.GetId())
	if err != nil {
		return &pb.S2C_DeleteSceneResponse{
			Message: "failed",
		}, err
	}

	// Returning success and logging if succeeded to create template
	log.Info(req.GetId())
	return &pb.S2C_DeleteSceneResponse{
		Message: "success",
	}, nil
}
func (s *SceneServer) QueryScenes(ctx context.Context, req *pb.C2S_QuerySceneRequest) (*pb.S2C_QuerySceneResponse, error) {
	// Converting template from proto to struct
	skip := (req.Page - 1) * req.Limit
	if skip < 0 {
		skip = 0
	}
	scene := services.SceneQuery{
		Id:         req.Id,
		Name:       req.Name,
		AppId:      req.AppId,
		InstanceId: req.InstanceId,
		TenantId:   req.TenantId,
		CategoryId: req.CategoryId,
		Skip:       &skip,
		Limit:      &req.Limit,
	}

	// Returning error and logging if failed to create template
	ans, err := services.QueryScenePage(ctx, scene)
	if err != nil {
		return nil, err
	}

	// Returning success and logging if succeeded to create template
	log.Info(scene)
	pages := pb.S2C_QuerySceneResponse{
		Page:  int64(ans.PageNum),
		Total: int64(ans.Total),
		Limit: int64(ans.PageSize),
	}
	scenes := []*pb.Scene{}
	for _, v := range ans.Records {
		scenes = append(scenes, &pb.Scene{
			Id:          v.Id.Hex(),
			Name:        v.Name,
			Describe:    v.Describe,
			SupportArgs: v.SupportArgs,
			AppId:       v.AppId,
			InstanceId:  v.InstanceId,
			TenantId:    v.TenantId,
			CategoryId:  v.CategoryId,
			CreatedBy:   v.CreatedBy,
			CreatedAt:   v.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   v.UpdatedAt.Format(time.RFC3339),
			UpdateBy:    v.UpdateBy,
		})
	}
	pages.Records = scenes

	return &pages, nil
}
func (s *SceneRelationServer) CreateSceneRelation(ctx context.Context, req *pb.C2S_CreateSceneRelationRequest) (*pb.SceneRelation, error) {
	// Converting template from proto to struct

	scene := services.SceneRelationCreate{
		SceneId:    req.GetSceneId(),
		TemplateId: req.GetTemplateId(),
		ConfigId:   req.GetConfigId(),
		// ConfigSubId: req.GetConfigSubId(),
		Source:     req.GetSource(),
		Target:     req.GetTarget(),
		AppId:      req.GetAppId(),
		InstanceId: req.GetInstanceId(),
		TenantId:   req.GetTenantId(),
		CreatedBy:  req.GetCreatedBy(),
		CreatedAt:  time.Now(),
	}
	if req.CustomEmailRequest != nil {
		scene.CustomEmailRequest = &models.CustomEmailRequest{
			Subject:     req.CustomEmailRequest.Subject,
			BodyType:    req.CustomEmailRequest.BodyType,
			Cc:          req.CustomEmailRequest.Cc,
			Bcc:         req.CustomEmailRequest.Bcc,
			Attachments: req.CustomEmailRequest.Attachments,
		}
	}
	if req.CustomSmsRequest != nil {
		scene.CustomSmsRequest = &models.CustomSmsRequest{
			TemplateParams: req.CustomSmsRequest.TemplateParams,
		}
	}
	if req.CustomFeishuRequest != nil {
		scene.CustomFeishuRequest = &models.CustomFeishuRequest{
			ReceiveIdType: req.CustomFeishuRequest.ReceiveIdType,
			MsgType:       req.CustomFeishuRequest.MsgType,
			CompanyId:     req.CustomFeishuRequest.CompanyId,
		}
	}
	if req.CustomDingTalkRequest != nil {
		scene.CustomDingtalkRequest = &models.CustomDingtalkRequest{
			CompanyId: req.CustomDingTalkRequest.CompanyId,
		}
	}

	// Returning error and logging if failed to create template
	ans, err := services.CreateSceneRelation(ctx, scene)
	if err != nil {
		return nil, err
	}

	// Returning success and logging if succeeded to create template
	log.Info(scene)
	return &pb.SceneRelation{
		SceneId:    ans.SceneId,
		TemplateId: ans.TemplateId,
		ConfigId:   ans.ConfigId,
		// ConfigSubId: ans.ConfigSubId,
		Source: ans.Source,
		Target: ans.Target,
		CustomEmailRequest: &pb.CustomEmailRequest{
			Subject:     ans.CustomEmailRequest.Subject,
			BodyType:    ans.CustomEmailRequest.BodyType,
			Cc:          ans.CustomEmailRequest.Cc,
			Bcc:         ans.CustomEmailRequest.Bcc,
			Attachments: ans.CustomEmailRequest.Attachments,
		},
		CustomSmsRequest: &pb.CustomSmsRequest{
			TemplateParams: ans.CustomSmsRequest.TemplateParams,
		},
		CustomFeishuRequest: &pb.CustomFeishuRequest{
			ReceiveIdType: ans.CustomFeishuRequest.ReceiveIdType,
			MsgType:       ans.CustomFeishuRequest.MsgType,
			CompanyId:     ans.CustomFeishuRequest.CompanyId,
		},
		CustomDingTalkRequest: &pb.CustomDingTalkRequest{
			CompanyId: ans.CustomDingtalkRequest.CompanyId,
		},
		AppId:      ans.AppId,
		InstanceId: ans.InstanceId,
		TenantId:   ans.TenantId,
		CreatedBy:  ans.CreatedBy,
		CreatedAt:  ans.CreatedAt.Format(time.RFC3339),
		UpdatedAt:  ans.UpdatedAt.Format(time.RFC3339),
		UpdateBy:   ans.UpdateBy,
	}, nil
}

func (s *SceneRelationServer) GetSceneRelation(ctx context.Context, req *pb.C2S_GetSceneRelationRequest) (*pb.SceneRelation, error) {
	// Converting template from proto to struct
	// Returning error and logging if failed to create template
	ans, err := services.GetSceneRelation(ctx, req.GetSceneId(), req.AppId)
	if err != nil {
		return nil, err
	}

	// Returning success and logging if succeeded to create template
	log.Info(req.GetSceneId())
	return &pb.SceneRelation{
		SceneId:    ans.SceneId,
		TemplateId: ans.TemplateId,
		ConfigId:   ans.ConfigId,
		// ConfigSubId: ans.ConfigSubId,
		Source: ans.Source,
		Target: ans.Target,
		CustomEmailRequest: &pb.CustomEmailRequest{
			Subject:     ans.CustomEmailRequest.Subject,
			BodyType:    ans.CustomEmailRequest.BodyType,
			Cc:          ans.CustomEmailRequest.Cc,
			Bcc:         ans.CustomEmailRequest.Bcc,
			Attachments: ans.CustomEmailRequest.Attachments,
		},
		CustomSmsRequest: &pb.CustomSmsRequest{
			TemplateParams: ans.CustomSmsRequest.TemplateParams,
		},
		CustomFeishuRequest: &pb.CustomFeishuRequest{
			ReceiveIdType: ans.CustomFeishuRequest.ReceiveIdType,
			MsgType:       ans.CustomFeishuRequest.MsgType,
			CompanyId:     ans.CustomFeishuRequest.CompanyId,
		},
		CustomDingTalkRequest: &pb.CustomDingTalkRequest{
			CompanyId: ans.CustomDingtalkRequest.CompanyId,
		},
		AppId:      ans.AppId,
		InstanceId: ans.InstanceId,
		TenantId:   ans.TenantId,
		CreatedBy:  ans.CreatedBy,
		CreatedAt:  ans.CreatedAt.Format(time.RFC3339),
		UpdatedAt:  ans.UpdatedAt.Format(time.RFC3339),
		UpdateBy:   ans.UpdateBy,
	}, nil
}

func (s *SceneRelationServer) UpdateSceneRelation(ctx context.Context, req *pb.C2S_UpdateSceneRelationRequest) (*pb.SceneRelation, error) {
	// Converting template from proto to struct
	scene := services.SceneRelationUpdate{
		SceneId:    req.GetSceneId(),
		TemplateId: req.TemplateId,
		ConfigId:   req.ConfigId,
		// ConfigSubId: req.ConfigSubId,
		Source:     req.Source,
		AppId:      req.GetAppId(),
		InstanceId: req.GetInstanceId(),
		TenantId:   req.GetTenantId(),
		UpdateBy:   req.UpdateBy,
	}
	if req.Target != nil {
		if req.Target[0] == "__empty__" {
			tmp := make([]string, 0)
			scene.Target = &tmp
		} else {
			scene.Target = &req.Target
		}
	}
	if req.CustomEmailRequest != nil {
		scene.CustomEmailRequest = &models.CustomEmailRequest{
			Subject:     req.CustomEmailRequest.Subject,
			BodyType:    req.CustomEmailRequest.BodyType,
			Cc:          req.CustomEmailRequest.Cc,
			Bcc:         req.CustomEmailRequest.Bcc,
			Attachments: req.CustomEmailRequest.Attachments,
		}
	}
	if req.CustomSmsRequest != nil {
		scene.CustomSmsRequest = &models.CustomSmsRequest{
			TemplateParams: req.CustomSmsRequest.TemplateParams,
		}
	}
	if req.CustomFeishuRequest != nil {
		scene.CustomFeishuRequest = &models.CustomFeishuRequest{
			ReceiveIdType: req.CustomFeishuRequest.ReceiveIdType,
			MsgType:       req.CustomFeishuRequest.MsgType,
			CompanyId:     req.CustomFeishuRequest.CompanyId,
		}
	}
	if req.CustomDingTalkRequest != nil {
		scene.CustomDingtalkRequest = &models.CustomDingtalkRequest{
			CompanyId: req.CustomDingTalkRequest.CompanyId,
		}
	}
	// Returning error and logging if failed to create template
	err := services.UpdateSceneRelation(ctx, scene)
	if err != nil {
		return nil, err
	}
	ans, err := services.GetSceneRelation(ctx, req.GetSceneId(), req.AppId)
	if err != nil {
		return nil, err
	}
	// Returning success and logging if succeeded to create template
	log.Info(scene)
	return &pb.SceneRelation{
		SceneId:    ans.SceneId,
		TemplateId: ans.TemplateId,
		ConfigId:   ans.ConfigId,
		// ConfigSubId: ans.ConfigSubId,
		Source: ans.Source,
		Target: ans.Target,
		CustomEmailRequest: &pb.CustomEmailRequest{
			Subject:     ans.CustomEmailRequest.Subject,
			BodyType:    ans.CustomEmailRequest.BodyType,
			Cc:          ans.CustomEmailRequest.Cc,
			Bcc:         ans.CustomEmailRequest.Bcc,
			Attachments: ans.CustomEmailRequest.Attachments,
		},
		CustomSmsRequest: &pb.CustomSmsRequest{
			TemplateParams: ans.CustomSmsRequest.TemplateParams,
		},
		CustomFeishuRequest: &pb.CustomFeishuRequest{
			ReceiveIdType: ans.CustomFeishuRequest.ReceiveIdType,
			MsgType:       ans.CustomFeishuRequest.MsgType,
			CompanyId:     ans.CustomFeishuRequest.CompanyId,
		},
		CustomDingTalkRequest: &pb.CustomDingTalkRequest{
			CompanyId: ans.CustomDingtalkRequest.CompanyId,
		},
		AppId:      ans.AppId,
		InstanceId: ans.InstanceId,
		TenantId:   ans.TenantId,
		CreatedBy:  ans.CreatedBy,
		CreatedAt:  ans.CreatedAt.Format(time.RFC3339),
		UpdatedAt:  ans.UpdatedAt.Format(time.RFC3339),
		UpdateBy:   ans.UpdateBy,
	}, nil
}

func (s *SceneRelationServer) DeleteSceneRelation(ctx context.Context, req *pb.C2S_DeleteSceneRelationRequest) (*pb.S2C_DeleteSceneRelationResponse, error) {
	// Returning error and logging if failed to create template
	err := services.DeleteSceneRelation(ctx, req.GetSceneId(), req.AppId)
	if err != nil {
		return &pb.S2C_DeleteSceneRelationResponse{
			Message: "failed",
		}, err
	}

	// Returning success and logging if succeeded to create template
	log.Info(req.GetSceneId())
	return &pb.S2C_DeleteSceneRelationResponse{
		Message: "success",
	}, nil
}

func (s *SceneRelationServer) QuerySceneRelations(ctx context.Context, req *pb.C2S_QuerySceneRelationRequest) (*pb.S2C_QuerySceneRelationResponse, error) {
	// Converting template from proto to struct
	skip := (req.Page - 1) * req.Limit
	if skip < 0 {
		skip = 0
	}
	scene := services.SceneRelationQuery{
		SceneId:    req.SceneId,
		TemplateId: req.TemplateId,
		ConfigId:   req.ConfigId,
		// ConfigSubId: req.ConfigSubId,
		AppId:      req.GetAppId(),
		InstanceId: req.GetInstanceId(),
		TenantId:   req.GetTenantId(),
		Skip:       &skip,
		Limit:      &req.Limit,
	}

	// Returning error and logging if failed to create template
	ans, err := services.QuerySceneRelationPage(ctx, scene)
	if err != nil {
		return nil, err
	}

	// Returning success and logging if succeeded to create template
	log.Info(scene)
	pages := pb.S2C_QuerySceneRelationResponse{
		Page:  int64(ans.PageNum),
		Total: int64(ans.Total),
		Limit: int64(ans.PageSize),
	}
	scenes := []*pb.SceneRelation{}
	for _, v := range ans.Records {
		scenes = append(scenes, &pb.SceneRelation{
			SceneId:    v.SceneId,
			TemplateId: v.TemplateId,
			ConfigId:   v.ConfigId,
			// ConfigSubId: v.ConfigSubId,
			Source: v.Source,
			Target: v.Target,
			CustomEmailRequest: &pb.CustomEmailRequest{
				Subject:     v.CustomEmailRequest.Subject,
				BodyType:    v.CustomEmailRequest.BodyType,
				Cc:          v.CustomEmailRequest.Cc,
				Bcc:         v.CustomEmailRequest.Bcc,
				Attachments: v.CustomEmailRequest.Attachments,
			},
			CustomSmsRequest: &pb.CustomSmsRequest{
				TemplateParams: v.CustomSmsRequest.TemplateParams,
			},
			CustomFeishuRequest: &pb.CustomFeishuRequest{
				ReceiveIdType: v.CustomFeishuRequest.ReceiveIdType,
				MsgType:       v.CustomFeishuRequest.MsgType,
				CompanyId:     v.CustomFeishuRequest.CompanyId,
			},
			CustomDingTalkRequest: &pb.CustomDingTalkRequest{
				CompanyId: v.CustomDingtalkRequest.CompanyId,
			},
			AppId:      v.AppId,
			InstanceId: v.InstanceId,
			TenantId:   v.TenantId,
			CreatedBy:  v.CreatedBy,
			CreatedAt:  v.CreatedAt.Format(time.RFC3339),
			UpdatedAt:  v.UpdatedAt.Format(time.RFC3339),
			UpdateBy:   v.UpdateBy,
		})
	}
	pages.Records = scenes

	return &pages, nil
}
func (s *SceneRelationServer) CopySceneRelation(ctx context.Context, req *pb.C2S_CopySceneRelationRequest) (*pb.S2C_CopySceneRelationResponse, error) {
	// Converting template from proto to struct
	// Returning error and logging if failed to create template
	err := services.CopySceneRelation(ctx, req.GetOldAppId(), req.GetNewAppId(), req.ForceUpdate)
	if err != nil {
		return &pb.S2C_CopySceneRelationResponse{
			Result: false,
		}, err
	}

	// Returning success and logging if succeeded to create template
	log.Info(req.GetOldAppId(), req.GetNewAppId())
	return &pb.S2C_CopySceneRelationResponse{
		Result: true,
	}, nil
}
