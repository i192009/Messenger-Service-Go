package services

import (
	"context"
	"messenger/restful/models"
	pb "messenger/services"
	"net/http"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc/codes"
)

// Validation arrays of message templates attributes
var languages = []string{"en", "cn"}
var types = []string{"greeting", "reminder", "announcement"}

// Template is a struct for message templates
type Template struct {
	Id                primitive.ObjectID `json:"_id" bson:"_id,omitempty"`
	AppId             string             `json:"appId" bson:"appId"`
	MessageTemplateId string             `json:"messageTemplateId" validate:"required" bson:"messageTemplateId"`
	Name              string             `json:"name" validate:"required" bson:"name"`
	Type              string             `json:"type" validate:"required,lowercase" bson:"type"`
	Content           string             `json:"content" validate:"required" bson:"content"`
	Params            []string           `json:"params" bson:"params"`
	CreatedBy         string             `json:"createdBy" validate:"required" bson:"createdBy"`
	Language          string             `json:"language" validate:"required,lowercase" bson:"language"`
	CreatedAt         time.Time          `json:"createdAt" bson:"createdAt"`
	UpdatedAt         time.Time          `json:"updatedAt" bson:"updatedAt"`
}

// CreateTemplate : Calls the required functions to create a message template
func CreateTemplate(ctx context.Context, template *Template) error {
	// Returning error and logging if failed to validate request JSON
	if err := ValidateJSON(template); err != nil {
		return err
	}

	// Returning error and logging if failed to validate create template request
	if err :=
		validateCreateTemplate(ctx, template); err != nil {
		return err
	}

	// Returning error and logging if failed to insert message template
	if err := createTemplate(ctx, template); err != nil {
		return err
	}

	// Returning nil if successful
	return nil
}

// UpdateTemplate : Calls the required functions to update a message template
func UpdateTemplate(ctx context.Context, template *Template, id string) error {
	// Initializing template struct
	var find Template

	// Returning error and logging if failed to find message template
	if err := getTemplateByTemplateId(ctx, id, &find); err != nil {
		return err
	}

	// Returning error and logging if failed to validate request JSON
	// if err := ValidateJSON(template); err != nil {
	// 	return err
	// }

	// Returning error and logging if failed to validate update template request
	if err := validateUpdateTemplate(ctx, template, &find); err != nil {
		return err
	}

	// Returning error and logging if failed to update message template
	if err := updateTemplate(ctx, template); err != nil {
		return err
	}

	// Returning nil if successful
	return nil
}

// DeleteTemplate : Calls the required functions to delete a message template
func DeleteTemplate(ctx context.Context, id string) error {
	// Initializing template struct
	var find Template

	// Returning error and logging if failed to find message template
	if err := getTemplateByTemplateId(ctx, id, &find); err != nil {
		return err
	}

	// Returning error and logging if failed to delete message template
	if err := deleteTemplate(ctx, &find); err != nil {
		return err
	}

	// Returning nil if successful
	return nil
}

// QueryTemplates : Calls the required functions to query message templates
func QueryTemplates(ctx context.Context, query *models.QueryTemplateParams) (*pb.QueryTemplatesResponse, error) {
	// Returning error and logging if failed to query message queryTemplateResponse
	queryTemplateResponse, err := queryTemplates(ctx, query)
	if err != nil {
		return nil, err
	}

	// Converting struct queryTemplateResponse to protobuf queryTemplateResponse
	response := make([]*pb.Template, len(queryTemplateResponse.Results.([]Template)))
	for i, template := range queryTemplateResponse.Results.([]Template) {
		response[i] = &pb.Template{
			Id:                template.Id.Hex(),
			CreatedAt:         template.CreatedAt.Format(time.RFC3339),
			UpdatedAt:         template.UpdatedAt.Format(time.RFC3339),
			MessageTemplateId: template.MessageTemplateId,
			Name:              template.Name,
			Params:            template.Params,
			CreatedBy:         template.CreatedBy,
			Language:          template.Language,
			Type:              template.Type,
			Content:           template.Content,
			AppId:             template.AppId,
		}
	}

	// Returning queryTemplateResponse and nil if successful
	return &pb.QueryTemplatesResponse{
		Page:    query.Page,
		Limit:   query.Limit,
		Total:   int64(len(response)),
		Results: response,
	}, nil
}

// GetTemplate : Calls the required functions to get a message template
func GetTemplate(ctx context.Context, templateId string) (*pb.Template, error) {
	var find Template

	// Returning error and logging if failed to validate GetTemplate request
	if err := validateGetTemplate(ctx, templateId, &find); err != nil {
		return nil, err
	}

	// Returning error and logging if failed to get message template
	template, err := getTemplate(ctx, templateId, &find)
	if err != nil {
		return nil, err
	}

	// Returning template and nil if successful
	return template, nil
}

// validateCreateTemplate : Validate create template request
func validateCreateTemplate(ctx context.Context, template *Template) error {
	// Returning error and logging if message template id already exists
	if err := checkTemplateIdExists(ctx, template.MessageTemplateId, template); err != nil {
		return err
	}

	// Returning error and logging if type is not valid
	if err := validateTemplateType(template.Type); err != nil {
		return err
	}

	// Returning error and logging if language is not supported
	if err := validateTemplateLanguage(template.Language); err != nil {
		return err
	}

	// Setting template attributes
	template.Id = primitive.NewObjectID()
	template.CreatedAt = time.Now()
	template.UpdatedAt = time.Now()

	// Returning nil if successful
	return nil
}

// validateUpdateTemplate : Validate update template request
func validateUpdateTemplate(ctx context.Context, template *Template, find *Template) error {
	// Validating message template id if template id is updated
	// if template.MessageTemplateId != find.MessageTemplateId {
	// 	// Returning error and logging if message template id already exists
	// 	if err := checkTemplateIdExists(ctx, template.MessageTemplateId, template); err != nil {
	// 		return err
	// 	}
	// }

	// Returning error and logging if type is not valid
	if template.Type != "" {
		if err := validateTemplateType(template.Type); err != nil {
			return err
		}
	} else {
		template.Type = find.Type
	}

	// Returning error and logging if language is not supported
	if template.Language != "" {
		if err := validateTemplateLanguage(template.Language); err != nil {
			return err
		}
	} else {
		template.Language = find.Language
	}
	if template.Content == "" {
		template.Content = find.Content
	}
	if template.Params == nil {
		template.Params = find.Params
	}
	if template.Name == "" {
		template.Name = find.Name
	}
	// Setting template attributes
	template.UpdatedAt = time.Now()
	template.Id = find.Id
	template.CreatedAt = find.CreatedAt

	// Returning nil if successful
	return nil
}

// validateGetTemplate : Validate get template request
func validateGetTemplate(ctx context.Context, templateId string, find *Template) error {
	// Returning error and logging if failed to find message template
	if err := getTemplateByTemplateId(ctx, templateId, find); err != nil {
		return err
	}

	// Returning nil if successful
	return nil
}

// createTemplate : Create message template in database
func createTemplate(ctx context.Context, template *Template) error {
	// Returning error and logging if failed to insert message template
	if err := insertOneTemplate(ctx, template); err != nil {
		return err
	}

	// Returning nil if successful
	return nil
}

// updateTemplate : Update message template in database
func updateTemplate(ctx context.Context, template *Template) error {
	// Returning error and logging if failed to update message template
	if err := updateOneTemplate(ctx, template); err != nil {
		return err
	}

	// Returning nil if successful
	return nil
}

// deleteTemplate : Delete message template in database
func deleteTemplate(ctx context.Context, template *Template) error {
	if err := deleteOneTemplate(ctx, template); err != nil {
		return err
	}

	return nil
}

// queryTemplates : Query message templates in database
func queryTemplates(ctx context.Context, params *models.QueryTemplateParams) (models.QueryTemplateResponse, error) {
	// Initializing message templates array
	var templates []Template

	// Initializing query struct
	query := bson.M{}

	// Adding name to query if name is not empty
	if params.Name != "" {
		// Setting name to query
		query["name"] = params.Name
	}

	// Adding type to query if type is not empty and is valid
	if params.Type != "" {
		// Returning error and logging if type is not valid
		if err := validateTemplateType(params.Type); err != nil {
			return models.QueryTemplateResponse{}, err
		} else {
			// Setting type to query
			query["type"] = params.Type
		}
	}

	// Adding created by to query if created by is not empty
	if params.CreatedBy != "" {
		// Setting createdBy to query
		query["createdBy"] = params.CreatedBy
	}
	if params.AppId != "" {
		query["appId"] = params.AppId
	}
	// Adding language to query if language is not empty and is supported
	if params.Language != "" {
		// Returning error and logging if language is not supported
		if err := validateTemplateLanguage(params.Language); err != nil {
			return models.QueryTemplateResponse{}, err
		} else {
			// Setting language to query
			query["language"] = params.Language
		}
	}

	if strings.ToLower(params.Sort) != "" && (strings.ToLower(params.Sort) != "asc" && strings.ToLower(params.Sort) != "desc") {
		Log.Error("Sort must be one of the following: asc, desc. Default is asc")
		return models.QueryTemplateResponse{}, CreateError(http.StatusBadRequest, 100008)
	} else {
		if strings.ToLower(params.Sort) == "" {
			params.Sort = "asc"
		}
	}

	if params.Order == "" {
		params.Order = "updatedAt"
	}

	// Setting Options to query message templates by pages
	skip := (params.Page - 1) * params.Limit
	limit := params.Limit
	opts := options.Find().SetSkip(skip).SetLimit(limit)

	if params.Sort == "asc" {
		opts.SetSort(bson.D{{params.Order, 1}})
	} else {
		opts.SetSort(bson.D{{params.Order, -1}})
	}

	// Count total messages filtered by query
	total, err := templateCollection.CountDocuments(ctx, query)
	if err != nil {
		// Returning error if failed to count messages
		Log.Error(err.Error())
		return models.QueryTemplateResponse{}, CreateError(codes.Internal, 1001)
	}
	//opts.SetProjection(bson.M{})
	// Querying message templates from MongoDB
	result, err := templateCollection.Find(ctx, query, opts)

	// Returning error and logging if failed to find message templates
	if err != nil {
		Log.Error(err.Error())
		return models.QueryTemplateResponse{}, CreateError(codes.Internal, 1001)
	}

	// Iterating over result and adding to message templates array
	for result.Next(ctx) {
		var messageTemplate Template
		err := result.Decode(&messageTemplate)

		// Returning error and logging if failed to decode message template
		if err != nil {
			Log.Error(err.Error())
			return models.QueryTemplateResponse{}, CreateError(codes.Internal, 1000)
		}

		// Adding message template to message templates array
		templates = append(templates, messageTemplate)
	}

	// Returning error and logging if no message templates are found
	if len(templates) == 0 {
		Log.Error("No message templates found")
		return models.QueryTemplateResponse{}, CreateError(codes.NotFound, 100205)
	}
	queryTemplateResponse := models.QueryTemplateResponse{
		Total:   total,
		Page:    params.Page,
		Limit:   params.Limit,
		Results: templates,
	}
	// Returning message templates and nil if successful
	return queryTemplateResponse, nil
}

// getTemplate : Get message template from database
func getTemplate(_ context.Context, _ string, template *Template) (*pb.Template, error) {
	// Converting struct template to protobuf template
	protoTemplate := &pb.Template{
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
	}

	// Returning nil and proto template if successfully converted to protobuf template
	return protoTemplate, nil
}

// findTemplateById: Find message template by Id is a function which is used by the API to find a message template by Id from mongoDB
func findTemplateById(ctx context.Context, id string, template *Template) error {
	// Returning error and logging if failed to convert id to object id
	objectId, err := ConvertToObjectId(id)
	// Returning error and logging if failed to convert id to ObjectID
	if err != nil {
		return err
	}

	// Setting filter to find message template to update
	filter := bson.M{"_id": objectId}

	// Finding message template from MongoDB
	err = templateCollection.FindOne(ctx, filter).Decode(&template)

	// Returning error and logging if message template does not exist
	if err != nil {
		Log.Error(err.Error())
		return CreateError(codes.NotFound, 100204)
	}

	// Returning nil if successful
	return nil
}

// checkTemplateIdExists: Find message template by template Id is a function which is used by the API to find a message template by template Id from mongoDB
func checkTemplateIdExists(ctx context.Context, templateId string, template *Template) error {
	// Setting Filter
	filter := bson.M{"messageTemplateId": templateId}

	// Finding Message Class Configuration
	err := templateCollection.FindOne(ctx, filter).Decode(template)

	// Returning error and logging if message template id already exists
	if err == nil {
		Log.Error("Message template id already exists")
		return CreateError(codes.AlreadyExists, 100201)
	}

	// Returning nil if successful
	return nil
}

// getTemplateByTemplateId: Find message template by template Id is a function which is used by the API to find a message template by template Id from mongoDB
func getTemplateByTemplateId(ctx context.Context, templateId string, template *Template) error {
	// Setting Filter
	filter := bson.M{"messageTemplateId": templateId}

	// Returning error and logging if message template id does not exist
	err := templateCollection.FindOne(ctx, filter).Decode(&template)
	if err != nil {
		Log.Error(err.Error())
		return CreateError(codes.NotFound, 100204)
	}

	// Returning nil if successful
	return nil
}

// insertOneTemplate: Insert one message template is a function which is used by the API to insert a message template into mongoDB
func insertOneTemplate(ctx context.Context, template *Template) error {
	// Returning error and logging if failed to insert message template
	_, err := templateCollection.InsertOne(ctx, template)
	if err != nil {
		Log.Error(err.Error())
		return CreateError(codes.Internal, 1001)
	}

	// Returning nil if successful
	return nil
}

// updateOneTemplate: Update one message template is a function which is used by the API to update a message template in mongoDB
func updateOneTemplate(ctx context.Context, template *Template) error {
	// Setting filter to find message template to update
	filter := bson.M{"_id": template.Id}

	// Updating message template in MongoDB
	update := bson.D{{"$set", bson.D{
		{"messageTemplateId", template.MessageTemplateId},
		{"name", template.Name},
		{"type", template.Type},
		{"params", template.Params},
		{"content", template.Content},
		{"createdBy", template.CreatedBy},
		{"language", template.Language},
		{"updatedAt", template.UpdatedAt},
	}}}

	// Returning error and logging if failed to update message template
	_, err := templateCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		Log.Error(err.Error())
		return CreateError(codes.Internal, 1001)
	}

	// Returning nil if successful
	return nil
}

// deleteOneTemplate: Delete one message template is a function which is used by the API to delete a message template from mongoDB
func deleteOneTemplate(ctx context.Context, template *Template) error {
	// Setting filter to find message template to update
	filter := bson.M{"_id": template.Id}

	// Returning error and logging if failed to delete message template
	_, err := templateCollection.DeleteOne(ctx, filter)
	if err != nil {
		Log.Error(err.Error())
		return CreateError(codes.Internal, 1001)
	}

	// Returning nil if successful
	return nil
}

// validateTemplateType: Validate message template type is a function which is used by the API to validate a message template type
func validateTemplateType(templateType string) error {
	// Returning error and logging if message template type is invalid
	if !contains(types, templateType) {
		Log.Error("Invalid message template type")
		return CreateError(codes.InvalidArgument, 100202)
	} else {
		return nil
	}
}

// validateTemplateLanguage: Validate message template language is a function which is used by the API to validate a message template language
func validateTemplateLanguage(language string) error {
	// Returning error and logging if message template language is invalid
	if !contains(languages, language) {
		Log.Error("Invalid message template language")
		return CreateError(codes.InvalidArgument, 100203)
	} else {
		return nil
	}
}
