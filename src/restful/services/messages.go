package services

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"messenger/configs"
	"messenger/restful/models"
	pb "messenger/services"
	"net/http"
	"net/mail"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	t "text/template"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	uuid "github.com/satori/go.uuid"
	"gitlab.zixel.cn/go/framework"
	"gitlab.zixel.cn/go/framework/database"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc/codes"
	"gopkg.in/gomail.v2"
)

// validation arrays for messages.go
var targetTypes = []string{"Global", "App"}
var emailBodyTypes = []string{"text/html", "text/plain"}
var connectionModes = []string{"WhiteList", "BlackList"}

const (
	GetContactInfoSelector     = 8
	GetApplicationInfoSelector = 30
)

// Message : Model of message
type Message struct {
	Id                    primitive.ObjectID           `json:"_id" bson:"_id"`
	LinkageId             primitive.ObjectID           `json:"linkageId,omitempty" bson:"linkageId,omitempty"`
	AppId                 string                       `json:"appId" bson:"appId"`
	OpenId                string                       `json:"openId" bson:"openId"`
	ClassId               string                       `json:"classId" validate:"required" bson:"classId"`
	SubClassId            string                       `json:"subclassId" bson:"subclassId"`
	MessageTemplateId     string                       `json:"messageTemplateId,omitempty" bson:"messageTemplateId,omitempty"`
	Params                []string                     `json:"params,omitempty" bson:"params,omitempty"`
	Content               string                       `json:"content" bson:"content"`
	Source                string                       `json:"source" validate:"required" bson:"source"`
	Status                string                       `json:"status,omitempty" bson:"status,omitempty"`
	Target                []string                     `json:"target" bson:"target"`
	StatusSync            map[string]string            `json:"statusSync,omitempty" bson:"statusSync,omitempty"`
	CreatedAt             time.Time                    `json:"createdAt" bson:"createdAt"`
	UpdatedAt             time.Time                    `json:"updatedAt" bson:"updatedAt"`
	CustomEmailRequest    models.CustomEmailRequest    `json:"customEmailRequest,omitempty" bson:"customEmailRequest,omitempty"`
	CustomSmsRequest      models.CustomSmsRequest      `json:"customSmsRequest,omitempty" bson:"customSmsRequest,omitempty"`
	CustomFeishuRequest   models.CustomFeishuRequest   `json:"customFeishuRequest,omitempty" bson:"customFeishuRequest,omitempty"`
	CustomDingtalkRequest models.CustomDingtalkRequest `json:"customDingtalkRequest,omitempty" bson:"customDingtalkRequest,omitempty"`
	Actions               []models.ActionModel         `json:"actions,omitempty"`
}

type WebSocketMessage struct {
	Action string `json:"action"`
	Data   string `json:"data"`
}

// ClientConnection : Client Connection contains the client connection information
type ClientConnection struct {
	openId      string
	connections map[string]*Connection
	mu          *sync.Mutex
}

type Connection struct {
	openId    string
	addr      string
	channels  []string
	whiteList []string
	blackList []string
	mode      string
	mu        *sync.Mutex
	conn      *websocket.Conn
	send      chan []byte
}

// Encode : Encode message to json
func (message *Message) Encode() []byte {
	marshal, err := json.Marshal(message)
	if err != nil {
		Log.Println(err)
	}

	return marshal
}

func (message *WebSocketMessage) encode() []byte {
	marshal, err := json.Marshal(message)
	if err != nil {
		Log.Error(err.Error())
	}

	return marshal
}

// SendMessage : Calls the required function to send message
func SendMessage(ctx context.Context, message *Message) error {
	// Initializing the configuration
	var configuration Configuration

	// Returning error and Logging if failed to validate request JSON
	if err := ValidateJSON(message); err != nil {
		return err
	}

	// Returning error and Logging if failed to validate send message request
	if err := validateSendMessage(ctx, message, &configuration); err != nil {
		return err
	}

	// Returning error and Logging if failed to send message
	if err := sendMessage(ctx, message, &configuration); err != nil {
		return err
	}

	// Returning nil if successful
	return nil
}
func SendMessageByScene(ctx context.Context, scene *SceneRelation, args []string) (*models.SendMessageV2Response, error) {
	// Initializing the configuration
	// Returning error and Logging if failed to validate request JSON
	// if err := ValidateJSON(scene); err != nil {
	// 	return nil, err
	// }

	// Returning error and Logging if failed to validate send message request
	ok, sceneRelation, err := validateSendMessageByScene(ctx, *scene)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	message := models.SendMessageV2Request{
		ClassId: sceneRelation.ConfigId,
		// SubClassId:        sceneRelation.ConfigSubId,
		MessageTemplateId: sceneRelation.TemplateId,
		Params:            args,
		Source:            sceneRelation.Source,
		Targets:           sceneRelation.Target,
		CustomEmailRequest: models.CustomEmailRequest{
			Subject:     sceneRelation.CustomEmailRequest.Subject,
			BodyType:    sceneRelation.CustomEmailRequest.BodyType,
			Cc:          sceneRelation.CustomEmailRequest.Cc,
			Bcc:         sceneRelation.CustomEmailRequest.Bcc,
			Attachments: sceneRelation.CustomEmailRequest.Attachments,
		},
		CustomSmsRequest: models.CustomSmsRequest{
			TemplateParams: args,
		},
		CustomFeishuRequestV2: models.CustomFeishuRequestV2{
			MsgType: sceneRelation.CustomFeishuRequest.MsgType,
		},
	}
	// Returning error and Logging if failed to send message
	if res, err := SendMessageV2(ctx, &message); err != nil {
		return nil, err
	} else {
		return res, nil

	}
}

// SendMessageV2 : Calls the required function to send message
func SendMessageV2(ctx context.Context, request *models.SendMessageV2Request) (*models.SendMessageV2Response, error) {
	// Initializing the configuration
	var configuration Configuration

	// Returning error and Logging if failed to validate request JSON
	if err := ValidateJSON(request); err != nil {
		return nil, err
	}

	// Returning error and Logging if failed to validate send request
	message, err := validateSendMessageV2(ctx, request, &configuration)
	if err != nil {
		return nil, err
	}

	// Returning error and Logging if failed to send request
	resp, err := sendMessageV2(ctx, message, &configuration)
	if err != nil {
		return nil, err
	} else {
		return resp, nil
	}
}

// UpdateMessageStatus : Calls the required function to update message status
func UpdateMessageStatus(
	ctx context.Context,
	message *Message,
	status *models.MessageStatus,
	id string) error {
	// Initializing the configuration
	var configuration Configuration

	// Returning error and Logging if failed to find message
	if err := findMessage(ctx, message, &configuration, id); err != nil {
		return err
	}

	// Returning error and Logging if failed to validate request JSON
	if err := ValidateJSON(message); err != nil {
		return err
	}

	// Returning error and Logging if failed to validate update message status request
	if err := validateUpdateMessageStatus(ctx, message, &configuration, status); err != nil {
		return err
	}

	// Returning error and Logging if failed to update message status
	if err := updateMessageStatus(ctx, message, &configuration, status); err != nil {
		return err
	}

	// Return nil if message status is updated successfully
	return nil
}

// HttpUpdateMessageStatus : Calls the required function to update message status
func HttpUpdateMessageStatus(c *gin.Context) error {
	var configuration Configuration
	var message Message

	if err := validateHttpUpdateMessageStatus(c, &message, &configuration); err != nil {
		return err
	}

	if err := httpUpdateMessageStatus(c, &message, &configuration); err != nil {
		return err
	} else {
		return nil
	}
}

// HttpUpdateMessagesStatus : Calls the required function to update message status
func HttpUpdateMessagesStatus(c *gin.Context) error {
	var configurations map[string]Configuration
	var messages []Message

	if err := validateHttpUpdateMessagesStatus(c, &messages, &configurations); err != nil {
		return err
	}

	if err := httpUpdateMessagesStatus(c, &messages); err != nil {
		return err
	}

	return nil
}

// DeleteMessage : Calls the required function to delete message
func DeleteMessage(
	ctx context.Context,
	message *Message,
	status *models.MessageStatus,
	id string) error {
	// Initializing the configuration
	var configuration Configuration

	// Returning error and Logging if failed to find message
	if err := findMessage(ctx, message, &configuration, id); err != nil {
		return err
	}

	// Returning error and Logging if failed to validate request JSON
	if err := ValidateJSON(message); err != nil {
		return err
	}

	// Returning error and Logging if failed to validate delete message request
	if err := validateDeleteMessage(ctx, message, &configuration, status); err != nil {
		return err
	}

	// Returning error and Logging if failed to delete message
	if err := deleteMessage(ctx, message, &configuration, status); err != nil {
		return err
	}

	// Return nil if message is deleted successfully
	return nil
}

// HttpDeleteMessage : Calls the required function to delete message status
func HttpDeleteMessage(c *gin.Context) error {
	var message Message
	var configuration Configuration

	if err := validateHttpDeleteMessage(c, &message, &configuration); err != nil {
		return err
	}

	if err := httpDeleteMessage(c, &message, &configuration); err != nil {
		return err
	} else {
		return nil
	}
}

// QueryMessages : Calls the required function to query messages
func QueryMessages(ctx context.Context, query *models.QueryMessageParams) (*pb.QueryMessagesResponse, error) {
	err := validateQueryMessages(ctx, query)
	if err != nil {
		return nil, err
	}

	// Returning error and Logging if failed to query queryMessageResponse
	queryMessageResponse, err := queryMessages(ctx, query)
	if err != nil {
		return nil, err
	}

	// Converting struct queryMessageResponse to protobuf queryMessageResponse
	response := make([]*pb.Message, len(queryMessageResponse.Results.([]Message)))
	for i, message := range queryMessageResponse.Results.([]Message) {
		responseActions := make([]*pb.MessageAction, len(message.Actions))
		for i, action := range message.Actions {
			responseActions[i] = &pb.MessageAction{
				Name: action.Name,
				Tips: action.Tips,
				Action: &pb.MessageActionTrigger{
					Type:   action.Action.Type,
					Url:    action.Action.URL,
					Method: action.Action.Method,
					Body:   action.Action.Body,
					Query:  action.Action.Query,
				},
				NextAction: action.NextAction,
			}
		}
		response[i] = &pb.Message{
			Id:                message.Id.Hex(),
			LinkageId:         message.LinkageId.Hex(),
			AppId:             message.AppId,
			OpenId:            message.OpenId,
			ClassId:           message.ClassId,
			SubclassId:        message.SubClassId,
			MessageTemplateId: message.MessageTemplateId,
			Content:           message.Content,
			Source:            message.Source,
			Target:            message.Target,
			Status:            message.Status,
			StatusSync:        ConvertMapToStruct(message.StatusSync),
			CreatedAt:         message.CreatedAt.Format(time.RFC3339),
			UpdatedAt:         message.UpdatedAt.Format(time.RFC3339),
			Actions:           responseActions,
		}
	}

	// Returning queryMessageResponse and nil if successful
	return &pb.QueryMessagesResponse{
		Page:    query.Page,
		Limit:   query.Limit,
		Total:   queryMessageResponse.Total,
		Results: response,
	}, nil
}

// HttpQueryMessages : Calls the required function to query messages
func HttpQueryMessages(c *gin.Context) (models.QueryMessageResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// get page and convert to int64
	page, err := strconv.ParseInt(c.Query("page"), 10, 64)
	if err != nil {
		Log.Error(err.Error())
		return models.QueryMessageResponse{}, CreateError(http.StatusBadRequest, 100004)
	}

	limit, err := strconv.ParseInt(c.Query("limit"), 10, 64)
	if err != nil {
		Log.Error(err.Error())
		return models.QueryMessageResponse{}, CreateError(http.StatusBadRequest, 100004)
	}

	// Converting query from proto to struct
	query := models.QueryMessageParams{
		AppId:       c.Query("appId"),
		OpenId:      c.Query("openId"),
		Source:      c.Query("source"),
		ClassId:     c.Query("classId"),
		ClassIds:    c.Query("classIds"),
		SubclassId:  c.Query("subclassId"),
		SubclassIds: c.Query("subclassIds"),
		Status:      c.Query("status"),
		Page:        page,
		Limit:       limit,
		Order:       c.Query("order"),
		Sort:        c.Query("sort"),
	}

	err = validateQueryMessages(ctx, &query)
	if err != nil {
		return models.QueryMessageResponse{}, err
	}

	// Returning error and Logging if failed to query messages
	messages, err := queryMessages(ctx, &query)
	if err != nil {
		return models.QueryMessageResponse{}, err
	} else {

		return messages, nil
	}
}

// GetMessages : Calls the required function to get cached messages for a user
func GetMessages(ctx context.Context, openId string) (*pb.QueryMessagesResponse, error) {
	// Returning error and Logging if failed to get user cached messages
	messages, err := getMessages(ctx, openId)
	if err != nil {
		return nil, err
	}

	// Converting struct messages to protobuf messages
	response := make([]*pb.Message, len(messages))
	for i, message := range messages {
		responseActions := make([]*pb.MessageAction, len(message.Actions))
		for i, action := range message.Actions {
			responseActions[i] = &pb.MessageAction{
				Name: action.Name,
				Tips: action.Tips,
				Action: &pb.MessageActionTrigger{
					Type:   action.Action.Type,
					Url:    action.Action.URL,
					Method: action.Action.Method,
					Body:   action.Action.Body,
					Query:  action.Action.Query,
				},
				NextAction: action.NextAction,
			}
		}
		response[i] = &pb.Message{
			Id:                message.Id.Hex(),
			LinkageId:         message.LinkageId.Hex(),
			AppId:             message.AppId,
			OpenId:            message.OpenId,
			ClassId:           message.ClassId,
			SubclassId:        message.SubClassId,
			MessageTemplateId: message.MessageTemplateId,
			Content:           message.Content,
			Source:            message.Source,
			Target:            message.Target,
			Status:            message.Status,
			StatusSync:        ConvertMapToStruct(message.StatusSync),
			CreatedAt:         message.CreatedAt.Format(time.RFC3339),
			UpdatedAt:         message.UpdatedAt.Format(time.RFC3339),
			Actions:           responseActions,
		}
	}

	// Returning messages and nil if successful
	return &pb.QueryMessagesResponse{
		Page:    1,
		Limit:   int64(len(messages)),
		Total:   int64(len(messages)),
		Results: response}, nil
}

// HttpGetMessages : Calls the required function to get cached messages for a user in http request
func HttpGetMessages(c *gin.Context) ([]Message, error) {
	if err := validateHttpGetMessages(c); err != nil {
		return nil, err
	}

	if messages, err := httpGetMessages(c); err != nil {
		return nil, err
	} else {
		return messages, nil
	}
}

// CreateWebSocketConnection : Create WebSocket Connection calls the required functions to create a websocket connection
func CreateWebSocketConnection(c *gin.Context) error {
	// Returning error and Logging if failed to validate websocket connection
	if err := validateWebSocketConnection(c); err != nil {
		return err
	}

	// Returning error and Logging if failed to create websocket connection
	if err := createWebSocketConnection(c); err != nil {
		return err
	}

	// Returning nil if successful
	return nil
}

// validateSendMessage : Validates send message request
func validateSendMessage(ctx context.Context, message *Message, configuration *Configuration) error {
	var class Configuration

	// Returning error and Logging if failed to find message class
	if err := getConfigurationByClassId(ctx, message.ClassId, &class); err != nil {
		return err
	}

	// Validating message subClass
	if message.SubClassId == "" || message.SubClassId == message.ClassId {
		message.SubClassId = message.ClassId
		*configuration = class
	} else {
		// Returning failed to find message subclass
		if err := getConfigurationByClassId(ctx, message.SubClassId, configuration); err != nil {
			return err
		}

		// Returning error and Logging if failed to validate message parent class with subclass
		if err := validateMessageParentClassId(class.ClassId, configuration.ParentId); err != nil {
			return err
		}
	}

	// Returning error and Logging if failed to validate message template
	if message.MessageTemplateId != "" {
		// Initializing message template model
		var template Template

		// Returning error and Logging if failed to find message template
		if err := getTemplateByTemplateId(ctx, message.MessageTemplateId, &template); err != nil {
			return err
		}
		if len(template.Params) != len(message.Params) {
			Log.Error("Invalid message params")
			return CreateError(codes.InvalidArgument, 100321)
		} else {
			data := make(map[string]interface{}, len(message.Params))
			for i := 0; i < len(message.Params); i++ {
				data[template.Params[i]] = message.Params[i]
			}
			temp, err := t.Must(t.New("todos").Parse(template.Content)).Parse(template.Content)
			if err != nil {
				Log.Error("Failed to validate template")
				return CreateError(codes.InvalidArgument, 100323)
			}
			// Create an io.Writer to write to the memory buffer
			var w io.Writer
			// Create a writer to store the output of the template
			w = new(bytes.Buffer)

			// Execute the template
			err = temp.Execute(w, data)
			if err != nil {
				Log.Error("Failed to execute template")
				return CreateError(codes.InvalidArgument, 100322)
			}

			// Setting message content to message template content
			message.Content = w.(*bytes.Buffer).String()
		}
	}

	// Returning error and Logging if failed to validate message target
	if err := validateMessageTarget(message.Target); err != nil {
		return nil
	}

	// Initializing message status sync
	message.StatusSync = map[string]string{}

	// Returning error and Logging if failed to validate message status
	if err := setDefaultMessageStatus(message, configuration); err != nil {
		return nil
	}

	// Returning error and Logging if failed to validate email body type
	if configuration.Mode == "email" {
		if message.CustomEmailRequest.BodyType != "" {
			if !contains(emailBodyTypes, message.CustomEmailRequest.BodyType) {
				Log.Error("Invalid email body type " + message.CustomEmailRequest.BodyType)
				return CreateError(codes.InvalidArgument, 100317)
			}
		}
	} else if configuration.Mode == "listening" {
		val, err := database.RedisGetStringCtx(ctx, redisUserListKey+message.SubClassId)
		if err != nil {
			Log.Error(err.Error())
			return CreateError(codes.Internal, 100329)
		}
		var users map[string]bool

		err = json.Unmarshal([]byte(*val), &users)
		if err != nil {
			Log.Error(err.Error())
			return CreateError(codes.Internal, 1000)
		} else {
			message.Target = []string{}
			for openId := range users {
				message.Target = append(message.Target, openId)
			}
		}
	}

	// Setting message id to new object id
	message.Id = primitive.NewObjectID()
	message.AppId = configuration.AppId
	message.CreatedAt = time.Now()
	message.UpdatedAt = time.Now()
	message.Actions = configuration.Actions

	return nil
}
func validateSendMessageByScene(ctx context.Context, scene SceneRelation) (bool, *SceneRelation, error) {
	ans, err := GetSceneRelation(ctx, scene.SceneId, scene.AppId)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return false, nil, nil
		} else {
			return false, nil, err
		}
	}

	return true, &ans, nil
}

// validateSendMessageV2 : Validates send message request
func validateSendMessageV2(ctx context.Context, request *models.SendMessageV2Request, configuration *Configuration) (*Message, error) {
	// Returning error and Logging if failed to find message configuration
	if err := getConfigurationByClassId(ctx, request.ClassId, configuration); err != nil {
		return nil, err
	}

	if configuration.ParentId != "0" {
		request.SubClassId = configuration.ClassId
		request.ClassId = configuration.ParentId
	} else {
		request.SubClassId = configuration.ClassId
	}

	if err := validateMessageTargetV2(request.Targets, configuration.Mode); err != nil {
		return nil, err
	}

	// Returning error and Logging if failed to validate message template
	if request.MessageTemplateId != "" {
		// Initializing message template model
		var template Template

		// Returning error and Logging if failed to find message template
		if err := getTemplateByTemplateId(ctx, request.MessageTemplateId, &template); err != nil {
			return nil, err
		}
		if len(template.Params) != len(request.Params) {
			Log.Error("Invalid message params")
			return nil, CreateError(codes.InvalidArgument, 100321)
		} else {
			data := make(map[string]interface{}, len(request.Params))
			for i := 0; i < len(request.Params); i++ {
				data[template.Params[i]] = request.Params[i]
			}
			temp, err := t.Must(t.New("todos").Parse(template.Content)).Parse(template.Content)
			if err != nil {
				Log.Error("Failed to validate template")
				return nil, CreateError(codes.InvalidArgument, 100323)
			}
			// Create an io.Writer to write to the memory buffer
			var w io.Writer
			// Create a writer to store the output of the template
			w = new(bytes.Buffer)

			// Execute the template
			err = temp.Execute(w, data)
			if err != nil {
				Log.Error("Failed to execute template")
				return nil, CreateError(codes.InvalidArgument, 100322)
			}

			// Setting message content to message template content
			request.Content = w.(*bytes.Buffer).String()
		}
	}

	// Returning error and Logging if failed to validate email body type
	if configuration.Mode == "email" {
		if request.CustomEmailRequest.BodyType != "" {
			if !contains(emailBodyTypes, request.CustomEmailRequest.BodyType) {
				Log.Error("Invalid email body type " + request.CustomEmailRequest.BodyType)
				return nil, CreateError(codes.InvalidArgument, 100317)
			}
		}
	} else if configuration.Mode == "listening" {
		val, err := database.RedisGetStringCtx(ctx, redisUserListKey+request.SubClassId)
		if err != nil {
			Log.Error(err.Error())
			return nil, CreateError(codes.Internal, 100329)
		}
		var users map[string]bool

		err = json.Unmarshal([]byte(*val), &users)
		if err != nil {
			Log.Error(err.Error())
			return nil, CreateError(codes.Internal, 1000)
		} else {
			request.Targets = []string{}
			for openId := range users {
				request.Targets = append(request.Targets, openId)
			}
		}
	}

	message := newMessage(request, configuration)
	return message, nil
}

// validateUpdateMessageStatus : Validates update message status request
func validateUpdateMessageStatus(_ context.Context, message *Message, class *Configuration, status *models.MessageStatus) error {
	// Returning error and Logging if message status is Deleted
	if err := validateMessageStatusToBeUpdatedNotDeleted(status.Status); err != nil {
		return err
	}

	// Returning error and Logging if message status does not exist in states of message class
	if err := validateMessageStatusExistInMessageClassStates(status.Status, class.States); err != nil {
		return err
	}

	// Validating message based on message types
	if class.Type == "exclusive" {
		// Returning error and Logging if message status to be updated is same as current status
		if err := validateMessageStatusToBeUpdatedEqualToCurrentStatus(status.Status, message.Status); err != nil {
			return err
		}

		// Returning error and Logging if message status already set to Deleted
		if err := validateMessageStatusIsDeleted(message.Status); err != nil {
			return err
		}

		// Returning error and Logging if openid does not have permissions to change message status
		if err := validateMessagePermissionToUpdateMessageStatusExclusiveMessage(status.OpenId, message.Target); err != nil {
			return err
		}

		// Setting message status
		message.Status = status.Status
	} else if class.Type == "broadcast" {
		// Returning error and Logging if message status is not validated of message type broadcast
		if err := validateMessageStatusForBroadcastMessage(message, status.Status, status.OpenId); err != nil {
			return err
		}

	} else {
		// Returning error and Logging if message status to be updated is same as current status
		if err := validateMessageStatusToBeUpdatedEqualToCurrentStatus(status.Status, message.Status); err != nil {
			return err
		}

		// Returning error and Logging if message status already set to Deleted
		if err := validateMessageStatusIsDeleted(message.Status); err != nil {
			return err
		}

		// Setting message status
		message.Status = status.Status
	}

	// Setting updatedAt
	message.UpdatedAt = time.Now()

	// Returning false if message is validated successfully
	return nil
}

// validateHttpUpdateMessageStatus : Validates http update message status request
func validateHttpUpdateMessageStatus(c *gin.Context, message *Message, configuration *Configuration) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	openId := c.GetHeader("Zixel-Open-Id")
	if openId == "" {
		Log.Error("OpenId is empty")
		return CreateError(http.StatusBadRequest, 100312)
	}

	status := c.Query("status")

	if status == "" {
		Log.Error("Status is empty")
		return CreateError(http.StatusBadRequest, 100330)
	}

	// Returning error and Logging if failed to find message
	if err := findMessage(ctx, message, configuration, c.Param("messageId")); err != nil {
		return err
	}

	// Returning error and Logging if message status is Deleted
	if err := validateMessageStatusToBeUpdatedNotDeleted(status); err != nil {
		return err
	}

	// Returning error and Logging if message status does not exist in states of message class
	if err := validateMessageStatusExistInMessageClassStates(status, configuration.States); err != nil {
		return err
	}
	// Validating message based on message types
	if configuration.Type == "exclusive" {
		// Returning error and Logging if message status to be updated is same as current status
		if err := validateMessageStatusToBeUpdatedEqualToCurrentStatus(status, message.Status); err != nil {
			return err
		}

		// Returning error and Logging if message status already set to Deleted
		if err := validateMessageStatusIsDeleted(message.Status); err != nil {
			return err
		}

		// Returning error and Logging if openid does not have permissions to change message status
		if err := validateMessagePermissionToUpdateMessageStatusExclusiveMessage(openId, message.Target); err != nil {
			return err
		}

		// Setting message status
		message.Status = status
	} else if configuration.Type == "broadcast" {
		// Returning error and Logging if message status is not validated of message type broadcast
		if err := validateMessageStatusForBroadcastMessage(message, status, openId); err != nil {
			return err
		}

	} else {
		// Returning error and Logging if message status to be updated is same as current status
		if err := validateMessageStatusToBeUpdatedEqualToCurrentStatus(status, message.Status); err != nil {
			return err
		}

		// Returning error and Logging if message status already set to Deleted
		if err := validateMessageStatusIsDeleted(message.Status); err != nil {
			return err
		}

		// Setting message status
		message.Status = status
	}
	return nil
}

// validateHttpUpdateMessageStatuses : Validates http update message statuses request
func validateHttpUpdateMessagesStatus(c *gin.Context, messages *[]Message, configurations *map[string]Configuration) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	openId := c.GetHeader("Zixel-Open-Id")
	if openId == "" {
		Log.Error("OpenId is empty")
		return CreateError(http.StatusBadRequest, 100312)
	}

	status := c.Query("status")
	if status == "" {
		Log.Error("Status is empty")
		return CreateError(http.StatusBadRequest, 100330)
	}

	queryClassIds := c.Query("classIds")
	querySubclassIds := c.Query("subclassIds")
	classIds := strings.Split(queryClassIds, ",")
	subclassIds := strings.Split(querySubclassIds, ",")
	var filterType string
	var filterIds []string
	if queryClassIds != "" && querySubclassIds != "" {
		Log.Error("Either classIds or subClassIds is required in params.")
		return CreateError(http.StatusBadRequest, 100331)
	} else {
		if queryClassIds != "" {
			filterType = "classId"
			filterIds = classIds
		} else if querySubclassIds != "" {
			filterType = "subclassId"
			filterIds = subclassIds
		} else {
			Log.Error("Either classIds or subClassIds is required in params.")
			return CreateError(http.StatusBadRequest, 100331)
		}
	}
	if err := queryConfigurationByIds(ctx, filterIds, configurations); err != nil {
		return err
	}

	msgs, err := queryMessagesByFilter(ctx, filterType, openId, filterIds, status)
	if err != nil {
		return err
	}

	cfs := *configurations
	for _, message := range msgs {
		configuration := cfs[message.SubClassId]
		// Returning error and Logging if message status is Deleted
		if err := validateMessageStatusToBeUpdatedNotDeleted(status); err != nil {
			continue
		}

		// Returning error and Logging if message status does not exist in states of message class
		if err := validateMessageStatusExistInMessageClassStates(status, configuration.States); err != nil {
			continue
		}
		// Validating message based on message types
		if configuration.Type == "exclusive" {
			// Returning error and Logging if message status to be updated is same as current status
			if err := validateMessageStatusToBeUpdatedEqualToCurrentStatus(status, message.Status); err != nil {
				continue
			}

			// Returning error and Logging if message status already set to Deleted
			if err := validateMessageStatusIsDeleted(message.Status); err != nil {
				continue
			}

			// Returning error and Logging if openid does not have permissions to change message status
			if err := validateMessagePermissionToUpdateMessageStatusExclusiveMessage(openId, message.Target); err != nil {
				continue
			}

			// Setting message status
			message.Status = status
		} else if configuration.Type == "broadcast" {
			// Returning error and Logging if message status is not validated of message type broadcast
			if err := validateMessageStatusForBroadcastMessage(&message, status, openId); err != nil {
				continue
			}

		} else {
			// Returning error and Logging if message status to be updated is same as current status
			if err := validateMessageStatusToBeUpdatedEqualToCurrentStatus(status, message.Status); err != nil {
				continue
			}

			// Returning error and Logging if message status already set to Deleted
			if err := validateMessageStatusIsDeleted(message.Status); err != nil {
				continue
			}

			// Setting message status
			message.Status = status
		}
		message.UpdatedAt = time.Now()
		*messages = append(*messages, message)
	}
	return nil
}

// validateDeleteMessage : Validates delete message request
func validateDeleteMessage(_ context.Context, message *Message, class *Configuration, status *models.MessageStatus) error {
	// Setting message status to Deleted
	status.Status = "deleted"

	// Returning error and Logging if message status does not exist in states of message class
	if err := validateMessageStatusExistInMessageClassStates(status.Status, class.States); err != nil {
		return err
	}

	// Setting updatedAt
	message.UpdatedAt = time.Now()

	// Validating message based on message types
	if class.Type == "exclusive" {
		// Returning error and Logging if openid does not have permissions to change message status
		if err := validateMessagePermissionToUpdateMessageStatusExclusiveMessage(status.OpenId, message.Target); err != nil {
			return err
		}

		// Setting message status
		message.Status = status.Status
	} else if class.Type == "broadcast" {
		// Setting message status
		message.StatusSync[status.OpenId] = status.Status
	} else {
		// Setting message status
		message.Status = status.Status
	}

	// Returning false if message is validated successfully
	return nil
}

// validateHttpDeleteMessage : Validates http delete messages request
func validateHttpDeleteMessage(c *gin.Context, message *Message, configuration *Configuration) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	status := "deleted"
	openId := c.GetHeader("Zixel-Open-Id")
	if openId == "" {
		Log.Error("OpenId is empty")
		return CreateError(http.StatusBadRequest, 100312)
	}

	// Returning error and Logging if failed to find message
	if err := findMessage(ctx, message, configuration, c.Param("messageId")); err != nil {
		return err
	}

	// Returning error and Logging if message status does not exist in states of message configuration
	if err := validateMessageStatusExistInMessageClassStates(status, configuration.States); err != nil {
		return err
	}

	// Setting updatedAt
	message.UpdatedAt = time.Now()

	// Validating message based on message types
	if configuration.Type == "exclusive" {
		// Returning error and Logging if openid does not have permissions to change message status
		if err := validateMessagePermissionToUpdateMessageStatusExclusiveMessage(openId, message.Target); err != nil {
			return err
		}

		// Setting message status
		message.Status = status
	} else if configuration.Type == "broadcast" {
		// Setting message status
		message.StatusSync[openId] = status
	} else {
		// Setting message status
		message.Status = status
	}

	return nil
}

// validateQueryMessages : Validates query message request
func validateQueryMessages(_ context.Context, params *models.QueryMessageParams) error {
	if params.Page == 0 {
		params.Page = 1
	}

	if params.Limit == 0 {
		params.Limit = 10
	}

	return nil
}

// validateHttpGetMessages : Validates http get messages request
func validateHttpGetMessages(c *gin.Context) error {
	openId := c.GetHeader("Zixel-Open-Id")
	if openId == "" {
		Log.Error("OpenId is empty")
		return CreateError(http.StatusBadRequest, 100312)
	}

	return nil
}

// validateWebSocketConnection : validates the request for websocket connection
func validateWebSocketConnection(c *gin.Context) error {
	ServerInstance.mu.Lock()
	defer ServerInstance.mu.Unlock()
	// Returning error and Logging if failed to parse request
	if err := c.Request.ParseForm(); err != nil {
		return err
	}

	// Returning error and Logging if open id is empty
	openId := c.GetHeader("Zixel-Open-Id")
	if openId == "" {
		Log.Info("OpenId Header is empty")
		openId = c.Query("openId")
		if openId == "" {
			Log.Error("OpenId is empty")
			return CreateError(http.StatusBadRequest, 1030207)
		}
	}

	// Getting client from WebSocketServer and setting client connection
	if _, ok := ServerInstance.clients[openId]; !ok {
		client := ClientConnection{
			openId:      openId,
			connections: make(map[string]*Connection),
			mu:          &sync.Mutex{},
		}

		Log.Debug("Client added to WebSocketServer")
		// Adding client to WebSocketServer
		ServerInstance.clients[client.openId] = &client
	}

	// Returning nil if successful
	return nil
}

// sendMessage : Sends message to target
func sendMessage(ctx context.Context, message *Message, configuration *Configuration) error {
	// Sending Message based on message mode in message configuration
	if configuration.Mode == "pull" {
		// Checking if configuration type is linking
		if configuration.Type == "linkage" {
			linkageMessage := *message
			linkageMessage.Id = primitive.NewObjectID()
			linkageMessage.LinkageId = message.Id
			linkageMessage.OpenId = message.Target[0]
			message.LinkageId = linkageMessage.Id

			// Returning error and Logging if failed to add message to user cached messages
			if err := addMessageToUserCachedMessagesRedis(ctx, message.Target[0], &linkageMessage); err != nil {
				return err
			}

			// Returning error and Logging if failed to insert message to database
			if err := insertOneLinkingMessage(ctx, &linkageMessage, configuration); err != nil {
				return err
			}
		} else {
			// Iterating through all openIds in message target and adding message to cache
			for _, receiver := range message.Target {
				// Returning error and Logging if failed to add message to user cached messages
				if err := addMessageToUserCachedMessagesRedis(ctx, receiver, message); err != nil {
					return err
				}
			}
		}
	} else if configuration.Mode == "push" {
		// Checking if configuration type is linking
		if configuration.Type == "linkage" {
			linkageMessage := *message
			linkageMessage.Id = primitive.NewObjectID()
			linkageMessage.LinkageId = message.Id
			linkageMessage.OpenId = message.Target[0]
			message.LinkageId = linkageMessage.Id

			redirectMessageToSockets(&linkageMessage, configuration)
			// Returning error and Logging if failed to insert message to database
			if err := insertOneLinkingMessage(ctx, &linkageMessage, configuration); err != nil {
				return err
			}
		} else {
			redirectMessageToSockets(message, configuration)
		}
	} else if configuration.Mode == "third-party" {
		if err := sendThirdPartyMessage(ctx, message, configuration); err != nil {
			return err
		}
	} else if configuration.Mode == "email" {
		if err := sendEmail(ctx, message, configuration); err != nil {
			return err
		}
	} else if configuration.Mode == "sms" {
		if err := sendSMS(ctx, message, configuration); err != nil {
			return err
		}
	} else if configuration.Mode == "feishu" {
		if err := sendFeishuMessage(ctx, message, configuration); err != nil {
			return err
		}
	} else if configuration.Mode == "dingtalk" {
		if err := sendDingTalkMessage(ctx, message, configuration); err != nil {
			return err
		}
	} else if configuration.Mode == "listening" {
		if err := sendListeningMessage(ctx, message, configuration); err != nil {
			return err
		}
	} else {
		return CreateError(http.StatusBadRequest, 100105)
	}

	// Returning error and Logging if failed to insert message
	if err := insertOneMessage(ctx, message, configuration); err != nil {
		return err
	}

	// Returning nil if successful
	return nil
}

// sendMessageV2 : Sends message to target
func sendMessageV2(ctx context.Context, message *Message, configuration *Configuration) (*models.SendMessageV2Response, error) {
	var failedTargets []string
	// Sending Message based on message mode in message configuration
	if configuration.Mode == "pull" {
		// Checking if configuration type is linking
		if configuration.Type == "linkage" {
			linkageMessage := *message
			linkageMessage.Id = primitive.NewObjectID()
			linkageMessage.LinkageId = message.Id
			linkageMessage.OpenId = message.Target[0]
			message.LinkageId = linkageMessage.Id

			// Returning error and Logging if failed to add message to user cached messages
			if err := addMessageToUserCachedMessagesRedis(ctx, message.Target[0], &linkageMessage); err != nil {
				return nil, err
			}

			// Returning error and Logging if failed to insert message to database
			if err := insertOneLinkingMessage(ctx, &linkageMessage, configuration); err != nil {
				return nil, err
			}
		} else {
			// Iterating through all openIds in message target and adding message to cache
			for _, receiver := range message.Target {
				// Returning error and Logging if failed to add message to user cached messages
				if err := addMessageToUserCachedMessagesRedis(ctx, receiver, message); err != nil {
					return nil, err
				}
			}
		}
	} else if configuration.Mode == "push" {
		// Checking if configuration type is linking
		if configuration.Type == "linkage" {
			linkageMessage := *message
			linkageMessage.Id = primitive.NewObjectID()
			linkageMessage.LinkageId = message.Id
			linkageMessage.OpenId = message.Target[0]
			message.LinkageId = linkageMessage.Id

			redirectMessageToSockets(&linkageMessage, configuration)
			// Returning error and Logging if failed to insert message to database
			if err := insertOneLinkingMessage(ctx, &linkageMessage, configuration); err != nil {
				return nil, err
			}
		} else {
			redirectMessageToSockets(message, configuration)
		}
	} else if configuration.Mode == "third-party" {
		if targets, err := sendThirdPartyMessageV2(ctx, message, configuration); err != nil {
			return nil, err
		} else {
			failedTargets = targets
		}
	} else if configuration.Mode == "email" {
		if targets, err := sendEmailV2(ctx, message, configuration); err != nil {
			return nil, err
		} else {
			failedTargets = targets
		}
	} else if configuration.Mode == "sms" {
		if targets, err := sendSmsV2(ctx, message, configuration); err != nil {
			return nil, err
		} else {
			failedTargets = targets
		}
	} else if configuration.Mode == "feishu" {
		if targets, err := sendFeishuMessageV2(ctx, message, configuration); err != nil {
			return nil, err
		} else {
			failedTargets = targets
		}
	} else if configuration.Mode == "dingtalk" {
		if targets, err := sendDingTalkMessageV2(ctx, message, configuration); err != nil {
			return nil, err
		} else {
			failedTargets = targets
		}
	} else if configuration.Mode == "listening" {
		if err := sendListeningMessage(ctx, message, configuration); err != nil {
			return nil, err
		}
	} else {
		return nil, CreateError(http.StatusBadRequest, 100105)
	}

	// Returning error and Logging if failed to insert message
	if err := insertOneMessage(ctx, message, configuration); err != nil {
		return nil, err
	}

	// Returning nil if successful
	return &models.SendMessageV2Response{
		MessageId:     message.Id.Hex(),
		FailedTargets: failedTargets,
	}, nil
}

// updateMessageStatus : Updates message status
func updateMessageStatus(ctx context.Context, message *Message, configuration *Configuration, status *models.MessageStatus) error {
	// Updating message status based on message mode in message configuration
	if configuration.Mode == "pull" {
		if configuration.Type == "linkage" {
			// Initializing linkage message
			var linkingMessage Message

			// Returning error and Logging if failed to get linkage message
			if err := findLinkingMessage(ctx, message.LinkageId, &linkingMessage, configuration); err != nil {
				return err
			}

			// Updating linkage message status
			linkingMessage.Status = status.Status
			linkingMessage.UpdatedAt = message.UpdatedAt

			// Returning error and Logging if failed to update linkage message
			if err := addMessageToRedis(ctx, &linkingMessage); err != nil {
				return err
			}

			// Update user cached messages.go with linkage message
			if err := updateMessageInUserCachedMessagesRedis(ctx, message.OpenId, linkingMessage); err != nil {
				return err
			}
		} else {
			// Iterating through all openid in message target and updating message status in cache
			for _, receiver := range message.Target {
				r := receiver
				// Checking if the openid updating status is not updated in cache
				if r != status.OpenId {
					if err := updateMessageInUserCachedMessagesRedis(ctx, r, *message); err != nil {
						return err
					}
				}
			}
		}
	} else {
		// Checking if configuration type is linking
		if configuration.Type == "linkage" {
			// Initializing linkage message
			var linkingMessage Message

			// Returning error and Logging if failed to get linkage message
			if err := findLinkingMessage(ctx, message.LinkageId, &linkingMessage, configuration); err != nil {
				return err
			}

			// Updating linkage message status
			linkingMessage.Status = status.Status
			linkingMessage.UpdatedAt = message.UpdatedAt

			// Returning error and Logging if failed to update linkage message
			if err := addMessageToRedis(ctx, &linkingMessage); err != nil {
				return err
			}

			redirectMessageToSockets(&linkingMessage, configuration)

			// Storing linkage message
			if err := insertOneLinkingMessage(ctx, &linkingMessage, configuration); err != nil {
				return err
			}

		} else {
			redirectMessageToSockets(message, configuration)

			if message.OpenId != status.OpenId {
				sendMessageToUserSocket(message.OpenId, message, configuration, true)
			}
		}
	}

	// Returning error and Logging if failed to update message
	if err := updateOneMessage(ctx, message, configuration); err != nil {
		return nil
	}

	// Returning nil if successful
	return nil
}

// httpUpdateMessageStatus : Updates message status using http
func httpUpdateMessageStatus(c *gin.Context, message *Message, configuration *Configuration) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	status := c.Query("status")
	openId := c.GetHeader("Zixel-Open-Id")

	// Updating message status based on message mode in message configuration
	if configuration.Mode == "pull" {
		if configuration.Type == "linkage" {
			// Initializing linkage message
			var linkingMessage Message

			// Returning error and Logging if failed to get linkage message
			if err := findLinkingMessage(ctx, message.LinkageId, &linkingMessage, configuration); err != nil {
				return err
			}

			// Updating linkage message status
			linkingMessage.Status = status
			linkingMessage.UpdatedAt = message.UpdatedAt

			// Returning error and Logging if failed to update linkage message
			if err := addMessageToRedis(ctx, &linkingMessage); err != nil {
				return err
			}

			// Update user cached messages.go with linkage message
			if err := updateMessageInUserCachedMessagesRedis(ctx, message.OpenId, linkingMessage); err != nil {
				return err
			}
		} else {
			// Iterating through all openid in message target and updating message status in cache
			for _, receiver := range message.Target {
				r := receiver
				// Checking if the openid updating status is not updated in cache
				if r != openId {
					if err := updateMessageInUserCachedMessagesRedis(ctx, r, *message); err != nil {
						return err
					}
				}
			}
		}
	} else {
		// Checking if configuration type is linking
		if configuration.Type == "linkage" {
			// Initializing linkage message
			var linkingMessage Message

			// Returning error and Logging if failed to get linkage message
			if err := findLinkingMessage(ctx, message.LinkageId, &linkingMessage, configuration); err != nil {
				return err
			}

			// Updating linkage message status
			linkingMessage.Status = status
			linkingMessage.UpdatedAt = message.UpdatedAt

			// Returning error and Logging if failed to update linkage message
			if err := addMessageToRedis(ctx, &linkingMessage); err != nil {
				return err
			}

			redirectMessageToSockets(&linkingMessage, configuration)

			// Storing linkage message
			if err := insertOneLinkingMessage(ctx, &linkingMessage, configuration); err != nil {
				return err
			}

		} else {
			redirectMessageToSockets(message, configuration)

			if message.OpenId != openId {
				sendMessageToUserSocket(message.OpenId, message, configuration, true)
			}
		}
	}

	// Returning error and Logging if failed to update message
	if err := updateOneMessage(ctx, message, configuration); err != nil {
		return nil
	}

	return nil
}

// httpUpdateMessagesStatus : Updates messages status using http
func httpUpdateMessagesStatus(c *gin.Context, messages *[]Message) error {
	writeModels := make([]mongo.WriteModel, len(*messages))

	if len(*messages) > 0 {
		for i, message := range *messages {
			filter := bson.M{"_id": message.Id}
			update := bson.M{"$set": message}

			model := mongo.NewUpdateOneModel().SetFilter(filter).SetUpdate(update)
			writeModels[i] = model
		}

		opts := options.BulkWrite().SetOrdered(false)
		res, err := messageCollection.BulkWrite(c, writeModels, opts)

		if err != nil {
			return err
		}

		Log.Info("Modified Messages: ", res.ModifiedCount, " Expected: ", len(*messages))
	}

	return nil
}

// deleteMessage : Deletes message
func deleteMessage(
	ctx context.Context,
	message *Message,
	configuration *Configuration,
	status *models.MessageStatus) error {
	// Deleting message based on message mode in message configuration
	if configuration.Mode == "pull" {
		// Checking if configuration type is linking
		if configuration.Type == "linkage" {
			// Initializing linkage message
			var linkingMessage Message

			// Returning error and Logging if failed to get linkage message
			if err := findLinkingMessage(ctx, message.LinkageId, &linkingMessage, configuration); err != nil {
				return err
			}

			// Updating linkage message status
			linkingMessage.Status = status.Status
			linkingMessage.UpdatedAt = message.UpdatedAt

			// Returning error and Logging if failed to update linkage message
			if err := addMessageToRedis(ctx, &linkingMessage); err != nil {
				return err
			}

			// Update user cached messages with linkage message
			if err := updateMessageInUserCachedMessagesRedis(ctx, message.OpenId, linkingMessage); err != nil {
				return err
			}
		} else {
			// Iterating through all openid in message target and updating message status in cache
			for _, receiver := range message.Target {
				r := receiver
				// Checking if the openid updating status is not updated in cache
				if r != status.OpenId {
					if err := updateMessageInUserCachedMessagesRedis(ctx, r, *message); err != nil {
						return err
					}
				}
			}
		}
	} else {
		// Sending message to application if application connection is available
		redirectMessageToSockets(message, configuration)

		if message.OpenId != status.OpenId {
			sendMessageToUserSocket(message.OpenId, message, configuration, true)
		}
	}

	// Returning error and Logging if failed to update message
	if err := updateOneMessage(ctx, message, configuration); err != nil {
		return err
	}

	// Returning nil if successful
	return nil
}

// httpDeleteMessage : Deletes message via http
func httpDeleteMessage(c *gin.Context, message *Message, configuration *Configuration) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	openId := c.GetHeader("Zixel-Open-Id")
	// Deleting message based on message mode in message configuration
	if configuration.Mode == "pull" {
		// Checking if configuration type is linking
		if configuration.Type == "linkage" {
			// Initializing linkage message
			var linkingMessage Message

			// Returning error and Logging if failed to get linkage message
			if err := findLinkingMessage(ctx, message.LinkageId, &linkingMessage, configuration); err != nil {
				return err
			}

			// Updating linkage message status
			linkingMessage.Status = "deleted"
			linkingMessage.UpdatedAt = message.UpdatedAt

			// Returning error and Logging if failed to update linkage message
			if err := addMessageToRedis(ctx, &linkingMessage); err != nil {
				return err
			}

			// Update user cached messages with linkage message
			if err := updateMessageInUserCachedMessagesRedis(ctx, message.OpenId, linkingMessage); err != nil {
				return err
			}
		} else {
			// Iterating through all openid in message target and updating message status in cache
			for _, receiver := range message.Target {
				r := receiver
				// Checking if the openid updating status is not updated in cache
				if r != openId {
					if err := updateMessageInUserCachedMessagesRedis(ctx, r, *message); err != nil {
						return err
					}
				}
			}
		}
	} else {
		// Sending message to application if application connection is available
		redirectMessageToSockets(message, configuration)

		if message.OpenId != openId {
			sendMessageToUserSocket(message.OpenId, message, configuration, true)
		}
	}

	// Returning error and Logging if failed to update message
	if err := updateOneMessage(ctx, message, configuration); err != nil {
		return err
	}
	return nil
}

// queryMessages : Queries messages
func queryMessages(ctx context.Context, params *models.QueryMessageParams) (models.QueryMessageResponse, error) {
	// Initializing messages
	var messages []Message

	// Initializing query parameters
	query := bson.M{
		"status": bson.D{{"$ne", "deleted"}},
	}

	// Adding appId to query parameters if not empty
	if params.AppId != "" {
		query["appId"] = params.AppId
	}

	// Adding OpenId and Status to query parameters if not empty
	if params.OpenId != "" && params.Status != "" {
		query["$or"] = bson.A{
			bson.M{"openId": params.OpenId},
			bson.M{
				"$or": bson.A{
					bson.M{
						"$and": bson.A{
							bson.M{"statusSync": bson.M{"$exists": true}},
							bson.M{"statusSync." + params.OpenId: params.Status},
						},
					},
					bson.M{
						"$and": bson.A{
							bson.M{"target": params.OpenId},
							bson.M{"status": params.Status},
						},
					},
				},
			},
		}
	} else if params.OpenId != "" {
		targetQuery := bson.M{"target": params.OpenId}
		query["$or"] = bson.A{
			bson.M{"openId": params.OpenId},
			targetQuery,
		}
	} else if params.Status != "" {
		query["status"] = params.Status
	}

	// Adding source to query parameters if not empty
	if params.Source != "" {
		query["source"] = params.Source
	}

	// Adding classId to query parameters if not empty
	if params.ClassIds != "" {
		query["classId"] = bson.D{{"$in", strings.Split(params.ClassIds, ",")}}
	} else {
		if params.ClassId != "" {
			query["classId"] = params.ClassId
		}
	}

	// Adding subclassId to query parameters if not empty
	if params.SubclassIds != "" {
		query["subclassId"] = bson.D{{"$in", strings.Split(params.SubclassIds, ",")}}
	} else {
		if params.SubclassId != "" {
			query["subclassId"] = params.SubclassId
		}
	}

	if strings.ToLower(params.Sort) != "" && (strings.ToLower(params.Sort) != "asc" && strings.ToLower(params.Sort) != "desc") {
		Log.Error("Sort must be one of the following: asc, desc. Default is asc")
		return models.QueryMessageResponse{}, CreateError(http.StatusBadRequest, 100008)
	} else {
		if strings.ToLower(params.Sort) == "" {
			params.Sort = "asc"
		}
	}

	if params.Order == "" {
		params.Order = "updatedAt"
	}

	// Count total messages filtered by query
	total, err := messageCollection.CountDocuments(ctx, query)
	if err != nil {
		// Returning error if failed to count messages
		Log.Error(err.Error())
		return models.QueryMessageResponse{}, CreateError(codes.Internal, 1001)
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

	//opts.SetProjection(bson.M{})

	// Finding messages by query
	cursor, err := messageCollection.Find(ctx, query, opts)
	if err != nil {
		// Returning error if failed to find messages
		Log.Error(err.Error())
		return models.QueryMessageResponse{}, CreateError(codes.Internal, 1001)
	}

	// Initializing class map
	var classMap = map[string]Configuration{}
	// Iterating through messages
	for cursor.Next(ctx) {
		var message Message
		var class Configuration
		// Decoding message
		err := cursor.Decode(&message)
		// Returning error and Logging if failed to decode message
		if err != nil {
			Log.Error(err.Error())
			continue
		}

		// Getting class configuration from class map
		classObj := classMap[message.ClassId]
		if classObj.Id == primitive.NilObjectID {
			// Getting message class configuration
			// Setting Filter
			filter := bson.M{"classId": message.SubClassId}

			// Finding Message Class Configuration
			err = configurationCollection.FindOne(ctx, filter).Decode(&class)

			// Returning error and Logging if failed to find message class configuration
			if err != nil {
				Log.Error(err.Error())
				return models.QueryMessageResponse{}, CreateError(codes.NotFound, 100112)
			}

			// Setting class configuration to class map
			classMap[message.SubClassId] = class
			classObj = class
		}
		// Checking class configuration message type and appending message to message array accordingly
		if classObj.Type == "exclusive" {
			if reflect.TypeOf(message.Target).Kind() == reflect.Slice {
				// Iterating through open id in message target
				var receivers []string
				for _, receiver := range message.Target {
					if reflect.TypeOf(receiver).Kind() == reflect.String {
						receivers = append(receivers, receiver)
					}
				}
				if message.OpenId == params.OpenId || contains(receivers, params.OpenId) {
					messages = append(messages, message)
				}
			}
		} else {
			// Appending message class configuration to message class configurations array
			messages = append(messages, message)
		}
	}

	// Returning error if no message class configurations found
	if len(messages) == 0 {
		messages = []Message{}
	}
	queryMessageResponse := models.QueryMessageResponse{
		Page:    params.Page,
		Limit:   params.Limit,
		Total:   total,
		Results: messages,
	}

	// Returning false if messages.go are queried successfully
	return queryMessageResponse, nil
}

// getMessages : Gets cached messages
func getMessages(ctx context.Context, openid string) ([]Message, error) {
	// Initializing messages
	var messages []Message

	// Returning error and Logging if failed to get messages from cache
	if err := getMessagesFromRedis(ctx, &messages, openid); err != nil {
		return nil, err
	}

	// Returning messages if successful
	return messages, nil
}

// httpGetMessages : Gets cached messages via http
func httpGetMessages(c *gin.Context) ([]Message, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// Initializing messages
	var messages []Message

	// Returning error and Logging if failed to get messages from cache
	if err := getMessagesFromRedis(ctx, &messages, c.GetHeader("Zixel-Open-Id")); err != nil {
		return nil, err
	}

	// Returning messages if successful
	return messages, nil
}

// createWebSocketConnection : create websocket connection for users
func createWebSocketConnection(c *gin.Context) error {
	ServerInstance.mu.Lock()
	defer ServerInstance.mu.Unlock()
	// Returning error and Logging if failed to create websocket connection
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		Log.Error(err.Error())
		framework.Error(c, framework.NewServiceError(1000, ""))
		return err
	}
	// Returning error and Logging if open id is empty
	openId := c.GetHeader("Zixel-Open-Id")
	if openId == "" {
		openId = c.Query("openId")
	}

	client := ServerInstance.clients[openId]
	client.mu.Lock()
	defer client.mu.Unlock()
	var mode string
	if contains(connectionModes, c.Query("mode")) {
		mode = c.Query("mode")
	} else {
		mode = "BlackList"
	}

	connection := &Connection{
		openId:    openId,
		conn:      conn,
		addr:      conn.RemoteAddr().String(),
		channels:  []string{},
		whiteList: []string{},
		blackList: []string{},
		mode:      mode,
		mu:        &sync.Mutex{},
		send:      make(chan []byte, 256),
	}

	client.connections[conn.RemoteAddr().String()] = connection

	// Adding client to WebSocketServer
	ServerInstance.clients[client.openId] = client

	// Running client reader and writer in goroutines
	go connection.writePump()
	go connection.readPump()

	// Returning nil if successful and send user success message
	message := gin.H{
		"message": "Connected to messenger service",
	}
	jsonMessage, err := json.Marshal(message)
	if err != nil {
		Log.Error(err.Error())
		return err
	}
	connection.send <- jsonMessage

	// Returning nil if successful
	return nil
}

// newMessage : New Message is a function which is used by the API to create a new message
func newMessage(request *models.SendMessageV2Request, configuration *Configuration) *Message {
	var message Message

	message.Id = primitive.NewObjectID()
	message.AppId = configuration.AppId
	message.ClassId = request.ClassId
	message.SubClassId = request.SubClassId
	message.MessageTemplateId = request.MessageTemplateId
	message.Params = request.Params
	message.Content = request.Content
	message.Source = request.Source
	message.Target = request.Targets

	if err := setDefaultMessageStatus(&message, configuration); err != nil {
		return nil
	}

	message.CreatedAt = time.Now()
	message.UpdatedAt = time.Now()

	if configuration.Mode == "email" {
		message.CustomEmailRequest = request.CustomEmailRequest
	} else if configuration.Mode == "sms" {
		message.CustomSmsRequest = request.CustomSmsRequest
	} else if configuration.Mode == "feishu" {
		message.CustomFeishuRequest = models.CustomFeishuRequest{
			MsgType: request.CustomFeishuRequestV2.MsgType,
		}
	} else if configuration.Mode == "dingtalk" {
		message.CustomDingtalkRequest = models.CustomDingtalkRequest{}
	} else if configuration.Mode == "third-party" {
		message.CustomFeishuRequest = models.CustomFeishuRequest{
			MsgType: request.CustomFeishuRequestV2.MsgType,
		}
		message.CustomDingtalkRequest = models.CustomDingtalkRequest{}
	}

	return &message
}

// findMessage : Find Message is a function which is used by the API to find a message
func findMessage(
	ctx context.Context,
	message *Message,
	class *Configuration,
	id string) error {

	// Returning error and Logging if failed to find message
	if err := findMessageById(ctx, message, id); err != nil {
		return err
	}

	// Returning error and Logging if failed to find message class configuration
	if err := getConfigurationByClassId(ctx, message.SubClassId, class); err != nil {
		return err
	}

	// Returning false if message and message class configuration is found
	return nil
}

// findMessageById : Find Message By Id is a function which is used by the API to find message by id
func findMessageById(ctx context.Context, message *Message, id string) error {
	var m Message
	// Returning error and Logging if failed to convert id to primitive object id
	objectId, err := ConvertToObjectId(id)
	if err != nil {
		return err
	}

	// Returning error and Logging if failed to find message
	val, err := database.RedisGetStringCtx(ctx, redisMessageKey+objectId.Hex())
	if err == nil {
		err := json.Unmarshal([]byte(*val), &m)
		if err != nil {
			Log.Error(err.Error())
			return CreateError(codes.Internal, 1000)
		}
		*message = m
		return nil
	} else {
		// Setting filter to find message
		filter := bson.M{"_id": objectId}

		// Finding message using message id from mongodb
		err = messageCollection.FindOne(ctx, filter).Decode(&message)

		// Returning error if message does not exist
		if err != nil {
			Log.Error(err.Error())
			return CreateError(codes.NotFound, 100306)
		}
	}

	// Returning nil if message is found
	return nil
}

// insertOneMessage : Insert One Message is a function which is used by the API to insert one message
func insertOneMessage(ctx context.Context, message *Message, class *Configuration) error {
	// Storing message in database if persisted is true in message class configuration
	if class.Persist {
		// Inserting message template into MongoDB
		_, err := messageCollection.InsertOne(ctx, message)

		// Returning error and Logging if failed to insert message template
		if err != nil {
			Log.Error(err.Error())
			return CreateError(codes.Internal, 1001)
		}
	} else {
		err := addMessageToRedis(ctx, message)
		if err != nil {
			return err
		}
	}

	// Returning nil if message is inserted successfully
	return nil
}

// updateOneMessage : Update One Message is a function which is used by the API to update one message
func updateOneMessage(ctx context.Context, message *Message, class *Configuration) error {
	if class.Persist {
		// Setting filter to find message to update
		filter := bson.M{"_id": message.Id}

		// Updating message template in MongoDB
		update := bson.D{{"$set", bson.D{
			{"status", message.Status},
			{"updatedAt", message.UpdatedAt},
			{"statusSync", message.StatusSync},
		}}}

		_, err := messageCollection.UpdateOne(ctx, filter, update)

		// Returning error if failed to update message
		if err != nil {
			Log.Error(err.Error())
			return CreateError(codes.Internal, 1001)
		}
	} else {
		var m Message
		// Updating message in memory if persisted is false in message class configuration
		val, err := database.RedisGetStringCtx(ctx, redisMessageKey+message.Id.Hex())
		if err == nil {
			err := json.Unmarshal([]byte(*val), &m)
			if err != nil {
				Log.Error(err.Error())
				return CreateError(codes.Internal, 1000)
			}
			m.Status = message.Status
			m.StatusSync = message.StatusSync
			m.UpdatedAt = message.UpdatedAt
			if err := addMessageToRedis(ctx, &m); err != nil {
				return err
			}
		} else {
			Log.Error(err.Error())
			return CreateError(codes.NotFound, 100306)
		}
	}

	// Returning nil if message is updated successfully
	return nil
}

// addMessageToRedis : Add Message To Redis is a function which is used by the API to add message to redis
func addMessageToRedis(ctx context.Context, message *Message) error {
	// Storing message in memory if persisted is false in message class configuration
	binary, err := MarshalBinary(*message)
	if err != nil {
		Log.Error(err.Error())
		return CreateError(codes.Internal, 1000)
	}
	timeout := time.Hour * 24
	// Returning error and Logging if failed to set message in cache
	err = database.RedisSetCtx(ctx, redisMessageKey+message.Id.Hex(), binary, &timeout)
	if err != nil {
		Log.Error(err.Error())
		return CreateError(codes.Internal, 1001)
	} else {
		return nil
	}
}

// getMessagesFromRedis : Get Messages From Redis is a function which is used by the API to get messages from redis
func getMessagesFromRedis(ctx context.Context, message *[]Message, openId string) error {
	var m []Message
	val, err := database.RedisGetStringCtx(ctx, redisUserMessagesKey+openId)
	if err == nil {
		err := json.Unmarshal([]byte(*val), &m)
		if err != nil {
			Log.Error(err.Error())
			return CreateError(codes.Internal, 1000)
		}
		*message = m
		err = database.RedisMultiDel([]string{redisUserMessagesKey + openId})
		if err != nil {
			Log.Error(err.Error())
			return CreateError(codes.Internal, 1001)
		} else {
			return nil
		}
	} else {
		return nil
	}
}

// queryMessages : Queries messages
func queryMessagesByFilter(ctx context.Context, filterType string, openId string, Ids []string, status string) ([]Message, error) {
	// Initializing messages
	var messages []Message

	// Initializing query parameters
	query := bson.M{}

	query[filterType] = bson.D{{"$in", Ids}}
	query["$or"] = bson.A{
		bson.M{
			"$and": bson.A{
				bson.M{"statusSync": bson.M{"$exists": true}},
				bson.M{"statusSync." + openId: bson.M{"$ne": status}},
			},
		},
		bson.M{
			"$and": bson.A{
				bson.M{"statusSync": bson.M{"$exists": false}},
				bson.M{
					"$and": bson.A{
						bson.M{"target": bson.M{"$in": []interface{}{openId}}},
						bson.M{"status": bson.M{"$ne": status}},
					},
				},
			},
		},
	}

	// Finding messages by query
	cursor, err := messageCollection.Find(ctx, query)
	if err != nil {
		// Returning error if failed to find messages
		Log.Error(err.Error())
		return nil, CreateError(codes.Internal, 1001)
	} else {
		// Iterating through messages
		for cursor.Next(ctx) {
			var message Message
			// Decoding message
			err := cursor.Decode(&message)
			// Returning error and Logging if failed to decode message
			if err != nil {
				Log.Error(err.Error())
				continue
			}
			// Appending message to messages
			messages = append(messages, message)
		}
	}

	// Returning false if messages.go are queried successfully
	return messages, nil
}

// addMessageToUserCachedMessagesRedis : Add Message To User Cached Messages Redis is a function which is used by the API to add message to user cached messages redis
func addMessageToUserCachedMessagesRedis(ctx context.Context, openId string, message *Message) error {
	var m []Message
	val, err := database.RedisGetStringCtx(ctx, redisUserMessagesKey+openId)
	if err == nil {
		err := json.Unmarshal([]byte(*val), &m)
		if err != nil {
			Log.Error(err.Error())
			return CreateError(codes.Internal, 1000)
		}
		m = append(m, *message)
		binary, err := MarshalBinary(m)
		if err != nil {
			Log.Error(err.Error())
			return CreateError(codes.Internal, 1000)
		}
		err = database.RedisSetCtx(ctx, redisUserMessagesKey+openId, binary, &infiniteDuration)
		if err != nil {
			Log.Error(err.Error())
			return CreateError(codes.Internal, 1001)
		}
		return nil
	} else {
		m = append(m, *message)
		binary, err := MarshalBinary(m)
		if err != nil {
			Log.Error(err.Error())
			return CreateError(codes.Internal, 1000)
		}
		err = database.RedisSetCtx(ctx, redisUserMessagesKey+openId, binary, &infiniteDuration)
		if err != nil {
			Log.Error(err.Error())
			return CreateError(codes.Internal, 1001)
		}
	}
	return nil
}

// queryMessagesByIds
func redirectMessageToSockets(message *Message, configuration *Configuration) {
	ServerInstance.mu.Lock()
	defer ServerInstance.mu.Unlock()
	var success []string
	for _, receiver := range message.Target {
		// Sending message to user if user connection is available
		check := sendMessageToUserSocket(receiver, message, configuration, false)
		if check {
			success = append(success, receiver)
		}
	}

	// Sending message to application if application connection is available
	if applicationConnections, ok := ServerInstance.ApplicationConnections[message.ClassId]; ok {
		for _, connection := range applicationConnections {
			connection.send <- message.Encode()
		}
	}

	// Remove all the receivers from the message who have received the message
	if len(success) > 1 {
		for _, receiver := range success {
			for i, target := range message.Target {
				if target == receiver {
					message.Target = append(message.Target[:i], message.Target[i+1:]...)
				}
			}
		}
	}

	for k, v := range ServerInstance.ToMessengerConnectionInstances {
		if v.conn != nil {
			v.send <- message.Encode()
		} else {
			Log.Error("Connection is nil: ", k)
		}
	}
}

// sendMessageToUserSocket : Send Message To User Socket is a function which is used by the API to send message to user socket
func sendMessageToUserSocket(openId string, message *Message, configuration *Configuration, persist bool) bool {
	if client, ok := ServerInstance.clients[openId]; ok {
		client.mu.Lock()
		defer client.mu.Unlock()
		for _, connection := range client.connections {
			func() {
				connection.mu.Lock()
				defer connection.mu.Unlock()
				if connection.conn != nil {
					if configuration.Mode == "listening" {
						if !contains(connection.channels, message.ClassId) {
							return
						}
					}

					if connection.mode == "BlackList" {
						if contains(connection.blackList, message.ClassId) {
							return
						}
					} else if connection.mode == "WhiteList" {
						if !contains(connection.whiteList, message.ClassId) {
							return
						}
					}

					connection.send <- message.Encode()
				} else {
					Log.Error("connection is nil for " + connection.openId)
				}
			}()
		}
		return true
	} else {
		if persist {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			// Returning error and Logging if failed to add message to user cached messages
			if err := addMessageToUserCachedMessagesRedis(ctx, openId, message); err != nil {
				Log.Error(err.Error())
			}
		}
		return false
	}
}

// insertOneLinkingMessage : Insert One Linking Message is a function which is used by the API to insert one linking message
func insertOneLinkingMessage(ctx context.Context, linkingMessage *Message, class *Configuration) error {
	if class.Persist {
		// Inserting message template into MongoDB
		_, err := messageCollection.InsertOne(ctx, linkingMessage)

		// Returning error and Logging if failed to insert message template
		if err != nil {
			Log.Error(err.Error())
			return CreateError(codes.Internal, 1001)
		}
	} else {
		// Storing message in memory if persisted is false in message class configuration
		if err := addMessageToRedis(ctx, linkingMessage); err != nil {
			return err
		}
	}
	return nil
}

// updateMessageInUserCachedMessagesRedis : Update Message In User Cached Messages Redis is a function which is used by the API to update message in user cached messages redis
func updateMessageInUserCachedMessagesRedis(ctx context.Context, openId string, message Message) error {
	var m []Message
	val, err := database.RedisGetStringCtx(ctx, redisUserMessagesKey+openId)
	if err == nil {
		err := json.Unmarshal([]byte(*val), &m)
		if err != nil {
			Log.Error(err.Error())
			return CreateError(codes.Internal, 1000)
		}
		// Iterating through all messages.go in user cache and updating message status
		var check bool
		for i := 0; i < len(m); i++ {
			// Checking if message id is same as message id in cache
			if m[i].Id == message.Id {
				m[i] = message
				check = true
				break
			}
		}
		// Adding message to cache if it does not exist in cache
		if !check {
			m = append(m, message)
		}

		binary, err := MarshalBinary(m)
		if err != nil {
			Log.Error(err.Error())
			return CreateError(codes.Internal, 1000)
		}
		err = database.RedisSetCtx(ctx, redisUserMessagesKey+openId, binary, &infiniteDuration)
		if err != nil {
			Log.Error(err.Error())
			return CreateError(codes.Internal, 1001)
		}
		return err
	} else {
		m = append(m, message)
		binary, err := MarshalBinary(m)
		if err != nil {
			Log.Error(err.Error())
			return CreateError(codes.Internal, 1000)
		}
		err = database.RedisSetCtx(ctx, redisUserMessagesKey+openId, binary, &infiniteDuration)
		if err != nil {
			Log.Error(err.Error())
			return CreateError(codes.Internal, 1001)
		}
	}
	return nil
}

// findLinkingMessage : Find Linking Message is a function which is used by the API to find linking message
func findLinkingMessage(ctx context.Context, linkageId primitive.ObjectID, linkingMessage *Message, class *Configuration) error {
	if class.Persist {
		// Setting filter to find message template to update
		filter := bson.M{"_id": linkageId}

		// Finding message using message id from mongodb
		err := messageCollection.FindOne(ctx, filter).Decode(&linkingMessage)

		// Returning error if message does not exist
		if err != nil {
			Log.Error(err.Error())
			return CreateError(codes.NotFound, 100306)
		}
	} else {
		var m Message
		// Finding message
		val, err := database.RedisGetStringCtx(ctx, redisMessageKey+linkingMessage.Id.Hex())
		if err == nil {
			err := json.Unmarshal([]byte(*val), &m)
			if err != nil {
				Log.Error(err.Error())
				return CreateError(codes.Internal, 1000)
			}
			*linkingMessage = m
		} else {
			Log.Error(err.Error())
			return CreateError(codes.NotFound, 100306)
		}
	}
	return nil
}

// validateMessageParentClassId : Validate Message Parent ClassId is a function which is used by the API to validate message parent class id
func validateMessageParentClassId(classId string, parentId string) error {
	if parentId != classId {
		Log.Error("Parent class id does not match class id")
		return CreateError(codes.InvalidArgument, 100302)
	}
	return nil
}

// validateMessageTarget : Validate Message Target is a function which is used by the API to validate message target
func validateMessageTarget(target interface{}) error {
	if reflect.TypeOf(target).Kind() == reflect.String {
		if !contains(targetTypes, target.(string)) {
			Log.Error("Target type is invalid")
			return CreateError(codes.InvalidArgument, 100305)
		}
	} else if reflect.TypeOf(target).Kind() == reflect.Slice {
		return nil
	} else {
		Log.Error("Target type is invalid")
		return CreateError(codes.InvalidArgument, 100305)
	}
	return nil
}

// validateMessageTarget : Validate Message Target is a function which is used by the API to validate message target
func validateMessageTargetV2(targets []string, mode string) error {
	if mode == "push" || mode == "pull" || mode == "listening" || mode == "third-party" || mode == "feishu" || mode == "dingtalk" {
		for _, target := range targets {
			// check if t is a valid open id by checking if it starts with oi-
			if !strings.HasPrefix(target, "oi_") {
				Log.Error("Target type is invalid")
				return CreateError(codes.InvalidArgument, 100305)
			}
		}
	} else if mode == "email" || mode == "sms" {
		if mode == "email" {
			for _, target := range targets {
				_, err := mail.ParseAddress(target)
				if err != nil && !strings.HasPrefix(target, "oi_") {
					Log.Error("Target type is invalid")
					return CreateError(codes.InvalidArgument, 100305)
				}
			}
		} else {
			for _, target := range targets {
				_, err := regexp.Match(phoneRegex, []byte(target))
				// check if t is a valid phone number
				if err != nil && !strings.HasPrefix(target, "oi_") {
					Log.Error("Target type is invalid")
					return CreateError(codes.InvalidArgument, 100305)
				}
			}
		}
	} else {
		Log.Error("Target type is invalid")
		return CreateError(codes.InvalidArgument, 100305)
	}

	return nil
}

// setDefaultMessageStatus : Set Default Message Status is a function which is used by the API to set default message status
func setDefaultMessageStatus(message *Message, class *Configuration) error {
	// Setting message status sync if message type is broadcast
	if class.Type == "broadcast" {
		if message.StatusSync == nil {
			message.StatusSync = make(map[string]string)
		}
		if reflect.TypeOf(message.Target).Kind() == reflect.Slice {
			// Iterating through all openid in message target
			for _, receiver := range message.Target {
				if reflect.TypeOf(receiver).Kind() == reflect.String {
					message.StatusSync[receiver] = class.States[0]
				}
			}
		} else if reflect.TypeOf(message.Target).Kind() == reflect.String {
			message.Status = class.States[0]
		} else {
			Log.Error("Target type is invalid")
			return CreateError(codes.InvalidArgument, 100305)
		}
	} else {
		// Setting message status
		message.Status = class.States[0]
	}
	return nil
}

// validateMessageStatusToBeUpdatedNotDeleted : Validate Message Status To Be Updated Not Deleted is a function which is used by the API to validate message status to be updated not deleted
func validateMessageStatusToBeUpdatedNotDeleted(status string) error {
	if status == "deleted" {
		Log.Error("Message status cannot be deleted")
		return CreateError(codes.InvalidArgument, 100309)
	}
	return nil
}

// validateMessageStatusExistInMessageClassStates : Validate Message Status Exist In Message Class States is a function which is used by the API to validate message status exist in message class states
func validateMessageStatusExistInMessageClassStates(status string, states []string) error {
	if !contains(states, status) {
		Log.Error("Message status does not exist in message class states")
		return CreateError(codes.InvalidArgument, 100307)
	}
	return nil
}

// validateMessageStatusToBeUpdatedEqualToCurrentStatus : Validate Message Status To Be Updated Equal To Current Status is a function which is used by the API to validate message status to be updated equal to current status
func validateMessageStatusToBeUpdatedEqualToCurrentStatus(status string, currentStatus string) error {
	if status == currentStatus {
		Log.Error("Message status to be updated is equal to current status")
		return CreateError(codes.InvalidArgument, 100308)
	}
	return nil
}

// validateMessageStatusIsDeleted : Validate Message Status Is Deleted is a function which is used by the API to validate message status is deleted
func validateMessageStatusIsDeleted(currentStatus string) error {
	if currentStatus == "deleted" {
		// Returning error and Logging if tried to change status from Deleted to another status
		Log.Error("Message status is deleted")
		return CreateError(codes.InvalidArgument, 100316)
	}
	return nil
}

// validateMessagePermissionToUpdateMessageStatusExclusiveMessage: Validate Message Permission To Update Message Status Exclusive Message is a function which is used by the API to validate message permission to update message status exclusive message
func validateMessagePermissionToUpdateMessageStatusExclusiveMessage(openId string, allowedUsers []string) error {
	if !contains(allowedUsers, openId) {
		Log.Error("User does not have permission to update message status")
		return CreateError(codes.PermissionDenied, 100313)
	}
	return nil
}

// validateMessageStatusForBroadcastMessage : Validate Message Status For Broadcast Message is a function which is used by the API to validate message status for broadcast message
func validateMessageStatusForBroadcastMessage(message *Message, status string, openId string) error {
	// Returning error and Logging if the message status to update is same as the current status
	if message.StatusSync[openId] != status {
		// Returning error and Logging if tried to change status from Deleted to another status
		if message.StatusSync[openId] != "deleted" {
			message.StatusSync[openId] = status
		} else {
			Log.Error("Message status is deleted")
			return CreateError(codes.InvalidArgument, 100316)
		}
	} else {
		Log.Error("Message status to be updated is equal to current status")
		return CreateError(codes.InvalidArgument, 100308)
	}
	return nil
}

// readPump : read messages from client
func (connection *Connection) readPump() {
	// Deferring closing of connection
	defer func() {
		connection.disconnect()
	}()

	// Setting connection read deadline and read limit
	connection.conn.SetReadLimit(maxMessageSize)
	err := connection.conn.SetReadDeadline(time.Now().Add(pongWait))

	// Logging error if failed to set read deadline
	if err != nil {
		Log.Error("Failed to set read deadline")
		return
	}

	// Logging error if failed to set pong handler
	connection.conn.SetPongHandler(func(string) error {
		err := connection.conn.SetReadDeadline(time.Now().Add(pongWait))
		if err != nil {
			return err
		}
		return nil
	})

	// Start endless read loop, waiting for messages from connection
	for {
		// Reading message from connection
		_, jsonMessage, err := connection.conn.ReadMessage()

		// Logging error if failed to read message and breaking loop
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				Log.Error("Error: %v", err)
			}
			break
		} else {
			// Handling message from connection
			go connection.handleNewMessage(jsonMessage)
		}
	}
}

// writePump : write messages to client
func (connection *Connection) writePump() {
	// Initializing ticker
	ticker := time.NewTicker(pingPeriod)

	// Deferring closing of connection and ticker
	defer func() {
		ticker.Stop()
	}()

	// Start endless write loop, waiting for messages from connection
	for {
		select {
		// Waiting for send channel message from connection
		case message, ok := <-connection.send:
			func() {
				connection.mu.Lock()
				defer connection.mu.Unlock()
				// Logging error if failed to set write deadline and breaking loop
				err := connection.conn.SetWriteDeadline(time.Now().Add(writeWait))
				if err != nil {
					Log.Error("Failed to set write deadline")
					return
				}
				// Logging error if failed to write message and breaking loop
				if !ok {
					// The WsServer closed the channel.
					err := connection.conn.WriteMessage(websocket.CloseMessage, []byte{})
					if err != nil {
						return
					}
				}

				// Logging error if failed get io.writeClosure
				w, err := connection.conn.NextWriter(websocket.TextMessage)
				if err != nil {
					Log.Error("Failed to get io.writeClosure")
					return
				}

				// Logging error if failed to write message
				_, err = w.Write(message)
				if err != nil {
					Log.Error("Failed to write message")
					return
				}

				// Attach queued chat messages to the current websocket message.
				n := len(connection.send)

				// Iterating through all queued messages
				for i := 0; i < n; i++ {
					// Logging error if failed to add newline to message
					_, err := w.Write(newline)
					if err != nil {
						Log.Error("Failed to add newline to message")
						return
					}

					// Logging error if failed to get message from send channel and writing message
					_, err = w.Write(<-connection.send)
					if err != nil {
						Log.Error("Failed to get message from send channel and writing message")
						return
					}
				}

				// Logging error if failed to close writer
				if err := w.Close(); err != nil {
					Log.Error("Failed to close writer")
					return
				}
			}()

		// Calling this case when ticker ticks
		case <-ticker.C:
			// Logging error if failed to set write deadline and breaking loop
			err := connection.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err != nil {
				Log.Error("Failed to set write deadline")
				return
			}

			// Logging error if failed to write ping message and breaking loop
			if err := connection.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// disconnect : disconnect client from server
func (connection *Connection) disconnect() {
	ServerInstance.mu.Lock()
	defer ServerInstance.mu.Unlock()
	connection.mu.Lock()
	defer connection.mu.Unlock()
	if client, ok := ServerInstance.clients[connection.openId]; ok {
		client.mu.Lock()
		defer client.mu.Unlock()
		if _, ok := client.connections[connection.addr]; ok {
			delete(client.connections, connection.addr)
		}
		if len(client.connections) == 0 {
			delete(ServerInstance.clients, connection.openId)
		}
	}

	if len(connection.channels) > 0 {
		for _, channel := range connection.channels {
			var users []string
			val, err := database.RedisGetString(redisUserListKey + channel)
			if err != nil {
				Log.Error("Failed to get users from redis")
			} else {
				err := json.Unmarshal([]byte(*val), &users)
				if err != nil {
					Log.Error(err.Error())
				}
			}
			if !contains(users, connection.openId) {
				Log.Error("User is not registered in configuration")
			} else {
				// remove user from list
				for i, user := range users {
					if user == connection.openId {
						users = append(users[:i], users[i+1:]...)
						break
					}
				}

				binary, err := MarshalBinary(users)
				if err != nil {
					Log.Error(err.Error())
					connection.send <- CreateErrorMessage("Internal server error.")
				}
				err = database.RedisSet(redisUserListKey+channel, binary, &infiniteDuration)
				if err != nil {
					Log.Error("Failed to remove user to configuration")
					if connection.conn != nil {
						connection.send <- CreateErrorMessage("Failed to remove user to configuration")
						return
					}
				} else {
					if connection.conn != nil {
						connection.send <- CreateSystemMessage(SuccessAction, "User removed from configuration")
					}
				}
			}
		}
	}

	// Logging if failed to close connection send channel and connection
	err := connection.conn.Close()
	if err != nil {
		Log.Error("Failed to close connection send channel and connection")
		return
	}

	Log.Info(connection.openId + " disconnected " + connection.conn.RemoteAddr().String())
}

// handleNewMessage : handle message from client based on message action
func (connection *Connection) handleNewMessage(message []byte) {
	var webSocketMessage WebSocketMessage
	err := json.Unmarshal(message, &webSocketMessage)
	if err != nil {
		Log.Error("Failed to unmarshal message")
		if connection.conn != nil {
			connection.send <- CreateErrorMessage(err.Error())
		}
	} else {
		switch webSocketMessage.Action {
		case AddConfigurationAction:
			connection.addConfiguration(webSocketMessage)
		case DeleteConfigurationAction:
			connection.deleteConfiguration(webSocketMessage)
		case AddToBlacklistAction:
			connection.addToBlacklist(webSocketMessage)
		case RemoveFromBlacklistAction:
			connection.removeFromBlacklist(webSocketMessage)
		case AddToWhitelistAction:
			connection.addToWhitelist(webSocketMessage)
		case RemoveFromWhitelistAction:
			connection.removeFromWhitelist(webSocketMessage)
		case ChangeFilteringModeAction:
			connection.changeFilteringMode(webSocketMessage)
		default:
			Log.Error("Invalid action")
			if connection.conn != nil {
				connection.send <- CreateErrorMessage("Invalid action")
			}
		}
	}
}

func (connection *Connection) addConfiguration(webSocketMessage WebSocketMessage) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	connection.mu.Lock()
	defer connection.mu.Unlock()
	var configuration Configuration
	err := getConfigurationByClassId(ctx, webSocketMessage.Data, &configuration)
	if err != nil {
		Log.Error("Failed to get configuration")
		if connection.conn != nil {
			connection.send <- CreateErrorMessage(err.Error())
		}
	} else {
		if configuration.Mode != "listening" {
			if connection.conn != nil {
				Log.Error("Configuration is not in listening mode")
				connection.send <- CreateErrorMessage("Configuration is not in listening mode")
			}
		} else {
			if contains(connection.channels, configuration.ClassId) {
				Log.Error("User already in configuration")
				if connection.conn != nil {
					connection.send <- CreateErrorMessage("User already in configuration")
					return
				}
			} else {
				connection.channels = append(connection.channels, configuration.ClassId)
			}

			var users map[string][]string
			val, err := database.RedisGetStringCtx(ctx, redisUserListKey+configuration.ClassId)
			if err != nil {
				Log.Error("Failed to get users from redis")
				if connection.conn != nil {
					connection.send <- CreateErrorMessage("Failed to get user list in configuration")
					return
				}
			} else {
				err := json.Unmarshal([]byte(*val), &users)
				if err != nil {
					Log.Error(err.Error())
					if connection.conn != nil {
						connection.send <- CreateErrorMessage("Internal server error.")
						return
					}
				}
			}
			userConnections := users[connection.openId]
			userConnections = append(userConnections, connection.conn.RemoteAddr().String())
			binary, err := MarshalBinary(users)
			if err != nil {
				Log.Error(err.Error())
				connection.send <- CreateErrorMessage("Internal server error.")
			}
			err = database.RedisSetCtx(ctx, redisUserListKey+configuration.ClassId, binary, &infiniteDuration)
			if err != nil {
				Log.Error("Failed to add user to configuration")
				if connection.conn != nil {
					connection.send <- CreateErrorMessage("Failed to add user to configuration")
					return
				}
			} else {
				if connection.conn != nil {
					connection.send <- CreateSystemMessage(SuccessAction, "User added to configuration")
				}
			}
		}
	}
}

func (connection *Connection) deleteConfiguration(webSocketMessage WebSocketMessage) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	connection.mu.Lock()
	defer connection.mu.Unlock()
	var configuration Configuration
	err := getConfigurationByClassId(ctx, webSocketMessage.Data, &configuration)
	if err != nil {
		Log.Error("Failed to get configuration")
		if connection.conn != nil {
			connection.send <- CreateErrorMessage(err.Error())
		}
	} else {
		if configuration.Mode != "listening" {
			if connection.conn != nil {
				Log.Error("Configuration is not in listening mode")
				connection.send <- CreateErrorMessage("Configuration is not in listening mode")
			}
		} else {
			if !contains(connection.channels, configuration.ClassId) {
				Log.Error("User is not registered in configuration")
				if connection.conn != nil {
					connection.send <- CreateErrorMessage("User is not registered in configuration")
					return
				}
			} else {
				for i, channel := range connection.channels {
					if channel == configuration.ClassId {
						connection.channels = append(connection.channels[:i], connection.channels[i+1:]...)
						break
					}
				}
			}

			ServerInstance.mu.Lock()
			defer ServerInstance.mu.Unlock()
			client := ServerInstance.clients[connection.openId]
			exist := false
			for _, connection := range client.connections {
				if contains(connection.channels, configuration.ClassId) {
					exist = true
					break
				}
			}

			if !exist {
				var users map[string][]string
				val, err := database.RedisGetStringCtx(ctx, redisUserListKey+configuration.ClassId)
				if err != nil {
					Log.Error("Failed to get users from redis")
					if connection.conn != nil {
						connection.send <- CreateErrorMessage("Failed to get user list in configuration")
						return
					}
				} else {
					err := json.Unmarshal([]byte(*val), &users)
					if err != nil {
						Log.Error(err.Error())
						if connection.conn != nil {
							connection.send <- CreateErrorMessage("Internal server error.")
							return
						}
					}
				}
				userConnections := users[connection.openId]
				for i, userConnection := range userConnections {
					if userConnection == connection.conn.RemoteAddr().String() {
						userConnections = append(userConnections[:i], userConnections[i+1:]...)
						break
					}
				}
				users[connection.openId] = userConnections

				if len(userConnections) == 0 {
					delete(users, connection.openId)
				}

				binary, err := MarshalBinary(users)
				if err != nil {
					Log.Error(err.Error())
					connection.send <- CreateErrorMessage("Internal server error.")
				}
				err = database.RedisSetCtx(ctx, redisUserListKey+configuration.ClassId, binary, &infiniteDuration)
				if err != nil {
					Log.Error("Failed to remove user to configuration")
					if connection.conn != nil {
						connection.send <- CreateErrorMessage("Failed to remove user to configuration")
						return
					}
				} else {
					if connection.conn != nil {
						connection.send <- CreateSystemMessage(SuccessAction, "User removed from configuration")
					}
				}
			}
		}
	}
}

func (connection *Connection) addToBlacklist(webSocketMessage WebSocketMessage) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	connection.mu.Lock()
	defer connection.mu.Unlock()
	var configuration Configuration
	err := getConfigurationByClassId(ctx, webSocketMessage.Data, &configuration)
	if err != nil {
		Log.Error("Failed to get configuration")
		if connection.conn != nil {
			connection.send <- CreateErrorMessage(err.Error())
		}
	} else {
		if contains(connection.blackList, webSocketMessage.Data) {
			Log.Error("Configuration is already in blacklist.")
			if connection.conn != nil {
				connection.send <- CreateErrorMessage("Configuration is already in blacklist.")
				return
			}
		} else {
			connection.blackList = append(connection.blackList, webSocketMessage.Data)
			if connection.conn != nil {
				connection.send <- CreateSystemMessage(SuccessAction, "Configuration added to blacklist")
			}
		}
	}
}

func (connection *Connection) removeFromBlacklist(webSocketMessage WebSocketMessage) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	connection.mu.Lock()
	defer connection.mu.Unlock()
	var configuration Configuration
	err := getConfigurationByClassId(ctx, webSocketMessage.Data, &configuration)
	if err != nil {
		Log.Error("Failed to get configuration")
		if connection.conn != nil {
			connection.send <- CreateErrorMessage(err.Error())
		}
	} else {
		if !contains(connection.blackList, webSocketMessage.Data) {
			Log.Error("Configuration is not in blacklist.")
			if connection.conn != nil {
				connection.send <- CreateErrorMessage("Configuration is not in blacklist.")
				return
			}
		} else {
			// remove user from list
			for i, channel := range connection.blackList {
				if channel == webSocketMessage.Data {
					connection.blackList = append(connection.blackList[:i], connection.blackList[i+1:]...)
					break
				}
			}

			connection.send <- CreateSystemMessage(SuccessAction, "Configuration removed from blacklist")
		}
	}
}

func (connection *Connection) addToWhitelist(webSocketMessage WebSocketMessage) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	connection.mu.Lock()
	defer connection.mu.Unlock()
	var configuration Configuration
	err := getConfigurationByClassId(ctx, webSocketMessage.Data, &configuration)
	if err != nil {
		Log.Error("Failed to get configuration")
		if connection.conn != nil {
			connection.send <- CreateErrorMessage(err.Error())
		}
	} else {
		if contains(connection.whiteList, webSocketMessage.Data) {
			Log.Error("Configuration is already in whitelist.")
			if connection.conn != nil {
				connection.send <- CreateErrorMessage("Configuration is already in whitelist.")
				return
			}
		} else {
			connection.whiteList = append(connection.whiteList, webSocketMessage.Data)
			connection.send <- CreateSystemMessage(SuccessAction, "Configuration added to whitelist")
		}
	}
}

func (connection *Connection) removeFromWhitelist(webSocketMessage WebSocketMessage) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	connection.mu.Lock()
	defer connection.mu.Unlock()
	var configuration Configuration
	err := getConfigurationByClassId(ctx, webSocketMessage.Data, &configuration)
	if err != nil {
		Log.Error("Failed to get configuration")
		if connection.conn != nil {
			connection.send <- CreateErrorMessage(err.Error())
		}
	} else {
		if !contains(connection.whiteList, webSocketMessage.Data) {
			Log.Error("Configuration is not in whitelist.")
			if connection.conn != nil {
				connection.send <- CreateErrorMessage("Configuration is not in whitelist.")
				return
			}
		} else {
			// remove user from list
			for i, channel := range connection.whiteList {
				if channel == webSocketMessage.Data {
					connection.whiteList = append(connection.whiteList[:i], connection.whiteList[i+1:]...)
					break
				}
			}

			connection.send <- CreateSystemMessage(SuccessAction, "Configuration removed from whitelist")
		}
	}
}

func (connection *Connection) changeFilteringMode(webSocketMessage WebSocketMessage) {
	if contains(connectionModes, webSocketMessage.Data) {
		connection.mode = webSocketMessage.Data
		if connection.conn != nil {
			connection.send <- CreateSystemMessage(SuccessAction, "Filtering mode changed to "+webSocketMessage.Data)
		}
	} else {
		Log.Error("Invalid mode")
		if connection.conn != nil {
			connection.send <- CreateErrorMessage("Invalid mode")
		}
	}
}

// DownloadFile will download a url to a local file. It's efficient because it will
func DownloadFile(url string) ([]byte, error) {
	//fileName := path2.Base(url)
	//path := "../bin/temp/" + fileName
	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		Log.Error(err.Error())
		return []byte{}, CreateError(codes.Internal, 100009)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			Log.Error(err.Error())
		}
	}(resp.Body)

	// Read the body into a byte array
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, CreateError(codes.Internal, 100009)
	} else {
		return body, nil
	}
}

func buildWsseHeader(appKey, appSecret string) string {
	const WsseHeaderFormat = "UsernameToken Username=\"%s\",PasswordDigest=\"%s\",Nonce=\"%s\",Created=\"%s\""

	var cTime = time.Now().Format("2006-01-02T15:04:05Z")
	var nonce = uuid.NewV4().String()
	nonce = strings.ReplaceAll(nonce, "-", "")

	h := sha256.New()
	h.Write([]byte(nonce + cTime + appSecret))
	passwordDigestBase64Str := base64.StdEncoding.EncodeToString(h.Sum(nil))

	return fmt.Sprintf(WsseHeaderFormat, appKey, passwordDigestBase64Str, nonce, cTime)
}

func sendEmail(ctx context.Context, message *Message, configuration *Configuration) error {
	msg := gomail.NewMessage()
	msg.SetHeader("From", configuration.CustomEmailConfiguration.EmailAccount)

	var to []string
	for _, receiver := range message.Target {
		if _, err := mail.ParseAddress(receiver); err == nil {
			to = append(to, receiver)
		} else {
			if userInfo, err := getUserContact(ctx, receiver); err != nil {
				Log.Error(err.Error())
			} else {
				to = append(to, userInfo.GetEmail())
			}
		}
	}
	var cc []string
	for _, receiver := range message.CustomEmailRequest.Cc {
		if _, err := mail.ParseAddress(receiver); err == nil {
			cc = append(cc, receiver)
		} else {
			if userInfo, err := getUserContact(ctx, receiver); err != nil {
				Log.Error(err.Error())
			} else {
				cc = append(cc, userInfo.GetEmail())
			}
		}
	}
	var bcc []string
	for _, receiver := range message.CustomEmailRequest.Bcc {
		if _, err := mail.ParseAddress(receiver); err == nil {
			bcc = append(bcc, receiver)
		} else {
			if userInfo, err := getUserContact(ctx, receiver); err != nil {
				Log.Error(err.Error())
			} else {
				bcc = append(bcc, userInfo.GetEmail())
			}
		}
	}
	msg.SetHeader("To", to...)
	msg.SetHeader("Cc", cc...)
	msg.SetHeader("Bcc", bcc...)
	msg.SetHeader("Subject", message.CustomEmailRequest.Subject)

	if message.CustomEmailRequest.BodyType == "" {
		msg.SetBody("text/plain", message.Content)
	} else {
		msg.SetBody(message.CustomEmailRequest.BodyType, message.Content)
	}

	//var attachmentsPaths []string
	for _, attachment := range message.CustomEmailRequest.Attachments {
		file, err := DownloadFile(attachment)
		if err != nil {
			return err
		} else {
			// Attach file byte array to email
			msg.Attach(attachment, gomail.SetCopyFunc(func(w io.Writer) error {
				_, err := w.Write(file)
				return err
			}))
		}
	}

	password, err := AES_DecryptBase64(configuration.CustomEmailConfiguration.EmailPassword, []byte(configs.SecretKey))
	if err != nil {
		return err
	}

	// Sending email
	dialer := gomail.NewDialer("smtp.feishu.cn",
		587,
		configuration.CustomEmailConfiguration.EmailAccount,
		string(password))
	if err := dialer.DialAndSend(msg); err != nil {
		// Returning error and Logging if failed to send email
		Log.Error(err.Error())
		return CreateError(codes.Internal, 100318, map[string]string{"error": err.Error()})
	} else {
		// Returning false if email sent successfully
		return nil
	}
}

func sendEmailV2(ctx context.Context, message *Message, configuration *Configuration) ([]string, error) {
	msg := gomail.NewMessage()
	msg.SetHeader("From", configuration.CustomEmailConfiguration.EmailAccount)
	var failedTargets []string
	var to []string
	for _, receiver := range message.Target {
		if _, err := mail.ParseAddress(receiver); err == nil {
			to = append(to, receiver)
		} else {
			if userInfo, err := getUserInfoV2(ctx, receiver, GetContactInfoSelector); err != nil {
				Log.Error(err.Error())
				failedTargets = append(failedTargets, receiver)
			} else {
				to = append(to, userInfo.SensitiveInfo.GetEmail())
			}
		}
	}
	var cc []string
	for _, receiver := range message.CustomEmailRequest.Cc {
		if _, err := mail.ParseAddress(receiver); err == nil {
			cc = append(cc, receiver)
		} else {
			if userInfo, err := getUserInfoV2(ctx, receiver, GetContactInfoSelector); err != nil {
				Log.Error(err.Error())
				failedTargets = append(failedTargets, receiver)
			} else {
				to = append(to, userInfo.SensitiveInfo.GetEmail())
			}
		}
	}
	var bcc []string
	for _, receiver := range message.CustomEmailRequest.Bcc {
		if _, err := mail.ParseAddress(receiver); err == nil {
			bcc = append(bcc, receiver)
		} else {
			if userInfo, err := getUserInfoV2(ctx, receiver, GetContactInfoSelector); err != nil {
				Log.Error(err.Error())
				failedTargets = append(failedTargets, receiver)
			} else {
				to = append(to, userInfo.SensitiveInfo.GetEmail())
			}
		}
	}
	msg.SetHeader("To", to...)
	msg.SetHeader("Cc", cc...)
	msg.SetHeader("Bcc", bcc...)
	msg.SetHeader("Subject", message.CustomEmailRequest.Subject)

	if message.CustomEmailRequest.BodyType == "" {
		msg.SetBody("text/plain", message.Content)
	} else {
		msg.SetBody(message.CustomEmailRequest.BodyType, message.Content)
	}

	//var attachmentsPaths []string
	for _, attachment := range message.CustomEmailRequest.Attachments {
		file, err := DownloadFile(attachment)
		if err != nil {
			return failedTargets, err
		} else {
			// Attach file byte array to email
			msg.Attach(attachment, gomail.SetCopyFunc(func(w io.Writer) error {
				_, err := w.Write(file)
				return err
			}))
		}
	}

	password, err := AES_DecryptBase64(configuration.CustomEmailConfiguration.EmailPassword, []byte(configs.SecretKey))
	if err != nil {
		return failedTargets, err
	}

	// Sending email
	dialer := gomail.NewDialer("smtp.feishu.cn",
		587,
		configuration.CustomEmailConfiguration.EmailAccount,
		string(password))
	if err := dialer.DialAndSend(msg); err != nil {
		// Returning error and Logging if failed to send email
		Log.Error(err.Error())
		return failedTargets, CreateError(codes.Internal, 100318, map[string]string{"error": err.Error()})
	} else {
		// Returning false if email sent successfully
		return failedTargets, nil
	}
}

func sendSMS(_ context.Context, message *Message, configuration *Configuration) error {
	const AUTH_HEADER_VALUE = "WSSE realm=\"SDP\",profile=\"UsernameToken\",type=\"Appkey\""

	statusCallBack := ""
	headers := map[string]string{
		"Content-Type":  "application/x-www-form-urlencoded",
		"Authorization": AUTH_HEADER_VALUE,
		"X-WSSE":        buildWsseHeader(configs.HuaweiAppKey, configs.HuaweiAppSecret),
	}
	var receivers string
	for _, receiver := range message.Target {
		match, err := regexp.Match(phoneRegex, []byte(receiver))
		if err != nil {
			Log.Error(err.Error())
			continue
		}
		if match {
			receivers += receiver + ","
		} else {
			if userInfo, err := getUserContact(context.Background(), receiver); err != nil {
				Log.Error(err.Error())
			} else {
				receivers += userInfo.GetPhone() + ","
			}
		}
	}

	body := "from=" + url.QueryEscape(configuration.CustomSmsConfiguration.Huawei.Channel) + "&to=" + url.QueryEscape(receivers) + "&templateId=" + url.QueryEscape(configuration.CustomSmsConfiguration.Huawei.TemplateId)

	if len(message.CustomSmsRequest.TemplateParams) > 0 {
		templateParams, _ := json.Marshal(message.CustomSmsRequest.TemplateParams)
		body += "&templateParas=" + url.QueryEscape(string(templateParams))
	}

	body += "&statusCallback=" + url.QueryEscape(statusCallBack)
	body += "&signature=" + url.QueryEscape(configuration.CustomSmsConfiguration.Huawei.Signature)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	// Creating Request URL
	reqUrl := url.URL{
		Scheme: "https",
		Host:   configs.HuaweiApiAddress,
		Path:   "/sms/batchSendSms/v1",
	}

	req, err := http.NewRequest("POST", reqUrl.String(), bytes.NewBuffer([]byte(body)))
	if err != nil {
		Log.Error(err.Error())
		return CreateError(codes.Internal, 100318, map[string]string{"error": err.Error()})
	}
	for key, header := range headers {
		req.Header.Set(key, header)
	}

	resp, err := client.Do(req)
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			Log.Error(err.Error())
		}
	}(resp.Body)

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		Log.Error(err.Error())
		return CreateError(codes.Internal, 100319, map[string]string{"error": err.Error()})
	}

	// Parse Response Body
	var data map[string]interface{}
	err = json.Unmarshal(respBody, &data)
	if err != nil {
		Log.Error(err.Error())
		return CreateError(codes.Internal, 1000, map[string]string{"error": err.Error()})
	}
	if data["code"] != nil && data["code"].(string) != "000000" {
		Log.Error(data)
		return CreateError(codes.InvalidArgument, 100333, map[string]string{"error": data["description"].(string)})
	} else {
		return nil
	}
}

func sendSmsV2(ctx context.Context, message *Message, configuration *Configuration) ([]string, error) {
	var failedTargets []string
	const AUTH_HEADER_VALUE = "WSSE realm=\"SDP\",profile=\"UsernameToken\",type=\"Appkey\""

	statusCallBack := ""
	headers := map[string]string{
		"Content-Type":  "application/x-www-form-urlencoded",
		"Authorization": AUTH_HEADER_VALUE,
		"X-WSSE":        buildWsseHeader(configs.HuaweiAppKey, configs.HuaweiAppSecret),
	}
	var receivers string
	for _, receiver := range message.Target {
		match, err := regexp.Match(phoneRegex, []byte(receiver))
		if err != nil {
			Log.Error(err.Error())
			continue
		}
		if match {
			receivers += receiver + ","
		} else {
			if userInfo, err := getUserInfoV2(ctx, receiver, GetContactInfoSelector); err != nil {
				Log.Error(err.Error())
				failedTargets = append(failedTargets, receiver)
			} else {
				receivers += userInfo.SensitiveInfo.GetPhone() + ","
			}
		}
	}

	body := "from=" + url.QueryEscape(configuration.CustomSmsConfiguration.Huawei.Channel) + "&to=" + url.QueryEscape(receivers) + "&templateId=" + url.QueryEscape(configuration.CustomSmsConfiguration.Huawei.TemplateId)

	if len(message.CustomSmsRequest.TemplateParams) > 0 {
		templateParams, _ := json.Marshal(message.CustomSmsRequest.TemplateParams)
		body += "&templateParas=" + url.QueryEscape(string(templateParams))
	}

	body += "&statusCallback=" + url.QueryEscape(statusCallBack)
	body += "&signature=" + url.QueryEscape(configuration.CustomSmsConfiguration.Huawei.Signature)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	// Creating Request URL
	reqUrl := url.URL{
		Scheme: "https",
		Host:   configs.HuaweiApiAddress,
		Path:   "/sms/batchSendSms/v1",
	}

	req, err := http.NewRequest("POST", reqUrl.String(), bytes.NewBuffer([]byte(body)))
	if err != nil {
		Log.Error(err.Error())
		return failedTargets, CreateError(codes.Internal, 100318, map[string]string{"error": err.Error()})
	}
	for key, header := range headers {
		req.Header.Set(key, header)
	}

	resp, err := client.Do(req)
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			Log.Error(err.Error())
		}
	}(resp.Body)

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		Log.Error(err.Error())
		return failedTargets, CreateError(codes.Internal, 100319, map[string]string{"error": err.Error()})
	}

	// Parse Response Body
	var data map[string]interface{}
	err = json.Unmarshal(respBody, &data)
	if err != nil {
		Log.Error(err.Error())
		return failedTargets, CreateError(codes.Internal, 1000, map[string]string{"error": err.Error()})
	}
	if data["code"] != nil && data["code"].(string) != "000000" {
		// split receivers
		failedTargets = append(failedTargets, strings.Split(receivers, ",")...)
		return failedTargets, CreateError(codes.InvalidArgument, 100333, map[string]string{"error": data["description"].(string)})
	} else {
		return failedTargets, nil
	}
}

func sendFeishuMessage(ctx context.Context, message *Message, configuration *Configuration) error {
	appId, err := AES_DecryptBase64(configuration.CustomFeishuConfiguration.AppId, []byte(configs.SecretKey))
	if err != nil {
		return err
	}

	var feishuConfigData models.FeishuConfigData
	val, err := database.RedisGetString(redisFeishuApplicationsKey + string(appId))
	if err != nil {
		Log.Error(err.Error())
		return CreateError(codes.Internal, 1001, map[string]string{"error": err.Error()})
	} else {
		err = json.Unmarshal([]byte(*val), &feishuConfigData)
		if err != nil {
			Log.Error(err.Error())
			return CreateError(codes.Internal, 1000, map[string]string{"error": err.Error()})
		}
	}

	var tenantAccessToken string
	companyId := message.CustomFeishuRequest.CompanyId

	getFeishuTokenResp, err := pb.ThirdAdapterService.GetFeiShuToken(context.Background(), &pb.FeishuTokenReq{
		SuiteId: feishuConfigData.AppId,
		CorpId:  companyId,
	})
	if err != nil {
		Log.Error(err.Error())
		return CreateError(codes.Internal, 100013, map[string]string{"error": err.Error()})
	} else {
		tenantAccessToken = getFeishuTokenResp.GetCorpToken()
	}

	if companyId == "" {
		feishuConfigData.TenantAccessToken = tenantAccessToken
	} else {
		if feishuConfigData.CompanyData == nil {
			feishuConfigData.CompanyData = make(map[string]models.FeishuCompany)
		}
		if companyData, ok := feishuConfigData.CompanyData[companyId]; ok {
			companyData.TenantAccessToken = tenantAccessToken
		} else {
			feishuConfigData.CompanyData[companyId] = models.FeishuCompany{
				TenantAccessToken: tenantAccessToken,
			}
		}
	}

	binary, err := MarshalBinary(feishuConfigData)
	if err != nil {
		Log.Error(err.Error())
		return CreateError(codes.Internal, 1000, map[string]string{"error": err.Error()})
	} else {
		err := database.RedisSet(redisFeishuApplicationsKey+string(appId), binary, &infiniteDuration)
		if err != nil {
			Log.Error(err.Error())
			return CreateError(codes.Internal, 1001, map[string]string{"error": err.Error()})
		}
	}

	for _, target := range message.Target {
		var receiveId string
		if message.CustomFeishuRequest.ReceiveIdType == "union_id" {
			user, err := pb.UserService.GetUser(ctx, &pb.GetUserRequest{OpenId: &target})
			if err != nil {
				Log.Error(err.Error())
				continue
			} else {
				accountInfos := user.AccountInfos
				for _, accountInfo := range accountInfos {
					if accountInfo.GetSource() == "feishu" {
						receiveId = accountInfo.GetSourceId()
						break
					}
				}
			}
		} else {
			receiveId = target
		}

		func() {
			body, _ := json.Marshal(
				gin.H{
					"receive_id": receiveId,
					"msg_type":   message.CustomFeishuRequest.MsgType,
					"content":    message.Content,
				})
			requestBody := bytes.NewBuffer(body)
			client := &http.Client{}
			req, err := http.NewRequest(
				"POST",
				"https://open.feishu.cn/open-apis/im/v1/messages?receive_id_type="+message.CustomFeishuRequest.ReceiveIdType,
				requestBody,
			)

			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", tenantAccessToken)

			resp, err := client.Do(req)
			if err != nil {
				Log.Error(err.Error())
				return
			}
			defer func(Body io.ReadCloser) {
				err := Body.Close()
				if err != nil {
					Log.Fatal(err)
				}
			}(resp.Body)
			//Read the response body
			resBody, err := io.ReadAll(resp.Body)
			if err != nil {
				Log.Error(err.Error())
				return
			}
			var data map[string]interface{}
			err = json.Unmarshal(resBody, &data)
			if err != nil {
				return
			}

			if data["code"] != nil && data["code"].(float64) != 0 {
				Log.Error(data)
				return
			} else {
				Log.Info(data)
				return
			}
		}()
	}
	return nil
}

func sendFeishuMessageV2(ctx context.Context, message *Message, configuration *Configuration) ([]string, error) {
	appId, err := AES_DecryptBase64(configuration.CustomFeishuConfiguration.AppId, []byte(configs.SecretKey))
	if err != nil {
		return []string{}, err
	}
	var failedTargets []string
	var tenantIds = make(map[string]string)

	for _, target := range message.Target {
		var receiveId string
		var tenantId string
		user, err := getUserInfoV2(ctx, target, GetApplicationInfoSelector)
		if err != nil {
			Log.Error(err.Error())
			failedTargets = append(failedTargets, target)
			continue
		} else {
			accountInfos := user.AccountInfos
			for _, accountInfo := range accountInfos {
				if accountInfo.GetSource() == "feishu" {
					receiveId = accountInfo.GetSourceId()
					break
				}
			}
		}
		tenantId = user.GetTenantId()
		if _, ok := tenantIds[tenantId]; !ok {
			getFeishuTokenResp, err := pb.ThirdAdapterService.GetFeiShuToken(context.Background(), &pb.FeishuTokenReq{
				SuiteId: string(appId),
				CorpId:  tenantId,
			})
			if err != nil {
				Log.Error(err.Error())
				failedTargets = append(failedTargets, target)
				continue
			} else {
				tenantIds[tenantId] = getFeishuTokenResp.GetCorpToken()
			}
		}

		func() {
			body, _ := json.Marshal(
				gin.H{
					"receive_id": receiveId,
					"msg_type":   message.CustomFeishuRequest.MsgType,
					"content":    message.Content,
				})
			requestBody := bytes.NewBuffer(body)
			client := &http.Client{}
			req, err := http.NewRequest(
				"POST",
				"https://open.feishu.cn/open-apis/im/v1/messages?receive_id_type="+message.CustomFeishuRequest.ReceiveIdType,
				requestBody,
			)

			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", tenantIds[tenantId])

			resp, err := client.Do(req)
			if err != nil {
				Log.Error(err.Error())
				return
			}
			defer func(Body io.ReadCloser) {
				err := Body.Close()
				if err != nil {
					Log.Fatal(err)
				}
			}(resp.Body)
			//Read the response body
			resBody, err := io.ReadAll(resp.Body)
			if err != nil {
				Log.Error(err.Error())
				return
			}
			var data map[string]interface{}
			err = json.Unmarshal(resBody, &data)
			if err != nil {
				return
			}

			if data["code"] != nil && data["code"].(float64) != 0 {
				Log.Error(data)
				failedTargets = append(failedTargets, target)
				return
			} else {
				Log.Info(data)
				return
			}
		}()
	}

	return failedTargets, nil
}

func sendDingTalkMessage(ctx context.Context, message *Message, configuration *Configuration) error {
	var dingTalkConfigData models.DingTalkConfigData
	val, err := database.RedisGetString(redisDingTalkApplicationsKey + configuration.CustomDingTalkConfiguration.AppId)
	if err != nil {
		Log.Error(err.Error())
		return CreateError(codes.Internal, 1001, map[string]string{"error": err.Error()})
	} else {
		err = json.Unmarshal([]byte(*val), &dingTalkConfigData)
		if err != nil {
			Log.Error(err.Error())
			return CreateError(codes.Internal, 1000, map[string]string{"error": err.Error()})
		}
	}
	// Create Query
	var accessToken string
	var agentId string
	companyId := message.CustomDingtalkRequest.CompanyId

	if companyId == "" {
		accessToken = dingTalkConfigData.AccessToken
		agentId = dingTalkConfigData.AgentId
	} else {
		if dingTalkConfigData.CompanyData == nil {
			dingTalkConfigData.CompanyData = make(map[string]models.DingTalkCompany)
		}
		if companyData, ok := dingTalkConfigData.CompanyData[companyId]; ok {
			accessToken = companyData.AccessToken
		} else {
			getDingtalkTokenResp, err := pb.ThirdAdapterService.GetDingtalkToken(context.Background(), &pb.DingtalkTokenReq{
				SuiteId: dingTalkConfigData.SuiteId,
				CorpId:  companyId,
			})
			if err != nil {
				Log.Error(err.Error())
				return CreateError(codes.Internal, 100013, map[string]string{"error": err.Error()})
			} else {
				accessToken = getDingtalkTokenResp.GetCorpToken()
				dingTalkConfigData.CompanyData[companyId] = models.DingTalkCompany{
					AccessToken: accessToken,
				}
				binary, err := MarshalBinary(dingTalkConfigData)
				if err != nil {
					Log.Error(err.Error())
					return CreateError(codes.Internal, 1000, map[string]string{"error": err.Error()})
				} else {
					err := database.RedisSet(redisDingTalkApplicationsKey+configuration.CustomDingTalkConfiguration.AppId, binary, &infiniteDuration)
					if err != nil {
						Log.Error(err.Error())
						return CreateError(codes.Internal, 1001, map[string]string{"error": err.Error()})
					}
				}
			}
		}
		if dingTalkConfigData.CompanyData[companyId].AgentId == "" {
			agentIdResp, err := GetDingTalkAuthInfo(dingTalkConfigData.SuiteKey, dingTalkConfigData.SuiteSecret, dingTalkConfigData.SuiteTicket, companyId)
			if err != nil {
				return err
			} else {
				companyData := dingTalkConfigData.CompanyData[companyId]
				companyData.AgentId = agentIdResp
				dingTalkConfigData.CompanyData[companyId] = companyData
				binary, err := MarshalBinary(dingTalkConfigData)
				if err != nil {
					Log.Error(err.Error())
					return CreateError(codes.Internal, 1000, map[string]string{"error": err.Error()})
				} else {
					err := database.RedisSet(redisDingTalkApplicationsKey+configuration.CustomDingTalkConfiguration.AppId, binary, &infiniteDuration)
					if err != nil {
						Log.Error(err.Error())
						return CreateError(codes.Internal, 1001, map[string]string{"error": err.Error()})
					} else {
						agentId = dingTalkConfigData.CompanyData[companyId].AgentId
					}
				}
			}
		} else {
			agentId = dingTalkConfigData.CompanyData[companyId].AgentId
		}
	}

	query := url.Values{}
	query.Add("access_token", accessToken)
	// Create Url
	reqUrl := url.URL{
		Scheme:   "https",
		Host:     "oapi.dingtalk.com",
		Path:     "/topapi/message/corpconversation/sendbytemplate",
		RawQuery: query.Encode(),
	}

	Log.Info(reqUrl.String())

	var useridList string

	// split target and separate by comma
	for i, target := range message.Target {
		var id string
		var unionId string
		user, err := pb.UserService.GetUser(ctx, &pb.GetUserRequest{OpenId: &target})
		if err != nil {
			Log.Error(err.Error())
			return err
		} else {
			accountInfos := user.AccountInfos
			for _, accountInfo := range accountInfos {
				if accountInfo.GetSource() == "dingtalk" {
					unionId = accountInfo.GetSourceId()
					break
				}
			}
		}

		userId, err := GetDingTalkUserIdByUnionId(accessToken, unionId)
		if err != nil {
			Log.Error(err.Error())
		} else {
			id = userId
		}

		if i == len(message.Target)-1 {
			useridList += id
			break
		} else {
			useridList += id + ","
		}
	}

	// Create Request Body
	reqBody := gin.H{
		"agent_id":    agentId,
		"template_id": configuration.CustomDingTalkConfiguration.TemplateId,
		"userid_list": useridList,
		"data":        message.Content,
	}

	Log.Info(reqBody)

	// Create Request Body
	body, err := json.Marshal(reqBody)
	if err != nil {
		Log.Error(err.Error())
		return CreateError(codes.Internal, 100319, map[string]string{"error": err.Error()})
	}

	// Create Request
	req := &http.Request{
		Method: "POST",
		URL:    &reqUrl,
		Body:   io.NopCloser(bytes.NewReader(body)),
	}

	// Create Client
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		Log.Error(err.Error())
		return CreateError(codes.Internal, 1000, map[string]string{"error": err.Error()})
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			Log.Fatal(err)
		}
	}(resp.Body)

	// Read Response Body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		Log.Error(err.Error())
		return CreateError(codes.Internal, 1000, map[string]string{"error": err.Error()})
	}

	// Parse Response Body
	var data map[string]interface{}
	err = json.Unmarshal(respBody, &data)
	if err != nil {
		Log.Error(err.Error())
		return CreateError(codes.Internal, 1000, map[string]string{"error": err.Error()})
	}

	Log.Info(data)

	if data["errcode"].(float64) != 0 {
		if data["errcode"].(float64) == 400002 {
			agentIdResp, err := GetDingTalkAuthInfo(dingTalkConfigData.SuiteKey, dingTalkConfigData.SuiteSecret, dingTalkConfigData.SuiteTicket, companyId)
			if err != nil {
				return err
			} else {
				companyData := dingTalkConfigData.CompanyData[companyId]
				companyData.AgentId = agentIdResp
				dingTalkConfigData.CompanyData[companyId] = companyData
				binary, err := MarshalBinary(dingTalkConfigData)
				if err != nil {
					Log.Error(err.Error())
					return CreateError(codes.Internal, 1000, map[string]string{"error": err.Error()})
				} else {
					err := database.RedisSet(redisDingTalkApplicationsKey+configuration.CustomDingTalkConfiguration.AppId, binary, &infiniteDuration)
					if err != nil {
						Log.Error(err.Error())
						return CreateError(codes.Internal, 1001, map[string]string{"error": err.Error()})
					} else {
						agentId = dingTalkConfigData.CompanyData[companyId].AgentId
						reqBody["agent_id"] = agentId
					}
				}
			}
		}
		Log.Error(data["errmsg"].(string))
		var corpToken string
		getDingtalkTokenResp, err := pb.ThirdAdapterService.GetDingtalkToken(context.Background(), &pb.DingtalkTokenReq{
			SuiteId: dingTalkConfigData.SuiteId,
			CorpId:  companyId,
		})
		corpToken = getDingtalkTokenResp.GetCorpToken()
		if err != nil {
			Log.Error(err.Error())
			return CreateError(codes.Internal, 100013, map[string]string{"error": err.Error()})
		} else {
			if companyId == "" {
				corpToken = getDingtalkTokenResp.GetCorpToken()
				dingTalkConfigData.AccessToken = corpToken
			} else {
				companyData := dingTalkConfigData.CompanyData[companyId]
				companyData.AccessToken = getDingtalkTokenResp.GetCorpToken()
				dingTalkConfigData.CompanyData[companyId] = companyData
			}
			binary, err := MarshalBinary(dingTalkConfigData)
			if err != nil {
				Log.Error(err.Error())
				return CreateError(codes.Internal, 1000, map[string]string{"error": err.Error()})
			} else {
				err := database.RedisSet(redisDingTalkApplicationsKey+configuration.CustomDingTalkConfiguration.AppId, binary, &infiniteDuration)
				if err != nil {
					Log.Error(err.Error())
					return CreateError(codes.Internal, 1001, map[string]string{"error": err.Error()})
				}
			}
		}
		var useridList string
		// split target and separate by comma
		for i, target := range message.Target {
			var id string
			var unionId string
			user, err := pb.UserService.GetUser(ctx, &pb.GetUserRequest{OpenId: &target})
			if err != nil {
				Log.Error(err.Error())
				return err
			} else {
				accountInfos := user.AccountInfos
				for _, accountInfo := range accountInfos {
					if accountInfo.GetSource() == "dingtalk" {
						unionId = accountInfo.GetSourceId()
						break
					}
				}
			}

			userId, err := GetDingTalkUserIdByUnionId(corpToken, unionId)
			if err != nil {
				Log.Error(err.Error())
			} else {
				id = userId
			}

			if i == len(message.Target)-1 {
				useridList += id
				break
			} else {
				useridList += id + ","
			}
		}

		reqBody["userid_list"] = useridList

		query.Set("access_token", corpToken)
		// Create Url
		reqUrl = url.URL{
			Scheme:   "https",
			Host:     "oapi.dingtalk.com",
			Path:     "/topapi/message/corpconversation/sendbytemplate",
			RawQuery: query.Encode(),
		}

		// Create Request Body
		body, err := json.Marshal(reqBody)
		if err != nil {
			Log.Error(err.Error())
			return CreateError(codes.Internal, 100319, map[string]string{"error": err.Error()})
		}

		// Create Request
		req := &http.Request{
			Method: "POST",
			URL:    &reqUrl,
			Body:   io.NopCloser(bytes.NewReader(body)),
		}

		// Create Client
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			Log.Error(err.Error())
			return CreateError(codes.Internal, 1000, map[string]string{"error": err.Error()})
		}
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				Log.Fatal(err)
			}
		}(resp.Body)

		// Read Response Body
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			Log.Error(err.Error())
			return CreateError(codes.Internal, 1000, map[string]string{"error": err.Error()})
		}

		// Parse Response Body
		var data map[string]interface{}
		err = json.Unmarshal(respBody, &data)
		if err != nil {
			Log.Error(err.Error())
			return CreateError(codes.Internal, 1000, map[string]string{"error": err.Error()})
		}

		if data["errcode"].(float64) != 0 {
			Log.Error(data["errmsg"].(string))
			return CreateError(codes.Internal, 100328, map[string]string{"error": data["errmsg"].(string)})
		} else {
			Log.Info(data)
			return nil
		}
	} else {
		Log.Info(data)
		return nil
	}
}

func sendDingTalkMessageV2(ctx context.Context, message *Message, configuration *Configuration) ([]string, error) {
	var failedTargets []string
	var dingTalkConfigData models.DingTalkConfigData
	val, err := database.RedisGetString(redisDingTalkApplicationsKey + configuration.CustomDingTalkConfiguration.AppId)
	if err != nil {
		Log.Error(err.Error())
		return failedTargets, CreateError(codes.Internal, 1001, map[string]string{"error": err.Error()})
	} else {
		err = json.Unmarshal([]byte(*val), &dingTalkConfigData)
		if err != nil {
			Log.Error(err.Error())
			return failedTargets, CreateError(codes.Internal, 1000, map[string]string{"error": err.Error()})
		}
	}
	type Tenant struct {
		Ids         []string
		AccessToken string
		AgentId     string
	}

	var tenants = make(map[string]Tenant)

	// split target and separate by comma
	for _, target := range message.Target {
		var unionId string
		var tenantId string
		userInfo, err := getUserInfoV2(ctx, target, GetApplicationInfoSelector)
		if err != nil {
			Log.Error(err.Error())
			failedTargets = append(failedTargets, target)
			continue
		} else {
			accountInfos := userInfo.GetAccountInfos()
			for _, accountInfo := range accountInfos {
				if accountInfo.GetSource() == "dingtalk" {
					unionId = accountInfo.GetSourceId()
					break
				}
			}
		}

		tenantId = userInfo.GetTenantId()
		if _, ok := tenants[tenantId]; !ok {
			getDingtalkTokenResp, err := pb.ThirdAdapterService.GetDingtalkToken(context.Background(), &pb.DingtalkTokenReq{
				SuiteId: dingTalkConfigData.SuiteId,
				CorpId:  tenantId,
			})
			if err != nil {
				Log.Error(err.Error())
				return failedTargets, CreateError(codes.Internal, 100013, map[string]string{"error": err.Error()})
			} else {
				tenants[tenantId] = Tenant{
					AccessToken: getDingtalkTokenResp.GetCorpToken(),
				}
			}
		}
		tenant := tenants[tenantId]
		userId, err := GetDingTalkUserIdByUnionId(tenant.AccessToken, unionId)
		if err != nil {
			Log.Error(err.Error())
			failedTargets = append(failedTargets, target)
			continue
		} else {
			if tenants[tenantId].Ids == nil {
				Ids := []string{userId}
				tenant.Ids = Ids
				tenants[tenantId] = tenant
			} else {
				tenant.Ids = append(tenants[tenantId].Ids, userId)
				tenants[tenantId] = tenant
			}
		}
	}

	for tenantId, tenant := range tenants {
		func() {
			query := url.Values{}
			query.Add("access_token", tenant.AccessToken)
			// Create Url
			reqUrl := url.URL{
				Scheme:   "https",
				Host:     "oapi.dingtalk.com",
				Path:     "/topapi/message/corpconversation/sendbytemplate",
				RawQuery: query.Encode(),
			}

			var useridList string

			// split target and separate by comma
			for i, id := range tenant.Ids {
				if i == len(tenant.Ids)-1 {
					useridList += id
					break
				} else {
					useridList += id + ","
				}
			}
			agentId := dingTalkConfigData.CompanyData[tenantId].AgentId

			// Create Request Body
			reqBody := gin.H{
				"agent_id":    agentId,
				"template_id": configuration.CustomDingTalkConfiguration.TemplateId,
				"userid_list": useridList,
				"data":        message.Content,
			}

			// Create Request Body
			body, err := json.Marshal(reqBody)
			if err != nil {
				Log.Error(err.Error())
				failedTargets = append(failedTargets, tenant.Ids...)
				return
			}

			// Create Request
			req := &http.Request{
				Method: "POST",
				URL:    &reqUrl,
				Body:   io.NopCloser(bytes.NewReader(body)),
			}

			// Create Client
			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				Log.Error(err.Error())
				failedTargets = append(failedTargets, tenant.Ids...)
				return
			}
			defer func(Body io.ReadCloser) {
				err := Body.Close()
				if err != nil {
					Log.Fatal(err)
				}
			}(resp.Body)

			// Read Response Body
			respBody, err := io.ReadAll(resp.Body)
			if err != nil {
				Log.Error(err.Error())
				failedTargets = append(failedTargets, tenant.Ids...)
				return
			}

			// Parse Response Body
			var data map[string]interface{}
			err = json.Unmarshal(respBody, &data)
			if err != nil {
				Log.Error(err.Error())
				failedTargets = append(failedTargets, tenant.Ids...)
				return
			}

			if data["errcode"].(float64) != 0 {
				if data["errcode"].(float64) == 400002 {
					agentIdResp, err := GetDingTalkAuthInfo(dingTalkConfigData.SuiteKey, dingTalkConfigData.SuiteSecret, dingTalkConfigData.SuiteTicket, tenantId)
					if err != nil {
						failedTargets = append(failedTargets, tenant.Ids...)
						return
					} else {
						companyData := dingTalkConfigData.CompanyData[tenantId]
						companyData.AgentId = agentIdResp
						dingTalkConfigData.CompanyData[tenantId] = companyData
						binary, err := MarshalBinary(dingTalkConfigData)
						if err != nil {
							Log.Error(err.Error())
							failedTargets = append(failedTargets, tenant.Ids...)
							return
						} else {
							err := database.RedisSet(redisDingTalkApplicationsKey+configuration.CustomDingTalkConfiguration.AppId, binary, &infiniteDuration)
							if err != nil {
								Log.Error(err.Error())
								failedTargets = append(failedTargets, tenant.Ids...)
								return
							} else {
								agentId = dingTalkConfigData.CompanyData[tenantId].AgentId
								reqBody["agent_id"] = agentId
							}
						}
					}
				}

				// Create Url
				reqUrl = url.URL{
					Scheme:   "https",
					Host:     "oapi.dingtalk.com",
					Path:     "/topapi/message/corpconversation/sendbytemplate",
					RawQuery: query.Encode(),
				}

				// Create Request Body
				body, err := json.Marshal(reqBody)
				if err != nil {
					Log.Error(err.Error())
					failedTargets = append(failedTargets, tenant.Ids...)
					return
				}

				// Create Request
				req := &http.Request{
					Method: "POST",
					URL:    &reqUrl,
					Body:   io.NopCloser(bytes.NewReader(body)),
				}

				// Create Client
				client := &http.Client{}
				resp, err := client.Do(req)
				if err != nil {
					Log.Error(err.Error())
					failedTargets = append(failedTargets, tenant.Ids...)
					return
				}
				defer func(Body io.ReadCloser) {
					err := Body.Close()
					if err != nil {
						Log.Fatal(err)
					}
				}(resp.Body)

				// Read Response Body
				respBody, err := io.ReadAll(resp.Body)
				if err != nil {
					Log.Error(err.Error())
					failedTargets = append(failedTargets, tenant.Ids...)
					return
				}

				// Parse Response Body
				var data map[string]interface{}
				err = json.Unmarshal(respBody, &data)
				if err != nil {
					Log.Error(err.Error())
					failedTargets = append(failedTargets, tenant.Ids...)
					return
				}

				if data["errcode"].(float64) != 0 {
					Log.Error(data["errmsg"].(string))
					failedTargets = append(failedTargets, tenant.Ids...)
					return
				} else {
					Log.Info(data)
					return
				}
			} else {
				Log.Info(data)
				return
			}
		}()
	}
	return failedTargets, nil
}

func sendListeningMessage(ctx context.Context, message *Message, configuration *Configuration) error {
	// Checking if configuration type is linking
	if configuration.Type == "linkage" {
		linkageMessage := *message
		linkageMessage.Id = primitive.NewObjectID()
		linkageMessage.LinkageId = message.Id
		linkageMessage.OpenId = message.Target[0]
		message.LinkageId = linkageMessage.Id

		redirectMessageToSockets(&linkageMessage, configuration)
		// Returning error and Logging if failed to insert message to database
		if err := insertOneLinkingMessage(ctx, &linkageMessage, configuration); err != nil {
			return err
		}
	} else {
		redirectMessageToSockets(message, configuration)
	}
	return nil
}

func sendThirdPartyMessage(ctx context.Context, message *Message, configuration *Configuration) error {
	// Feishu Logic
	feishuAppId, err := AES_DecryptBase64(configuration.CustomFeishuConfiguration.AppId, []byte(configs.SecretKey))
	if err != nil {
		return err
	}

	var feishuConfigData models.FeishuConfigData
	val, err := database.RedisGetString(redisFeishuApplicationsKey + string(feishuAppId))
	if err != nil {
		Log.Error(err.Error())
		return CreateError(codes.Internal, 1001, map[string]string{"error": err.Error()})
	} else {
		err = json.Unmarshal([]byte(*val), &feishuConfigData)
		if err != nil {
			Log.Error(err.Error())
			return CreateError(codes.Internal, 1000, map[string]string{"error": err.Error()})
		}
	}

	// Create Query
	var tenantAccessToken string
	feishuCompanyId := message.CustomFeishuRequest.CompanyId

	if feishuCompanyId == "" {
		feishuCompanyId = feishuConfigData.TenantKey
		tenantAccessToken = feishuConfigData.TenantAccessToken
	} else {
		if feishuConfigData.CompanyData == nil {
			feishuConfigData.CompanyData = make(map[string]models.FeishuCompany)
		}
		if companyData, ok := feishuConfigData.CompanyData[feishuCompanyId]; ok {
			tenantAccessToken = companyData.TenantAccessToken
		} else {
			getFeishuTokenResp, err := pb.ThirdAdapterService.GetFeiShuToken(context.Background(), &pb.FeishuTokenReq{
				SuiteId: feishuConfigData.AppId,
				CorpId:  feishuCompanyId,
			})
			if err != nil {
				Log.Error(err.Error())
				return CreateError(codes.Internal, 100013, map[string]string{"error": err.Error()})
			} else {
				tenantAccessToken = getFeishuTokenResp.GetCorpToken()
				feishuConfigData.CompanyData[feishuCompanyId] = models.FeishuCompany{
					TenantAccessToken: tenantAccessToken,
				}
				binary, err := MarshalBinary(feishuConfigData)
				if err != nil {
					Log.Error(err.Error())
					return CreateError(codes.Internal, 1000, map[string]string{"error": err.Error()})
				} else {
					err := database.RedisSet(redisFeishuApplicationsKey+string(feishuAppId), binary, &infiniteDuration)
					if err != nil {
						Log.Error(err.Error())
						return CreateError(codes.Internal, 1001, map[string]string{"error": err.Error()})
					}
				}
			}
		}
	}

	// DingtalkLog
	var dingTalkConfigData models.DingTalkConfigData
	val, err = database.RedisGetString(redisDingTalkApplicationsKey + configuration.CustomDingTalkConfiguration.AppId)
	if err != nil {
		Log.Error(err.Error())
		return CreateError(codes.Internal, 1001, map[string]string{"error": err.Error()})
	} else {
		err = json.Unmarshal([]byte(*val), &dingTalkConfigData)
		if err != nil {
			Log.Error(err.Error())
			return CreateError(codes.Internal, 1000, map[string]string{"error": err.Error()})
		}
	}

	// Create Query
	var dingtalkAccessToken string
	var agentId string
	dingtalkCompanyId := message.CustomDingtalkRequest.CompanyId

	if dingtalkCompanyId == "" {
		dingtalkAccessToken = dingTalkConfigData.AccessToken
		agentId = dingTalkConfigData.AgentId
	} else {
		if dingTalkConfigData.CompanyData == nil {
			dingTalkConfigData.CompanyData = make(map[string]models.DingTalkCompany)
		}
		if companyData, ok := dingTalkConfigData.CompanyData[dingtalkCompanyId]; ok {
			dingtalkAccessToken = companyData.AccessToken
		} else {
			getDingtalkTokenResp, err := pb.ThirdAdapterService.GetDingtalkToken(context.Background(), &pb.DingtalkTokenReq{
				SuiteId: dingTalkConfigData.SuiteId,
				CorpId:  dingtalkCompanyId,
			})
			if err != nil {
				Log.Error(err.Error())
				return CreateError(codes.Internal, 100013, map[string]string{"error": err.Error()})
			} else {
				dingtalkAccessToken = getDingtalkTokenResp.GetCorpToken()
				dingTalkConfigData.CompanyData[dingtalkCompanyId] = models.DingTalkCompany{
					AccessToken: dingtalkAccessToken,
				}
				binary, err := MarshalBinary(dingTalkConfigData)
				if err != nil {
					Log.Error(err.Error())
					return CreateError(codes.Internal, 1000, map[string]string{"error": err.Error()})
				} else {
					err := database.RedisSet(redisDingTalkApplicationsKey+configuration.CustomDingTalkConfiguration.AppId, binary, &infiniteDuration)
					if err != nil {
						Log.Error(err.Error())
						return CreateError(codes.Internal, 1001, map[string]string{"error": err.Error()})
					}
				}
			}
		}
		if dingTalkConfigData.CompanyData[dingtalkCompanyId].AgentId == "" {
			agentIdResp, err := GetDingTalkAuthInfo(dingTalkConfigData.SuiteKey, dingTalkConfigData.SuiteSecret, dingTalkConfigData.SuiteTicket, dingtalkCompanyId)
			if err != nil {
				return err
			} else {
				companyData := dingTalkConfigData.CompanyData[dingtalkCompanyId]
				companyData.AgentId = agentIdResp
				dingTalkConfigData.CompanyData[dingtalkCompanyId] = companyData
				binary, err := MarshalBinary(dingTalkConfigData)
				if err != nil {
					Log.Error(err.Error())
					return CreateError(codes.Internal, 1000, map[string]string{"error": err.Error()})
				} else {
					err := database.RedisSet(redisDingTalkApplicationsKey+configuration.CustomDingTalkConfiguration.AppId, binary, &infiniteDuration)
					if err != nil {
						Log.Error(err.Error())
						return CreateError(codes.Internal, 1001, map[string]string{"error": err.Error()})
					} else {
						agentId = dingTalkConfigData.CompanyData[dingtalkCompanyId].AgentId
					}
				}
			}
		} else {
			agentId = dingTalkConfigData.CompanyData[dingtalkCompanyId].AgentId
		}
	}

	query := url.Values{}
	query.Add("access_token", dingtalkAccessToken)
	// Create Url
	reqUrl := url.URL{
		Scheme:   "https",
		Host:     "oapi.dingtalk.com",
		Path:     "/topapi/message/corpconversation/sendbytemplate",
		RawQuery: query.Encode(),
	}

	Log.Info(reqUrl.String())

	var feishuUsers []string
	var dingtalkUsers string
	var ziqian []string

	// Getting all users
	// split target and separate by comma
	for i, target := range message.Target {
		var id string
		var unionId string
		user, err := pb.UserService.GetUser(ctx, &pb.GetUserRequest{OpenId: &target})
		if err != nil {
			Log.Error(err.Error())
			return err
		} else {
			accountInfos := user.AccountInfos
			for _, accountInfo := range accountInfos {
				if accountInfo.GetSource() == "dingtalk" {
					unionId = accountInfo.GetSourceId()
					userId, err := GetDingTalkUserIdByUnionId(dingtalkAccessToken, unionId)
					if err != nil {
						Log.Error(err.Error())
						getDingtalkTokenResp, err := pb.ThirdAdapterService.GetDingtalkToken(context.Background(), &pb.DingtalkTokenReq{
							SuiteId: dingTalkConfigData.SuiteId,
							CorpId:  dingtalkCompanyId,
						})
						if err != nil {
							Log.Error(err.Error())
						} else {
							Log.Info(getDingtalkTokenResp.GetCorpToken())
							dingtalkAccessToken = getDingtalkTokenResp.GetCorpToken()
							userId, err := GetDingTalkUserIdByUnionId(dingtalkAccessToken, unionId)
							if err != nil {
								Log.Error(err.Error())
							} else {
								id = userId
								if dingtalkCompanyId == "" {
									dingtalkAccessToken = getDingtalkTokenResp.GetCorpToken()
									dingTalkConfigData.AccessToken = dingtalkAccessToken
								} else {
									companyData := dingTalkConfigData.CompanyData[dingtalkCompanyId]
									companyData.AccessToken = getDingtalkTokenResp.GetCorpToken()
									dingTalkConfigData.CompanyData[dingtalkCompanyId] = companyData
								}
								binary, err := MarshalBinary(dingTalkConfigData)
								if err != nil {
									Log.Error(err.Error())
								} else {
									err := database.RedisSet(redisDingTalkApplicationsKey+configuration.CustomDingTalkConfiguration.AppId, binary, &infiniteDuration)
									if err != nil {
										Log.Error(err.Error())
									}
								}
							}
						}
					} else {
						id = userId
					}

					if i == len(message.Target)-1 {
						dingtalkUsers += id
						break
					} else {
						dingtalkUsers += id + ","
					}
				} else if accountInfo.GetSource() == "feishu" {
					feishuUsers = append(feishuUsers, accountInfo.GetSourceId())
				} else if accountInfo.GetSource() == "jumeaux" {
					ziqian = append(ziqian, target)
				}
			}
		}
	}

	message.Target = ziqian

	// Redirect message to sockets for ziqian users
	Log.Info("Redirect message to sockets for ziqian users")
	redirectMessageToSockets(message, configuration)

	for _, receiveId := range feishuUsers {
		func() {
			body, _ := json.Marshal(
				gin.H{
					"receive_id": receiveId,
					"msg_type":   message.CustomFeishuRequest.MsgType,
					"content":    message.Content,
				})
			requestBody := bytes.NewBuffer(body)
			client := &http.Client{}
			req, err := http.NewRequest(
				"POST",
				"https://open.feishu.cn/open-apis/im/v1/messages?receive_id_type="+message.CustomFeishuRequest.ReceiveIdType,
				requestBody,
			)

			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", tenantAccessToken)

			resp, err := client.Do(req)
			if err != nil {
				Log.Error(err.Error())
				return
			}
			defer func(Body io.ReadCloser) {
				err := Body.Close()
				if err != nil {
					Log.Fatal(err)
				}
			}(resp.Body)
			//Read the response body
			resBody, err := io.ReadAll(resp.Body)
			if err != nil {
				Log.Error(err.Error())
				return
			}
			var data map[string]interface{}
			err = json.Unmarshal(resBody, &data)
			if err != nil {
				return
			}
			Log.Info(data)

			if data["code"] != nil && data["code"].(float64) != 0 {
				getFeishuTokenResp, err := pb.ThirdAdapterService.GetFeiShuToken(context.Background(), &pb.FeishuTokenReq{
					SuiteId: feishuConfigData.AppId,
					CorpId:  feishuCompanyId,
				})
				if err != nil {
					Log.Error(err.Error())
					return
				} else {
					Log.Info(getFeishuTokenResp)
					tenantAccessToken = getFeishuTokenResp.GetCorpToken()
					if feishuConfigData.CompanyData == nil {
						feishuConfigData.CompanyData = make(map[string]models.FeishuCompany)
					}
					companyData := feishuConfigData.CompanyData[feishuCompanyId]
					companyData.TenantAccessToken = tenantAccessToken
					feishuConfigData.CompanyData[feishuCompanyId] = companyData
					binary, err := MarshalBinary(feishuConfigData)
					if err != nil {
						Log.Error(err.Error())
						return
					} else {
						err := database.RedisSet(redisFeishuApplicationsKey+string(feishuAppId), binary, &infiniteDuration)
						if err != nil {
							Log.Error(err.Error())
							return
						}
					}
					req.Header.Set("Authorization", tenantAccessToken)
					resp, err := client.Do(req)
					if err != nil {
						Log.Error(err.Error())
						return
					}
					defer func(Body io.ReadCloser) {
						err := Body.Close()
						if err != nil {
							Log.Fatal(err)
						}
					}(resp.Body)
					//Read the response body
					resBody, err := io.ReadAll(resp.Body)
					if err != nil {
						Log.Error(err.Error())
						return
					}
					var data map[string]interface{}
					err = json.Unmarshal(resBody, &data)
					if err != nil {
						return
					}
					Log.Info(data)
					if data["code"] != nil && data["code"].(float64) != 0 {
						Log.Error(data)
						return
					} else {
						Log.Info(data)
					}
				}
			}
		}()
	}

	if dingtalkUsers != "" {
		// Create Request Body
		dingtalkReqBody := gin.H{
			"agent_id":    agentId,
			"template_id": configuration.CustomDingTalkConfiguration.TemplateId,
			"userid_list": dingtalkUsers,
			"data":        message.Content,
		}

		Log.Info(dingtalkReqBody)

		// Create Request Body
		dingtalkBody, err := json.Marshal(dingtalkReqBody)
		if err != nil {
			Log.Error(err.Error())
			return CreateError(codes.Internal, 100319, map[string]string{"error": err.Error()})
		}

		// Create Request
		dingtalkReq := &http.Request{
			Method: "POST",
			URL:    &reqUrl,
			Body:   io.NopCloser(bytes.NewReader(dingtalkBody)),
		}

		// Create Client
		client := &http.Client{}
		dingtalkResp, err := client.Do(dingtalkReq)
		if err != nil {
			Log.Error(err.Error())
			return CreateError(codes.Internal, 1000, map[string]string{"error": err.Error()})
		}
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				Log.Fatal(err)
			}
		}(dingtalkResp.Body)

		// Read Response Body
		respBody, err := io.ReadAll(dingtalkResp.Body)
		if err != nil {
			Log.Error(err.Error())
			return CreateError(codes.Internal, 1000, map[string]string{"error": err.Error()})
		}

		// Parse Response Body
		var data map[string]interface{}
		err = json.Unmarshal(respBody, &data)
		if err != nil {
			Log.Error(err.Error())
			return CreateError(codes.Internal, 1000, map[string]string{"error": err.Error()})
		}
		Log.Info(data)
	}

	return nil
}

func sendThirdPartyMessageV2(ctx context.Context, message *Message, configuration *Configuration) ([]string, error) {
	var failedTargets []string

	feishuAppId, err := AES_DecryptBase64(configuration.CustomFeishuConfiguration.AppId, []byte(configs.SecretKey))
	if err != nil {
		return []string{}, err
	}
	var feishuIds = make(map[string]string)
	var feishuTenantIds = make(map[string]string)

	var dingTalkConfigData models.DingTalkConfigData
	val, err := database.RedisGetString(redisDingTalkApplicationsKey + configuration.CustomDingTalkConfiguration.AppId)
	if err != nil {
		Log.Error(err.Error())
		return failedTargets, CreateError(codes.Internal, 1001, map[string]string{"error": err.Error()})
	} else {
		err = json.Unmarshal([]byte(*val), &dingTalkConfigData)
		if err != nil {
			Log.Error(err.Error())
			return failedTargets, CreateError(codes.Internal, 1000, map[string]string{"error": err.Error()})
		}
	}
	type Tenant struct {
		Ids         []string
		AccessToken string
		AgentId     string
	}

	var dingtalkTenants = make(map[string]Tenant)

	var ziqianIds = make([]string, 0)
	for _, target := range message.Target {
		userInfo, err := getUserInfoV2(ctx, target, GetApplicationInfoSelector)
		if err != nil {
			Log.Error(err.Error())
			failedTargets = append(failedTargets, target)
			continue
		} else {
			accountInfos := userInfo.AccountInfos
			for _, accountInfo := range accountInfos {
				if accountInfo.GetSource() == "feishu" {
					feishuIds[accountInfo.GetSourceId()] = userInfo.GetTenantId()
					if _, ok := feishuTenantIds[userInfo.GetTenantId()]; !ok {
						getFeishuTokenResp, err := pb.ThirdAdapterService.GetFeiShuToken(context.Background(), &pb.FeishuTokenReq{
							SuiteId: string(feishuAppId),
							CorpId:  userInfo.GetTenantId(),
						})
						if err != nil {
							Log.Error(err.Error())
							failedTargets = append(failedTargets, target)
							continue
						} else {
							feishuTenantIds[userInfo.GetTenantId()] = getFeishuTokenResp.GetCorpToken()
						}
					}
					break
				} else if accountInfo.GetSource() == "dingtalk" {
					if _, ok := dingtalkTenants[userInfo.GetTenantId()]; !ok {
						getDingtalkTokenResp, err := pb.ThirdAdapterService.GetDingtalkToken(context.Background(), &pb.DingtalkTokenReq{
							SuiteId: dingTalkConfigData.SuiteId,
							CorpId:  userInfo.GetTenantId(),
						})
						if err != nil {
							Log.Error(err.Error())
							return failedTargets, CreateError(codes.Internal, 100013, map[string]string{"error": err.Error()})
						} else {
							dingtalkTenants[userInfo.GetTenantId()] = Tenant{
								AccessToken: getDingtalkTokenResp.GetCorpToken(),
							}
						}
					}
					tenant := dingtalkTenants[userInfo.GetTenantId()]
					userId, err := GetDingTalkUserIdByUnionId(tenant.AccessToken, accountInfo.GetSourceId())
					if err != nil {
						Log.Error(err.Error())
						failedTargets = append(failedTargets, target)
						continue
					} else {
						if dingtalkTenants[userInfo.GetTenantId()].Ids == nil {
							Ids := []string{userId}
							tenant.Ids = Ids
							dingtalkTenants[userInfo.GetTenantId()] = tenant
						} else {
							tenant.Ids = append(dingtalkTenants[userInfo.GetTenantId()].Ids, userId)
							dingtalkTenants[userInfo.GetTenantId()] = tenant
						}
					}
					break
				} else if accountInfo.GetSource() == "jumeaux" {
					ziqianIds = append(ziqianIds, target)
					break
				}
			}
		}
	}

	for feishuId, feishuTenant := range feishuIds {
		func() {
			body, _ := json.Marshal(
				gin.H{
					"receive_id": feishuId,
					"msg_type":   message.CustomFeishuRequest.MsgType,
					"content":    message.Content,
				})
			requestBody := bytes.NewBuffer(body)
			client := &http.Client{}
			req, err := http.NewRequest(
				"POST",
				"https://open.feishu.cn/open-apis/im/v1/messages?receive_id_type="+message.CustomFeishuRequest.ReceiveIdType,
				requestBody,
			)

			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", feishuTenantIds[feishuTenant])

			resp, err := client.Do(req)
			if err != nil {
				Log.Error(err.Error())
				return
			}
			defer func(Body io.ReadCloser) {
				err := Body.Close()
				if err != nil {
					Log.Fatal(err)
				}
			}(resp.Body)
			//Read the response body
			resBody, err := io.ReadAll(resp.Body)
			if err != nil {
				Log.Error(err.Error())
				return
			}
			var data map[string]interface{}
			err = json.Unmarshal(resBody, &data)
			if err != nil {
				return
			}

			if data["code"] != nil && data["code"].(float64) != 0 {
				Log.Error(data)
				failedTargets = append(failedTargets, feishuId)
				return
			} else {
				Log.Info(data)
				return
			}
		}()
	}

	for dingtalkTenantId, dingtalkTenant := range dingtalkTenants {
		func() {
			query := url.Values{}
			query.Add("access_token", dingtalkTenant.AccessToken)
			// Create Url
			reqUrl := url.URL{
				Scheme:   "https",
				Host:     "oapi.dingtalk.com",
				Path:     "/topapi/message/corpconversation/sendbytemplate",
				RawQuery: query.Encode(),
			}

			var useridList string

			// split target and separate by comma
			for i, id := range dingtalkTenant.Ids {
				if i == len(dingtalkTenant.Ids)-1 {
					useridList += id
					break
				} else {
					useridList += id + ","
				}
			}
			agentId := dingTalkConfigData.CompanyData[dingtalkTenantId].AgentId

			// Create Request Body
			reqBody := gin.H{
				"agent_id":    agentId,
				"template_id": configuration.CustomDingTalkConfiguration.TemplateId,
				"userid_list": useridList,
				"data":        message.Content,
			}

			// Create Request Body
			body, err := json.Marshal(reqBody)
			if err != nil {
				Log.Error(err.Error())
				failedTargets = append(failedTargets, dingtalkTenant.Ids...)
				return
			}

			// Create Request
			req := &http.Request{
				Method: "POST",
				URL:    &reqUrl,
				Body:   io.NopCloser(bytes.NewReader(body)),
			}

			// Create Client
			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				Log.Error(err.Error())
				failedTargets = append(failedTargets, dingtalkTenant.Ids...)
				return
			}
			defer func(Body io.ReadCloser) {
				err := Body.Close()
				if err != nil {
					Log.Fatal(err)
				}
			}(resp.Body)

			// Read Response Body
			respBody, err := io.ReadAll(resp.Body)
			if err != nil {
				Log.Error(err.Error())
				failedTargets = append(failedTargets, dingtalkTenant.Ids...)
				return
			}

			// Parse Response Body
			var data map[string]interface{}
			err = json.Unmarshal(respBody, &data)
			if err != nil {
				Log.Error(err.Error())
				failedTargets = append(failedTargets, dingtalkTenant.Ids...)
				return
			}

			if data["errcode"].(float64) != 0 {
				if data["errcode"].(float64) == 400002 {
					agentIdResp, err := GetDingTalkAuthInfo(dingTalkConfigData.SuiteKey, dingTalkConfigData.SuiteSecret, dingTalkConfigData.SuiteTicket, dingtalkTenantId)
					if err != nil {
						failedTargets = append(failedTargets, dingtalkTenant.Ids...)
						return
					} else {
						companyData := dingTalkConfigData.CompanyData[dingtalkTenantId]
						companyData.AgentId = agentIdResp
						dingTalkConfigData.CompanyData[dingtalkTenantId] = companyData
						binary, err := MarshalBinary(dingTalkConfigData)
						if err != nil {
							Log.Error(err.Error())
							failedTargets = append(failedTargets, dingtalkTenant.Ids...)
							return
						} else {
							err := database.RedisSet(redisDingTalkApplicationsKey+configuration.CustomDingTalkConfiguration.AppId, binary, &infiniteDuration)
							if err != nil {
								Log.Error(err.Error())
								failedTargets = append(failedTargets, dingtalkTenant.Ids...)
								return
							} else {
								agentId = dingTalkConfigData.CompanyData[dingtalkTenantId].AgentId
								reqBody["agent_id"] = agentId
							}
						}
					}
				}

				// Create Url
				reqUrl = url.URL{
					Scheme:   "https",
					Host:     "oapi.dingtalk.com",
					Path:     "/topapi/message/corpconversation/sendbytemplate",
					RawQuery: query.Encode(),
				}

				// Create Request Body
				body, err := json.Marshal(reqBody)
				if err != nil {
					Log.Error(err.Error())
					failedTargets = append(failedTargets, dingtalkTenant.Ids...)
					return
				}

				// Create Request
				req := &http.Request{
					Method: "POST",
					URL:    &reqUrl,
					Body:   io.NopCloser(bytes.NewReader(body)),
				}

				// Create Client
				client := &http.Client{}
				resp, err := client.Do(req)
				if err != nil {
					Log.Error(err.Error())
					failedTargets = append(failedTargets, dingtalkTenant.Ids...)
					return
				}
				defer func(Body io.ReadCloser) {
					err := Body.Close()
					if err != nil {
						Log.Fatal(err)
					}
				}(resp.Body)

				// Read Response Body
				respBody, err := io.ReadAll(resp.Body)
				if err != nil {
					Log.Error(err.Error())
					failedTargets = append(failedTargets, dingtalkTenant.Ids...)
					return
				}

				// Parse Response Body
				var data map[string]interface{}
				err = json.Unmarshal(respBody, &data)
				if err != nil {
					Log.Error(err.Error())
					failedTargets = append(failedTargets, dingtalkTenant.Ids...)
					return
				}

				if data["errcode"].(float64) != 0 {
					Log.Error(data["errmsg"].(string))
					failedTargets = append(failedTargets, dingtalkTenant.Ids...)
					return
				} else {
					Log.Info(data)
					return
				}
			} else {
				Log.Info(data)
				return
			}
		}()
	}

	message.Target = ziqianIds

	// Redirect message to sockets for ziqian users
	Log.Info("Redirect message to sockets for ziqian users")
	redirectMessageToSockets(message, configuration)

	return failedTargets, nil
}

// CreateErrorMessage : This function is used to create error messages
func CreateErrorMessage(content string) []byte {
	// Initialize session destruction message and broadcast to all clients
	message := &WebSocketMessage{
		Action: ErrorAction,
		Data:   content,
	}
	return message.encode()
}

// CreateSystemMessage : This function is used to create system messages
func CreateSystemMessage(action, content string) []byte {
	// Initialize session destruction message and broadcast to all clients
	message := &WebSocketMessage{
		Action: action,
		Data:   content,
	}
	return message.encode()
}
