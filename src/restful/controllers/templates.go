package controllers

// importing packages
import (
	"context"
	"messenger/restful/models"
	"messenger/restful/services"
	pb "messenger/services"
	"time"
)

// TemplateServer : Implementation of TemplateServer grpc
type TemplateServer struct {
	pb.TemplateServiceServer
}

// CreateTemplate : Call the service to create a new message template
func (s *TemplateServer) CreateTemplate(ctx context.Context, req *pb.CreateTemplateRequest) (*pb.Template, error) {
	// Converting template from proto to struct
	template := services.Template{
		MessageTemplateId: req.GetMessageTemplateId(),
		Name:              req.GetName(),
		Type:              req.GetType(),
		Content:           req.GetContent(),
		Params:            req.GetParams(),
		Language:          req.GetLanguage(),
		CreatedBy:         req.GetCreatedBy(),
		AppId:             req.GetAppId(),
	}

	// Returning error and logging if failed to create template
	if err := services.CreateTemplate(ctx, &template); err != nil {
		return nil, err
	}

	// Returning success and logging if succeeded to create template

	return &pb.Template{
		Id:                template.Id.Hex(),
		MessageTemplateId: template.MessageTemplateId,
		Name:              template.Name,
		Type:              template.Type,
		Params:            template.Params,
		Content:           template.Content,
		CreatedBy:         template.CreatedBy,
		Language:          template.Language,
		CreatedAt:         template.CreatedAt.Format(time.RFC3339),
		UpdatedAt:         template.UpdatedAt.Format(time.RFC3339),
		AppId:             template.AppId,
	}, nil
}

// UpdateTemplate : Call the service to update an existing message template
func (s *TemplateServer) UpdateTemplate(ctx context.Context, req *pb.UpdateTemplateRequest) (*pb.Template, error) {
	// Converting template from proto to struct
	template := services.Template{
		MessageTemplateId: req.GetMessageTemplateId(),
		Name:              req.GetName(),
		Type:              req.GetType(),
		Content:           req.GetContent(),
		Language:          req.GetLanguage(),
		CreatedBy:         req.GetCreatedBy(),
		Params:            req.GetParams(),
		AppId:             req.GetAppId(),
	}

	if req.Params != nil && req.Params[0] == "__empty__" {
		template.Params = make([]string, 0)
	}
	// Getting templateId
	id := req.GetMessageTemplateId()

	// Returning error and logging if failed to update template
	if err := services.UpdateTemplate(ctx, &template, id); err != nil {
		return nil, err
	}

	// Returning success and logging if succeeded to update template
	return &pb.Template{
		Id:                template.Id.Hex(),
		MessageTemplateId: template.MessageTemplateId,
		Name:              template.Name,
		Type:              template.Type,
		Params:            template.Params,
		Content:           template.Content,
		CreatedBy:         template.CreatedBy,
		Language:          template.Language,
		CreatedAt:         template.CreatedAt.Format(time.RFC3339),
		UpdatedAt:         template.UpdatedAt.Format(time.RFC3339),
		AppId:             template.AppId,
	}, nil
}

// DeleteTemplate : Call the service to delete an existing message template
func (s *TemplateServer) DeleteTemplate(ctx context.Context, req *pb.DeleteTemplateRequest) (*pb.DeleteTemplateResponse, error) {
	// Getting templateId
	id := req.GetId()

	// Returning error and logging if failed to delete template
	if err := services.DeleteTemplate(ctx, id); err != nil {
		return nil, err
	}

	// Returning success and logging if succeeded to delete template
	log.Info("Id: " + id + " template deleted successfully")
	return &pb.DeleteTemplateResponse{
		Message: "Template Deleted Successfully!",
	}, nil
}

// QueryTemplates : Call the service to query message templates
func (s *TemplateServer) QueryTemplates(ctx context.Context, req *pb.QueryTemplatesRequest) (*pb.QueryTemplatesResponse, error) {
	// Converting query from proto to struct
	query := models.QueryTemplateParams{
		Name:     req.GetName(),
		Type:     req.GetType(),
		Language: req.GetLanguage(),
		Page:     req.GetPage(),
		Limit:    req.GetLimit(),
		Order:    req.GetOrder(),
		Sort:     req.GetSort(),
		AppId:    req.GetAppId(),
	}

	// Returning error and logging if failed to query templates
	templates, err := services.QueryTemplates(ctx, &query)
	if err != nil {
		return nil, err
	}

	// Returning success and logging if succeeded to query templates
	return templates, nil
}

// GetTemplate : Call the service to get a message template
func (s *TemplateServer) GetTemplate(ctx context.Context, req *pb.GetTemplateRequest) (*pb.Template, error) {
	// Getting templateId
	id := req.GetId()

	// Returning error and logging if failed to get template
	template, err := services.GetTemplate(ctx, id)
	if err != nil {
		return nil, err
	}

	// Returning success and logging if succeeded to get template
	return template, nil
}
