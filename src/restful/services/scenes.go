package services

import (
	"context"
	"fmt"
	"messenger/restful/models"
	"time"

	"github.com/bytedance/sonic"
	"gitlab.zixel.cn/go/framework"
	"gitlab.zixel.cn/go/framework/xutil"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc/codes"
)

const (
	SCENE_FIELD_ID          = "_id"
	SCENE_FIELD_NAME        = "name"
	SCENE_FIELD_DESCRIBE    = "describe"
	SCENE_FIELD_SUPPORTARGS = "supportArgs"
	SCENE_FIELD_APPID       = "appId"
	SCENE_FIELD_INSTANCEID  = "instanceId"
	SCENE_FIELD_TENANTID    = "tenantId"
	SCENE_FIELD_CATEGORYID  = "categoryId"
	SCENE_FIELD_CREATEDBY   = "createdBy"
	SCENE_FIELD_UPDATEDBY   = "updateBy"
	SCENE_FIELD_CREATEDAT   = "createdAt"
	SCENE_FIELD_UPDATEDAT   = "updatedAt"

	SCENE_RELATION_FIELD_SCENEID               = "sceneId"
	SCENE_RELATION_FIELD_TEMPLATEID            = "templateId"
	SCENE_RELATION_FIELD_CONFIGID              = "configId"
	SCENE_RELATION_FIELD_CONFIGSUBID           = "configSubId"
	SCENE_RELATION_FIELD_APPID                 = "appId"
	SCENE_RELATION_FIELD_INSTANCEID            = "instanceId"
	SCENE_RELATION_FIELD_TENANTID              = "tenantId"
	SCENE_RELATION_FIELD_CREATEDBY             = "createdBy"
	SCENE_RELATION_FIELD_UPDATEDBY             = "updateBy"
	SCENE_RELATION_FIELD_CREATEDAT             = "createdAt"
	SCENE_RELATION_FIELD_UPDATEDAT             = "updatedAt"
	SCENE_RELATION_FILED_CUSTOMEEMAILREQUEST   = "customEmailRequest"
	SCENE_RELATION_FILED_CUSTOMESMSREQUEST     = "customSmsRequest"
	SCENE_RELATION_FILED_CUSTOMFEISHUREQUEST   = "customFeishuRequest"
	SCENE_RELATION_FILED_CUSTOMDINGTALKREQUEST = "customDingtalkRequest"
	SCENE_RELATION_FIELD_SOURCE                = "source"
	SCENE_RELATION_FIELD_TARGET                = "target"
)

// Scene : Scene struct describe a scene wich is used to send message and link to a template and config
type Scene struct {
	Id          primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Name        string             `bson:"name,omitempty" json:"name,omitempty"`
	Describe    string             `bson:"describe,omitempty" json:"describe,omitempty"`       //描述发送消息的场景，和使用场景
	SupportArgs []string           `bson:"supportArgs,omitempty" json:"supportArgs,omitempty"` //说明支持的参数和相应的位置关系
	AppId       string             `bson:"appId,omitempty" json:"appId,omitempty"`
	InstanceId  string             `bson:"instanceId,omitempty" json:"instanceId,omitempty"`
	TenantId    string             `bson:"tenantId,omitempty" json:"tenantId,omitempty"`
	CategoryId  int64              `bson:"categoryId,omitempty" json:"categoryId,omitempty"` //场景的分类id
	CreatedBy   string             `json:"createdBy" validate:"required" bson:"createdBy"`
	UpdateBy    string             `json:"updateBy" validate:"required" bson:"updateBy"`
	CreatedAt   time.Time          `json:"createdAt" bson:"createdAt"`
	UpdatedAt   time.Time          `json:"updatedAt" bson:"updatedAt"`
}
type SceneQuery struct {
	Id         *string `json:"id,omitempty"`
	Name       *string `json:"name,omitempty"`
	AppId      string  `bson:"appId,omitempty" json:"appId,omitempty"`
	InstanceId string  `bson:"instanceId,omitempty" json:"instanceId,omitempty"`
	TenantId   string  `bson:"tenantId,omitempty" json:"tenantId,omitempty"`
	CategoryId *int64  `bson:"categoryId,omitempty" json:"categoryId,omitempty"` //场景的分类id
	Skip       *int64  `json:"skip,omitempty"`
	Limit      *int64  `json:"limit,omitempty"`
}
type SceneCreate struct {
	Name        string   `bson:"name,omitempty" json:"name,omitempty"`
	Describe    string   `bson:"describe,omitempty" json:"describe,omitempty"`       //描述发送消息的场景，和使用场景
	SupportArgs []string `bson:"supportArgs,omitempty" json:"supportArgs,omitempty"` //说明支持的参数和相应的位置关系
	AppId       string   `bson:"appId,omitempty" json:"appId,omitempty"`
	InstanceId  string   `bson:"instanceId,omitempty" json:"instanceId,omitempty"`
	TenantId    string   `bson:"tenantId,omitempty" json:"tenantId,omitempty"`
	CategoryId  int64    `bson:"categoryId,omitempty" json:"categoryId,omitempty"` //场景的分类id
	CreatedBy   string   `json:"createdBy" validate:"required" bson:"createdBy"`
}
type SceneUpdate struct {
	Id          string    `json:"id,omitempty"`
	Name        *string   `json:"name,omitempty"`
	Describe    *string   `json:"describe,omitempty"`    //描述发送消息的场景，和使用场景
	SupportArgs *[]string `json:"supportArgs,omitempty"` //说明支持的参数和相应的位置关系
	AppId       string    `json:"appId,omitempty"`
	InstanceId  string    `json:"instanceId,omitempty"`
	TenantId    string    `json:"tenantId,omitempty"`
	CategoryId  *int64    `json:"categoryId,omitempty"` //场景的分类id
	UpdateBy    string    `json:"updateBy" validate:"required"`
}
type ScenePage struct {
	PageVO  `json:",inline"`
	Records []Scene `json:"records"`
}

// SceneRelation : SceneRelation struct describe a scene wich is used to send message and link to a template and config
type SceneRelation struct {
	SceneId    string `bson:"sceneId,omitempty" json:"sceneId,omitempty"`
	TemplateId string `bson:"templateId,omitempty" json:"templateId,omitempty"`
	ConfigId   string `bson:"configId,omitempty" json:"configId,omitempty"`
	// ConfigSubId           string                       `bson:"configSubId,omitempty" json:"configSubId,omitempty"`
	Source                string                       `json:"source" validate:"required" bson:"source"`
	Target                []string                     `json:"target" bson:"target"`
	CustomEmailRequest    models.CustomEmailRequest    `json:"customEmailRequest,omitempty" bson:"customEmailRequest,omitempty"`
	CustomSmsRequest      models.CustomSmsRequest      `json:"customSmsRequest,omitempty" bson:"customSmsRequest,omitempty"`
	CustomFeishuRequest   models.CustomFeishuRequest   `json:"customFeishuRequest,omitempty" bson:"customFeishuRequest,omitempty"`
	CustomDingtalkRequest models.CustomDingtalkRequest `json:"customDingtalkRequest,omitempty" bson:"customDingtalkRequest,omitempty"`
	AppId                 string                       `bson:"appId,omitempty" json:"appId,omitempty"`
	InstanceId            string                       `bson:"instanceId,omitempty" json:"instanceId,omitempty"`
	TenantId              string                       `bson:"tenantId,omitempty" json:"tenantId,omitempty"`
	CreatedBy             string                       `json:"createdBy" validate:"required" bson:"createdBy"`
	UpdateBy              string                       `json:"updateBy" validate:"required" bson:"updateBy"`
	CreatedAt             time.Time                    `json:"createdAt" bson:"createdAt"`
	UpdatedAt             time.Time                    `json:"updatedAt" bson:"updatedAt"`
}
type SceneRelationCreate struct {
	SceneId    string `bson:"sceneId,omitempty" json:"sceneId,omitempty"`
	TemplateId string `bson:"templateId,omitempty" json:"templateId,omitempty"`
	ConfigId   string `bson:"configId,omitempty" json:"configId,omitempty"`
	// ConfigSubId           string                        `bson:"configSubId,omitempty" json:"configSubId,omitempty"`
	Source                string                        `json:"source" validate:"required" bson:"source"`
	Target                []string                      `json:"target" bson:"target"`
	CustomEmailRequest    *models.CustomEmailRequest    `json:"customEmailRequest,omitempty" bson:"customEmailRequest,omitempty"`
	CustomSmsRequest      *models.CustomSmsRequest      `json:"customSmsRequest,omitempty" bson:"customSmsRequest,omitempty"`
	CustomFeishuRequest   *models.CustomFeishuRequest   `json:"customFeishuRequest,omitempty" bson:"customFeishuRequest,omitempty"`
	CustomDingtalkRequest *models.CustomDingtalkRequest `json:"customDingtalkRequest,omitempty" bson:"customDingtalkRequest,omitempty"`
	AppId                 string                        `bson:"appId,omitempty" json:"appId,omitempty"`
	InstanceId            string                        `bson:"instanceId,omitempty" json:"instanceId,omitempty"`
	TenantId              string                        `bson:"tenantId,omitempty" json:"tenantId,omitempty"`
	CreatedBy             string                        `json:"createdBy" validate:"required" bson:"createdBy"`
	CreatedAt             time.Time                     `json:"createdAt" bson:"createdAt"`
}
type SceneRelationUpdate struct {
	SceneId    string  `json:"sceneId,omitempty"`
	TemplateId *string `json:"templateId,omitempty"`
	ConfigId   *string `json:"configId,omitempty"`
	// ConfigSubId           *string                       `json:"configSubId,omitempty"`
	Source                *string                       `json:"source" validate:"required" bson:"source"`
	Target                *[]string                     `json:"target" bson:"target"`
	CustomEmailRequest    *models.CustomEmailRequest    `json:"customEmailRequest,omitempty" bson:"customEmailRequest,omitempty"`
	CustomSmsRequest      *models.CustomSmsRequest      `json:"customSmsRequest,omitempty" bson:"customSmsRequest,omitempty"`
	CustomFeishuRequest   *models.CustomFeishuRequest   `json:"customFeishuRequest,omitempty" bson:"customFeishuRequest,omitempty"`
	CustomDingtalkRequest *models.CustomDingtalkRequest `json:"customDingtalkRequest,omitempty" bson:"customDingtalkRequest,omitempty"`
	AppId                 string                        `json:"appId,omitempty"`
	InstanceId            string                        `json:"instanceId,omitempty"`
	TenantId              string                        `json:"tenantId,omitempty"`
	UpdateBy              string                        `json:"updateBy" validate:"required"`
	UpdatedAt             time.Time                     `json:"updatedAt"`
}
type SceneRelationQuery struct {
	SceneId    *string `json:"sceneId,omitempty"`
	TemplateId *string `json:"templateId,omitempty"`
	ConfigId   *string `json:"configId,omitempty"`
	// ConfigSubId *string `json:"configSubId,omitempty"`
	AppId      string `bson:"appId,omitempty" json:"appId,omitempty"`
	InstanceId string `bson:"instanceId,omitempty" json:"instanceId,omitempty"`
	TenantId   string `bson:"tenantId,omitempty" json:"tenantId,omitempty"`
	Skip       *int64 `json:"skip,omitempty"`
	Limit      *int64 `json:"limit,omitempty"`
}
type SceneRelationPage struct {
	PageVO  `json:",inline"`
	Records []SceneRelation `json:"records"`
}
type PageVO struct {
	PageNum   int32 `json:"pageNum,omitempty"`
	PageSize  int32 `json:"pageSize,omitempty"`
	TotalPage int32 `json:"totalPage,omitempty"`
	Total     int64 `json:"total,omitempty"`
}

func (s SceneUpdate) buildDocument() bson.M {
	ans := bson.M{}
	if s.Name != nil {
		ans[SCENE_FIELD_NAME] = *s.Name
	}
	if s.Describe != nil {
		ans[SCENE_FIELD_DESCRIBE] = *s.Describe
	}
	if s.SupportArgs != nil {
		ans[SCENE_FIELD_SUPPORTARGS] = *s.SupportArgs
	}
	if s.CategoryId != nil {
		ans[SCENE_FIELD_CATEGORYID] = *s.CategoryId
	}
	ans[SCENE_FIELD_UPDATEDBY] = s.UpdateBy
	ans[SCENE_FIELD_UPDATEDAT] = time.Now()
	return ans
}
func (s SceneRelationUpdate) buildDocument() bson.M {
	ans := bson.M{}
	if s.TemplateId != nil {
		ans[SCENE_RELATION_FIELD_TEMPLATEID] = *s.TemplateId
	}
	if s.ConfigId != nil {
		ans[SCENE_RELATION_FIELD_CONFIGID] = *s.ConfigId
	}
	// if s.ConfigSubId != nil {
	// 	ans[SCENE_RELATION_FIELD_CONFIGSUBID] = *s.ConfigSubId
	// }
	if s.Source != nil {
		ans[SCENE_RELATION_FIELD_SOURCE] = *s.Source
	}
	if s.Target != nil {
		ans[SCENE_RELATION_FIELD_TARGET] = *s.Target
	}
	if s.CustomEmailRequest != nil {
		ans[SCENE_RELATION_FILED_CUSTOMEEMAILREQUEST] = *s.CustomEmailRequest
	}
	if s.CustomSmsRequest != nil {
		ans[SCENE_RELATION_FILED_CUSTOMESMSREQUEST] = *s.CustomSmsRequest
	}
	if s.CustomFeishuRequest != nil {
		ans[SCENE_RELATION_FILED_CUSTOMFEISHUREQUEST] = *s.CustomFeishuRequest
	}
	if s.CustomDingtalkRequest != nil {
		ans[SCENE_RELATION_FILED_CUSTOMDINGTALKREQUEST] = *s.CustomDingtalkRequest
	}
	ans[SCENE_RELATION_FIELD_UPDATEDBY] = s.UpdateBy
	ans[SCENE_RELATION_FIELD_UPDATEDAT] = time.Now()
	return ans
}

// CreateScene : CreateScene function is used to create a new scene
func CreateScene(ctx context.Context, scene *SceneCreate) (Scene, error) {
	//var scene1 Scene
	Scene := Scene{}
	// Returning error and logging if failed to validate request JSON
	if err := ValidateJSON(scene); err != nil {
		return Scene, err
	}

	// Returning error and logging if failed to validate create template request
	if err := validateCreateScene(ctx, *scene); err != nil {
		return Scene, err
	}

	Scene, err := createScene(ctx, Scene, *scene)
	if err != nil {
		return Scene, err
	}
	return Scene, nil
}

// CreateSceneRelation : CreateSceneRelation function is used to create a new scene
func UpdateScene(ctx context.Context, scene SceneUpdate) error {
	// Returning error and logging if failed to validate request JSON
	// if err := ValidateJSON(scene); err != nil {
	// 	return err
	// }

	// Returning error and logging if failed to validate create template request
	if err := validateUpdateScene(ctx, scene); err != nil {
		return err
	}
	if err := updateScene(ctx, scene); err != nil {
		return err
	}
	return nil
}
func DeleteScene(ctx context.Context, idstr string) error {

	if err := deleteScene(ctx, idstr); err != nil {
		return err
	}
	//id := primitive.NewObjectID()
	//id.UnmarshalText([]byte(idstr))
	//_, err := sceneCollection.DeleteOne(ctx, bson.M{
	//	SCENE_FIELD_ID: id,
	//})
	//if err != nil {
	//	Log.Error(err.Error())
	//	return err
	//}
	return nil
}
func QueryScenePage(ctx context.Context, query SceneQuery) (ScenePage, error) {
	ans, err := queryScenePage(ctx, query)
	if err != nil {
		return ans, err
	}
	return ans, nil
}
func GetScene(ctx context.Context, sceneId string) (Scene, error) {
	ans, err := getScene(ctx, sceneId)
	if err != nil {
		return ans, err
	}
	return ans, nil
}
func CreateSceneRelation(ctx context.Context, scene SceneRelationCreate) (SceneRelation, error) {
	sceneRelation := SceneRelation{}
	// Returning error and logging if failed to validate request JSON
	if err := ValidateJSON(scene); err != nil {
		return sceneRelation, err
	}

	// Returning error and logging if failed to validate create template request
	if err := validateCreateSceneRelation(ctx, scene); err != nil {
		return sceneRelation, err
	}
	sceneRelation, err := createSceneRelation(ctx, sceneRelation, scene)
	if err != nil {
		return sceneRelation, err
	}
	return sceneRelation, nil
}

// CreateSceneRelation : CreateSceneRelation function is used to create a new scene
func UpdateSceneRelation(ctx context.Context, scene SceneRelationUpdate) error {
	// Returning error and logging if failed to validate request JSON
	// if err := ValidateJSON(scene); err != nil {
	// 	return err
	// }
	// Returning error and logging if failed to validate create template request
	if err := validateUpdateSceneRelation(ctx, scene); err != nil {
		return err
	}
	if err := updateSceneRelation(ctx, scene); err != nil {
		return err
	}
	return nil
}

func DeleteSceneRelation(ctx context.Context, idstr string, appId string) error {
	if err := deleteSceneRelation(ctx, idstr, appId); err != nil {
		return err
	}
	return nil
}
func QuerySceneRelationPage(ctx context.Context, query SceneRelationQuery) (SceneRelationPage, error) {
	ans, err := querySceneRelationPage(ctx, query)
	if err != nil {
		return ans, err
	}
	return ans, nil
}
func GetSceneRelation(ctx context.Context, sceneId string, appId string) (SceneRelation, error) {
	ans, err := getSceneRelation(ctx, sceneId, appId)
	if err != nil {
		return ans, err
	}
	return ans, nil
}
func CopySceneRelation(ctx context.Context, oldAppId string, newAppId string, forceUpdate bool) error {
	if err := copySceneRelation(ctx, oldAppId, newAppId, forceUpdate); err != nil {
		return err
	}
	return nil
}

// CreateSceneRelation : CreateSceneRelation function is used to create a new scene
func validateCreateScene(ctx context.Context, scene SceneCreate) error {
	// Returning error and logging if message template type is invalid
	if scene.Name == "" {
		Log.Error("name can not be empty")
		return CreateError(codes.InvalidArgument, 100202)
	}
	if scene.Describe == "" {
		Log.Error("describe can not be empty")
		return CreateError(codes.InvalidArgument, 100202)
	}
	if scene.AppId == "" {
		Log.Error("appId can not be empty")
		return CreateError(codes.InvalidArgument, 100202)
	}
	if scene.InstanceId == "" {
		Log.Error("instanceId can not be empty")
		return CreateError(codes.InvalidArgument, 100202)
	}
	if scene.TenantId == "" {
		Log.Error("tenantId can not be empty")
		return CreateError(codes.InvalidArgument, 100202)
	}
	if scene.CreatedBy == "" {
		Log.Error("createdBy can not be empty")
		return CreateError(codes.InvalidArgument, 100202)
	}
	return nil
}
func validateUpdateScene(ctx context.Context, scene SceneUpdate) error {
	// Returning error and logging if message template type is invalid
	if scene.Id == "" {
		Log.Error("id data can not be empty")
		return CreateError(codes.InvalidArgument, 100202)
	}
	if scene.AppId == "" {
		Log.Error("appId can not be empty")
		return CreateError(codes.InvalidArgument, 100202)
	}
	if scene.InstanceId == "" {
		Log.Error("instanceId can not be empty")
		return CreateError(codes.InvalidArgument, 100202)
	}
	if scene.TenantId == "" {
		Log.Error("tenantId can not be empty")
		return CreateError(codes.InvalidArgument, 100202)
	}
	if scene.UpdateBy == "" {
		Log.Error("updateBy can not be empty")
		return CreateError(codes.InvalidArgument, 100202)
	}
	return nil
}
func insertOneScene(ctx context.Context, scene Scene) error {
	// Returning error and logging if failed to insert message scene
	_, err := sceneCollection.InsertOne(ctx, scene)
	if err != nil {
		Log.Error(err.Error())
		if mongo.IsDuplicateKeyError(err) {
			return framework.NewServiceError(framework.ERR_SYS_DATABASE, "scene already exists in the same scope")
		} else {
			return framework.NewServiceError(framework.ERR_SYS_DATABASE, err.Error())
		}
	}

	// Returning nil if successful
	return nil
}
func updateOneScene(ctx context.Context, filter bson.M, docuemnts bson.M) error {
	// Returning error and logging if failed to insert message scene
	_, err := sceneCollection.UpdateOne(ctx, filter, docuemnts)
	if err != nil {
		Log.Error(err.Error())
		if mongo.IsDuplicateKeyError(err) {
			return framework.NewServiceError(framework.ERR_SYS_DATABASE, "scene already exists in the same scope")
		} else {
			return framework.NewServiceError(framework.ERR_SYS_DATABASE, err.Error())
		}
	}

	// Returning nil if successful
	return nil
}
func validateCreateSceneRelation(ctx context.Context, scene SceneRelationCreate) error {
	// Returning error and logging if message template type is invalid
	if scene.SceneId == "" {
		Log.Error("sceneId can not be empty")
		return CreateError(codes.InvalidArgument, 100202)
	}
	if scene.TemplateId == "" {
		Log.Error("templateId can not be empty")
		return CreateError(codes.InvalidArgument, 100202)
	}
	if scene.ConfigId == "" {
		Log.Error("configId can not be empty")
		return CreateError(codes.InvalidArgument, 100202)
	}
	if scene.Source == "" {
		Log.Error("source can not be empty")
		return CreateError(codes.InvalidArgument, 100202)
	}
	if scene.Target == nil {
		Log.Error("target can not be empty")
		return CreateError(codes.InvalidArgument, 100202)
	}
	if scene.CreatedBy == "" {
		Log.Error("createdBy can not be empty")
		return CreateError(codes.InvalidArgument, 100202)
	}
	return nil
}
func insertOneSceneRelation(ctx context.Context, scene SceneRelation) error {
	// Returning error and logging if failed to insert message scene
	_, err := sceneRelationCollection.InsertOne(ctx, scene)
	if err != nil {
		Log.Error(err.Error())
		if mongo.IsDuplicateKeyError(err) {
			return framework.NewServiceError(framework.ERR_SYS_DATABASE, "scene relation already exists in the same scope")
		} else {
			return framework.NewServiceError(framework.ERR_SYS_DATABASE, err.Error())
		}
	}

	// Returning nil if successful
	return nil
}
func validateUpdateSceneRelation(ctx context.Context, update SceneRelationUpdate) error {
	if update.SceneId == "" {
		Log.Error("sceneId can not be empty")
		return CreateError(codes.InvalidArgument, 100202)
	}
	if update.AppId == "" {
		Log.Error("appId can not be empty")
		return CreateError(codes.InvalidArgument, 100202)
	}
	// if update.InstanceId == "" {
	// 	Log.Error("instanceId can not be empty")
	// 	return CreateError(codes.InvalidArgument, 100202)
	// }
	// if update.TenantId == "" {
	// 	Log.Error("tenantId can not be empty")
	// 	return CreateError(codes.InvalidArgument, 100202)
	// }
	if update.UpdateBy == "" {
		Log.Error("updateBy can not be empty")
		return CreateError(codes.InvalidArgument, 100202)
	}
	return nil
}
func updateOneSceneRelation(ctx context.Context, filter bson.M, docuemnts bson.M) error {
	// Returning error and logging if failed to insert message scene
	_, err := sceneRelationCollection.UpdateOne(ctx, filter, docuemnts)
	if err != nil {
		Log.Error(err.Error())
		if mongo.IsDuplicateKeyError(err) {
			return framework.NewServiceError(framework.ERR_SYS_DATABASE, "scene relation already exists in the same scope")
		} else {
			return framework.NewServiceError(framework.ERR_SYS_DATABASE, err.Error())
		}
	}

	// Returning nil if successful
	return nil
}
func copySceneRelation(ctx context.Context, oldAppId string, newAppId string, forceUpdate bool) error {
	configurations := make([]any, 0)
	templates := make([]any, 0)
	sceneRelations := make([]any, 0)
	existConfigIds := make([]string, 0)
	existTemplateIds := make([]string, 0)
	existSceneIds := make([]string, 0)
	if forceUpdate {
		configurationCollection.DeleteMany(ctx, bson.M{"appId": fmt.Sprintf("%v", newAppId)})
		templateCollection.DeleteMany(ctx, bson.M{"appId": fmt.Sprintf("%v", newAppId)})
		sceneRelationCollection.DeleteMany(ctx, bson.M{"appId": fmt.Sprintf("%v", newAppId)})
	} else {
		cursor, err := configurationCollection.Find(ctx, bson.M{"appId": fmt.Sprintf("%v", newAppId)})
		if err != nil {
			// Returning error if failed to find message class configurations
			Log.Error(err.Error())
			return CreateError(codes.Internal, 1001)
		}

		// Iterating through message class configurations
		for cursor.Next(ctx) {
			var configuration Configuration
			// Decoding message class configuration
			err := cursor.Decode(&configuration)
			if err != nil {
				// Returning error if failed to decode message class configuration
				Log.Error(err.Error())
				return CreateError(codes.Internal, 1000)
			}
			existConfigIds = append(existConfigIds, configuration.ClassId)
		}
		cursor, err = templateCollection.Find(ctx, bson.M{"appId": fmt.Sprintf("%v", newAppId)})
		if err != nil {
			// Returning error if failed to find message class configurations
			Log.Error(err.Error())
			return CreateError(codes.Internal, 1001)
		}

		// Iterating through message class configurations
		for cursor.Next(ctx) {
			var template Template
			// Decoding message class configuration
			err := cursor.Decode(&template)
			if err != nil {
				// Returning error if failed to decode message class configuration
				Log.Error(err.Error())
				return CreateError(codes.Internal, 1000)
			}
			existTemplateIds = append(existTemplateIds, template.MessageTemplateId)
		}
		cursor, err = sceneRelationCollection.Find(ctx, bson.M{"appId": fmt.Sprintf("%v", newAppId)})
		if err != nil {
			// Returning error if failed to find message class configurations
			Log.Error(err.Error())
			return CreateError(codes.Internal, 1001)
		}

		// Iterating through message class configurations
		for cursor.Next(ctx) {
			var sceneRelation SceneRelation
			// Decoding message class configuration
			err := cursor.Decode(&sceneRelation)
			if err != nil {
				// Returning error if failed to decode message class configuration
				Log.Error(err.Error())
				return CreateError(codes.Internal, 1000)
			}
			// Appending message class configuration to message class configurations array
			existSceneIds = append(existSceneIds, sceneRelation.SceneId)
		}
	}
	cursor, err := configurationCollection.Find(ctx, bson.M{"appId": fmt.Sprintf("%v", oldAppId)})
	if err != nil {
		// Returning error if failed to find message class configurations
		Log.Error(err.Error())
		return CreateError(codes.Internal, 1001)
	}

	// Iterating through message class configurations
	for cursor.Next(ctx) {
		var configuration Configuration
		// Decoding message class configuration
		err := cursor.Decode(&configuration)
		if err != nil {
			// Returning error if failed to decode message class configuration
			Log.Error(err.Error())
			return CreateError(codes.Internal, 1000)
		}
		configuration.Id = primitive.NewObjectID()
		configuration.ClassId = configuration.ClassId + fmt.Sprintf("-%v", newAppId)
		configuration.AppId = fmt.Sprintf("%v", newAppId)
		// Appending message class configuration to message class configurations array
		if !xutil.InSlice(configuration.ClassId, existConfigIds) {
			configurations = append(configurations, configuration)
		}
	}
	cursor, err = templateCollection.Find(ctx, bson.M{"appId": fmt.Sprintf("%v", oldAppId)})
	if err != nil {
		// Returning error if failed to find message class configurations
		Log.Error(err.Error())
		return CreateError(codes.Internal, 1001)
	}

	// Iterating through message class configurations
	for cursor.Next(ctx) {
		var template Template
		// Decoding message class configuration
		err := cursor.Decode(&template)
		if err != nil {
			// Returning error if failed to decode message class configuration
			Log.Error(err.Error())
			return CreateError(codes.Internal, 1000)
		}
		template.Id = primitive.NewObjectID()
		template.MessageTemplateId = template.MessageTemplateId + fmt.Sprintf("-%v", newAppId)
		template.AppId = fmt.Sprintf("%v", newAppId)
		// Appending message class configuration to message class configurations array
		if !xutil.InSlice(template.MessageTemplateId, existTemplateIds) {
			templates = append(templates, template)
		}
	}
	cursor, err = sceneRelationCollection.Find(ctx, bson.M{"appId": fmt.Sprintf("%v", oldAppId)})
	if err != nil {
		// Returning error if failed to find message class configurations
		Log.Error(err.Error())
		return CreateError(codes.Internal, 1001)
	}

	// Iterating through message class configurations
	for cursor.Next(ctx) {
		var sceneRelation SceneRelation
		// Decoding message class configuration
		err := cursor.Decode(&sceneRelation)
		if err != nil {
			// Returning error if failed to decode message class configuration
			Log.Error(err.Error())
			return CreateError(codes.Internal, 1000)
		}
		sceneRelation.AppId = newAppId
		sceneRelation.ConfigId = sceneRelation.ConfigId + fmt.Sprintf("-%v", newAppId)
		sceneRelation.TemplateId = sceneRelation.TemplateId + fmt.Sprintf("-%v", newAppId)
		// Appending message class configuration to message class configurations array
		if !xutil.InSlice(sceneRelation.SceneId, existSceneIds) {
			sceneRelations = append(sceneRelations, sceneRelation)
		}
	}
	//insert new configurations
	if len(configurations) > 0 {
		_, err = configurationCollection.InsertMany(ctx, configurations)
		if err != nil {
			Log.Error(err.Error())
			configurationCollection.DeleteMany(ctx, bson.M{"appId": fmt.Sprintf("%v", newAppId)})
			templateCollection.DeleteMany(ctx, bson.M{"appId": fmt.Sprintf("%v", newAppId)})
			sceneRelationCollection.DeleteMany(ctx, bson.M{"appId": fmt.Sprintf("%v", newAppId)})
			return CreateError(codes.Internal, 1001)
		}
	}
	//insert new templates
	if len(templates) > 0 {
		_, err = templateCollection.InsertMany(ctx, templates)
		if err != nil {
			Log.Error(err.Error())
			configurationCollection.DeleteMany(ctx, bson.M{"appId": fmt.Sprintf("%v", newAppId)})
			templateCollection.DeleteMany(ctx, bson.M{"appId": fmt.Sprintf("%v", newAppId)})
			sceneRelationCollection.DeleteMany(ctx, bson.M{"appId": fmt.Sprintf("%v", newAppId)})
			return CreateError(codes.Internal, 1001)
		}
	}
	//insert new sceneRelations
	if len(sceneRelations) > 0 {
		_, err = sceneRelationCollection.InsertMany(ctx, sceneRelations)
		if err != nil {
			Log.Error(err.Error())
			configurationCollection.DeleteMany(ctx, bson.M{"appId": fmt.Sprintf("%v", newAppId)})
			templateCollection.DeleteMany(ctx, bson.M{"appId": fmt.Sprintf("%v", newAppId)})
			sceneRelationCollection.DeleteMany(ctx, bson.M{"appId": fmt.Sprintf("%v", newAppId)})
			return CreateError(codes.Internal, 1001)
		}
	}
	return nil

}
func deleteScene(ctx context.Context, idstr string) error {
	id := primitive.NewObjectID()
	id.UnmarshalText([]byte(idstr))
	_, err := sceneCollection.DeleteOne(ctx, bson.M{
		SCENE_FIELD_ID: id,
	})
	if err != nil {
		Log.Error(err.Error())
		return err
	}
	return nil
}
func updateScene(ctx context.Context, scene SceneUpdate) error {
	id := primitive.NewObjectID()
	id.UnmarshalText([]byte(scene.Id))
	// Returning error and logging if failed to insert message template
	if err := updateOneScene(ctx, bson.M{
		SCENE_FIELD_ID:         id,
		SCENE_FIELD_TENANTID:   scene.TenantId,
		SCENE_FIELD_INSTANCEID: scene.InstanceId,
		SCENE_FIELD_APPID:      scene.AppId,
	}, bson.M{"$set": scene.buildDocument()}); err != nil {
		return err
	}
	return nil
}
func createScene(ctx context.Context, createScene Scene, scene SceneCreate) (Scene, error) {
	reqData, err := sonic.Marshal(scene)
	if err != nil {
		Log.Error("parse scene error")
		return createScene, CreateError(codes.InvalidArgument, 100202)
	}
	err = sonic.Unmarshal(reqData, &createScene)
	if err != nil {
		Log.Error("parse scene error ")
		return createScene, CreateError(codes.InvalidArgument, 100202)
	}
	createScene.Id = primitive.NewObjectID()
	createScene.CreatedAt = time.Now()
	// Returning error and logging if failed to insert message template
	if err := insertOneScene(ctx, createScene); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return createScene, framework.NewServiceError(framework.ERR_SYS_DATABASE, "scene already exists in the same scope")
		} else {
			return createScene, framework.NewServiceError(framework.ERR_SYS_DATABASE, err.Error())
		}
	}
	return createScene, nil
}
func queryScenePage(ctx context.Context, query SceneQuery) (ScenePage, error) {
	filter := bson.M{}
	if query.Id != nil {
		id := primitive.NewObjectID()
		id.UnmarshalText([]byte(*query.Id))
		filter[SCENE_FIELD_ID] = id
	}
	if query.Name != nil {
		filter[SCENE_FIELD_NAME] = bson.M{"$regex": fmt.Sprintf(".*%v.*", *query.Name)}
	}
	filter[SCENE_FIELD_TENANTID] = query.TenantId
	filter[SCENE_FIELD_INSTANCEID] = query.InstanceId
	filter[SCENE_FIELD_APPID] = query.AppId
	if query.Skip == nil {
		query.Skip = new(int64)
		*query.Skip = 0
	}
	if query.Limit == nil {
		query.Limit = new(int64)
		*query.Limit = 10
	}
	if query.CategoryId != nil {
		filter[SCENE_FIELD_CATEGORYID] = *query.CategoryId
	}
	ans := ScenePage{
		PageVO: PageVO{
			PageSize: int32(*query.Limit),
		},
	}

	counter, err := sceneCollection.CountDocuments(ctx, filter)
	if err != nil {
		Log.Error(err.Error())
		return ans, err
	}
	ans.Total = counter
	if *query.Limit == 0 {
		ans.PageNum = 1
		ans.TotalPage = 1
	} else {
		ans.PageNum = int32(*query.Skip/(*query.Limit)) + 1
		ans.TotalPage = int32(counter / *query.Limit)
	}
	records := []Scene{}
	cursor, err := sceneCollection.Find(ctx, filter, &options.FindOptions{
		Skip:  query.Skip,
		Limit: query.Limit,
	})
	// Returning error and logging if failed to find message templates
	if err != nil {
		Log.Error(err.Error())
		return ans, CreateError(codes.Internal, 1001)
	}

	// Iterating over result and adding to message templates array
	for cursor.Next(ctx) {
		var scene Scene
		err := cursor.Decode(&scene)

		// Returning error and logging if failed to decode message template
		if err != nil {
			Log.Error(err.Error())
			return ans, CreateError(codes.Internal, 1000)
		}

		// Adding message template to message templates array
		records = append(records, scene)
	}
	ans.Records = records
	return ans, nil
}
func getScene(ctx context.Context, sceneId string) (Scene, error) {
	filter := bson.M{}
	id := primitive.NewObjectID()
	id.UnmarshalText([]byte(sceneId))
	filter[SCENE_FIELD_ID] = id
	ans := Scene{}
	res := sceneCollection.FindOne(ctx, filter)
	res.Decode(&ans)
	return ans, nil
}
func createSceneRelation(ctx context.Context, createSceneRelation SceneRelation, scene SceneRelationCreate) (SceneRelation, error) {
	reqData, err := sonic.Marshal(scene)
	if err != nil {
		Log.Error("parse scene error")
		return createSceneRelation, CreateError(codes.InvalidArgument, 100202)
	}
	err = sonic.Unmarshal(reqData, &createSceneRelation)
	if err != nil {
		Log.Error("parse scene error")
		return createSceneRelation, CreateError(codes.InvalidArgument, 100202)
	}
	createSceneRelation.CreatedAt = time.Now()
	// Returning error and logging if failed to insert message template
	if err := insertOneSceneRelation(ctx, createSceneRelation); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return createSceneRelation, framework.NewServiceError(framework.ERR_SYS_DATABASE, "scene relation already exists in the same scope")
		} else {
			return createSceneRelation, framework.NewServiceError(framework.ERR_SYS_DATABASE, err.Error())
		}
	}
	return createSceneRelation, nil
}
func updateSceneRelation(ctx context.Context, scene SceneRelationUpdate) error {
	filter := bson.M{
		SCENE_RELATION_FIELD_SCENEID: scene.SceneId,
		SCENE_RELATION_FIELD_APPID:   scene.AppId,
	}
	if scene.TenantId != "" {
		filter[SCENE_RELATION_FIELD_TENANTID] = scene.TenantId
	}
	if scene.InstanceId != "" {
		filter[SCENE_RELATION_FIELD_INSTANCEID] = scene.InstanceId
	}
	// Returning error and logging if failed to insert message template
	if err := updateOneSceneRelation(ctx, filter, bson.M{"$set": scene.buildDocument()}); err != nil {
		return err
	}
	return nil
}
func deleteSceneRelation(ctx context.Context, idstr string, appId string) error {
	_, err := sceneRelationCollection.DeleteOne(ctx, bson.M{
		SCENE_RELATION_FIELD_SCENEID: idstr,
		SCENE_FIELD_APPID:            appId,
	})
	if err != nil {
		Log.Error(err.Error())
		return err
	}
	return nil
}
func querySceneRelationPage(ctx context.Context, query SceneRelationQuery) (SceneRelationPage, error) {
	filter := bson.M{}
	if query.SceneId != nil {
		filter[SCENE_RELATION_FIELD_SCENEID] = query.SceneId
	}
	if query.TemplateId != nil {
		filter[SCENE_RELATION_FIELD_TEMPLATEID] = query.TemplateId
	}
	if query.ConfigId != nil {
		filter[SCENE_RELATION_FIELD_CONFIGID] = query.ConfigId
	}
	// if query.ConfigSubId != nil {
	// 	filter[SCENE_RELATION_FIELD_CONFIGSUBID] = query.ConfigSubId
	// }
	if query.TenantId != "" {
		filter[SCENE_RELATION_FIELD_TENANTID] = query.TenantId
	}
	if query.InstanceId != "" {
		filter[SCENE_RELATION_FIELD_INSTANCEID] = query.InstanceId
	}
	filter[SCENE_RELATION_FIELD_APPID] = query.AppId
	if query.Skip == nil {
		query.Skip = new(int64)
		*query.Skip = 0
	}
	if query.Limit == nil {
		query.Limit = new(int64)
		*query.Limit = 10
	}
	ans := SceneRelationPage{
		PageVO: PageVO{
			PageSize: int32(*query.Limit),
		},
	}
	counter, err := sceneRelationCollection.CountDocuments(ctx, filter)
	if err != nil {
		Log.Error(err.Error())
		return ans, err
	}
	if *query.Limit == 0 {
		ans.PageNum = 1
		ans.TotalPage = 1
	} else {
		ans.PageNum = int32(*query.Skip/(*query.Limit)) + 1
		ans.TotalPage = int32(counter / *query.Limit)
	}
	ans.Total = counter
	records := []SceneRelation{}
	cursor, err := sceneRelationCollection.Find(ctx, filter, &options.FindOptions{
		Skip:  query.Skip,
		Limit: query.Limit,
	})
	// Returning error and logging if failed to find message templates
	if err != nil {
		Log.Error(err.Error())
		return ans, CreateError(codes.Internal, 1001)
	}

	// Iterating over result and adding to message templates array
	for cursor.Next(ctx) {
		var scene SceneRelation
		err := cursor.Decode(&scene)

		// Returning error and logging if
		if err != nil {
			Log.Error(err.Error())
			return ans, CreateError(codes.Internal, 1000)
		}
		records = append(records, scene)
	}
	ans.Records = records
	return ans, nil
}
func getSceneRelation(ctx context.Context, sceneId string, appId string) (SceneRelation, error) {
	filter := bson.M{}
	filter[SCENE_RELATION_FIELD_SCENEID] = sceneId
	filter[SCENE_RELATION_FIELD_APPID] = appId
	ans := SceneRelation{}
	res := sceneRelationCollection.FindOne(ctx, filter)
	res.Decode(&ans)
	return ans, nil
}
