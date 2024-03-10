package controllers

// importing packages
import (
	"context"
	"messenger/restful/models"
	"messenger/restful/services"
	pb "messenger/services"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MessageServer : Implementation of MessageServer grpc
type MessageServer struct {
	pb.MessageServiceServer
}

// SendMessage : Call the service to send the message
func (s *MessageServer) SendMessage(ctx context.Context, req *pb.SendMessageRequest) (*pb.Message, error) {
	// Converting message from proto to struct
	message := services.Message{
		AppId:             req.GetAppId(),
		OpenId:            req.GetOpenId(),
		ClassId:           req.GetClassId(),
		SubClassId:        req.GetSubclassId(),
		MessageTemplateId: req.GetMessageTemplateId(),
		Params:            req.GetParams(),
		Content:           req.GetContent(),
		Source:            req.GetSource(),
		Target:            req.GetTarget(),
		CustomEmailRequest: models.CustomEmailRequest{
			Subject:     req.GetCustomEmailRequest().GetSubject(),
			BodyType:    req.GetCustomEmailRequest().GetBodyType(),
			Cc:          req.GetCustomEmailRequest().GetCc(),
			Bcc:         req.GetCustomEmailRequest().GetBcc(),
			Attachments: req.GetCustomEmailRequest().GetAttachments(),
		},
		CustomSmsRequest: models.CustomSmsRequest{
			TemplateParams: req.GetCustomSmsRequest().GetTemplateParams(),
		},
		CustomFeishuRequest: models.CustomFeishuRequest{
			ReceiveIdType: req.GetCustomFeishuRequest().GetReceiveIdType(),
			MsgType:       req.GetCustomFeishuRequest().GetMsgType(),
			CompanyId:     req.GetCustomFeishuRequest().GetCompanyId(),
		},
		CustomDingtalkRequest: models.CustomDingtalkRequest{
			CompanyId: req.GetCustomDingTalkRequest().GetCompanyId(),
		},
	}

	// Returning error and logging if failed to send message
	if err := services.SendMessage(ctx, &message); err != nil {
		return nil, err
	}

	// Converting actions from struct to proto
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

	// Returning success and logging if message sent successfully
	return &pb.Message{
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
		StatusSync:        services.ConvertMapToStruct(message.StatusSync),
		CreatedAt:         message.CreatedAt.Format(time.RFC3339),
		UpdatedAt:         message.UpdatedAt.Format(time.RFC3339),
		Actions:           responseActions,
	}, nil
}

// SendMessageV2 : Call the service to send the message
func (s *MessageServer) SendMessageV2(ctx context.Context, req *pb.C2S_SendMessageV2ReqT) (*pb.S2C_SendMessageV2RpnT, error) {
	// Converting message from proto to struct
	request := models.SendMessageV2Request{
		ClassId:           req.GetClassId(),
		MessageTemplateId: req.GetMessageTemplateId(),
		Params:            req.GetParams(),
		Content:           req.GetContent(),
		Source:            req.GetSource(),
		Targets:           req.GetTarget(),
		CustomEmailRequest: models.CustomEmailRequest{
			Subject:     req.GetCustomEmailRequest().GetSubject(),
			BodyType:    req.GetCustomEmailRequest().GetBodyType(),
			Cc:          req.GetCustomEmailRequest().GetCc(),
			Bcc:         req.GetCustomEmailRequest().GetBcc(),
			Attachments: req.GetCustomEmailRequest().GetAttachments(),
		},
		CustomSmsRequest: models.CustomSmsRequest{
			TemplateParams: req.GetCustomSmsRequest().GetTemplateParams(),
		},
		CustomFeishuRequestV2: models.CustomFeishuRequestV2{
			MsgType: req.GetCustomFeishuRequest().GetMsgType(),
		},
	}

	// Returning error and logging if failed to send message
	resp, err := services.SendMessageV2(ctx, &request)
	if err != nil {
		return nil, err
	}

	var success bool
	var msg string
	if len(resp.FailedTargets) > 0 {
		success = false
		msg = "Message sent with some failed targets!"
	} else {
		success = true
		msg = "Message sent successfully!"
	}
	// Returning success and logging if message sent successfully
	return &pb.S2C_SendMessageV2RpnT{
		MessageId:     resp.MessageId,
		Message:       msg,
		Success:       success,
		FailedTargets: resp.FailedTargets,
	}, nil
}

// UpdateMessageStatus : Call the service to update the status of the message
func (s *MessageServer) UpdateMessageStatus(ctx context.Context, req *pb.UpdateMessageRequest) (*pb.Message, error) {
	// Initializing message struct
	var message services.Message

	// Converting message status from proto to struct
	status := models.MessageStatus{
		Status: req.GetStatus(),
		OpenId: req.GetOpenId(),
	}

	// Getting message id
	id := req.GetId()

	// Returning error and logging if failed to update message status
	if err := services.UpdateMessageStatus(ctx, &message, &status, id); err != nil {
		return nil, err
	}

	// Converting actions from struct to proto
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

	// Returning success and logging if message status updated successfully
	log.Info(message)
	return &pb.Message{
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
		StatusSync:        services.ConvertMapToStruct(message.StatusSync),
		CreatedAt:         message.CreatedAt.Format(time.RFC3339),
		UpdatedAt:         message.UpdatedAt.Format(time.RFC3339),
		Actions:           responseActions,
	}, nil
}

// DeleteMessage : Call the service to delete the message
func (s *MessageServer) DeleteMessage(ctx context.Context, req *pb.DeleteMessageRequest) (*pb.DeleteMessageResponse, error) {
	// Initializing message struct
	var message services.Message

	// Converting message status from proto to struct
	status := models.MessageStatus{
		OpenId: req.GetOpenId(),
	}

	// Getting message id
	id := req.GetId()

	// Returning error and logging if failed to delete message
	if err := services.DeleteMessage(ctx, &message, &status, id); err != nil {
		return nil, err
	}

	// Returning success and logging if message deleted successfully
	log.Info(message.Id.Hex() + "status changed to deleted")
	return &pb.DeleteMessageResponse{
		Message: "Message Deleted successfully",
	}, nil
}

// QueryMessages : Call the service to query the messages
func (s *MessageServer) QueryMessages(ctx context.Context, req *pb.QueryMessagesRequest) (*pb.QueryMessagesResponse, error) {
	// Converting query from proto to struct
	query := models.QueryMessageParams{
		AppId:       req.GetAppId(),
		OpenId:      req.GetOpenId(),
		Source:      req.GetSource(),
		ClassId:     req.GetClassId(),
		ClassIds:    req.GetClassIds(),
		SubclassId:  req.GetSubclassId(),
		SubclassIds: req.GetSubclassIds(),
		Status:      req.GetStatus(),
		Page:        req.GetPage(),
		Limit:       req.GetLimit(),
		Order:       req.GetOrder(),
		Sort:        req.GetSort(),
	}

	// Returning error and logging if failed to query messages
	messages, err := services.QueryMessages(ctx, &query)
	if err != nil {
		return nil, err
	}

	// Returning success and logging if messages queried successfully
	log.Info("Messages queried successfully!")
	return messages, nil
}

// GetMessage : Call the service to get the messages
func (s *MessageServer) GetMessage(ctx context.Context, req *pb.GetMessagesRequest) (*pb.QueryMessagesResponse, error) {
	// Getting user openId
	openid := req.GetOpenId()

	// Returning error and logging if failed to get user messages
	messages, err := services.GetMessages(ctx, openid)
	if err != nil {
		return nil, err
	}

	// Returning success and logging if messages queried successfully
	log.Info(messages)
	return messages, nil
}

// CreateWebSocketConnection : Create WebSocket Connection for push messaging
func CreateWebSocketConnection(c *gin.Context) {
	// Returning error and logging if failed to create WebSocket connection
	if err := services.CreateWebSocketConnection(c); err != nil {
		return
	}

	// Returning success and logging if WebSocket connection created successfully
	services.Log.Info("WebSocket Connection Created")
}

// HttpGetMessages : Calls the service to get the cached messages for the user
func HttpGetMessages(c *gin.Context) {
	// Returning error and logging if failed to get user messages
	messages, err := services.HttpGetMessages(c)
	if err != nil {
		SendErrorToJson(c, err)
		return
	} else {
		// Returning success and logging if messages queried successfully
		services.Log.Info(messages)
		c.JSON(http.StatusOK, messages)
	}
}

// HttpUpdateMessageStatus : Calls the service to update the status of the message
func HttpUpdateMessageStatus(c *gin.Context) {
	// Returning error and logging if failed to update message status
	if err := services.HttpUpdateMessageStatus(c); err != nil {
		SendErrorToJson(c, err)
		return
	} else {
		// Returning success and logging if message status updated successfully
		services.Log.Info("Message Status Updated " + c.Param("messageId"))
		c.JSON(http.StatusOK, gin.H{
			"message": "Message Status Updated successfully",
		})
	}
}

// HttpUpdateMessagesStatus : Calls the service to update the status of the message
func HttpUpdateMessagesStatus(c *gin.Context) {
	// Returning error and logging if failed to update message status
	if err := services.HttpUpdateMessagesStatus(c); err != nil {
		SendErrorToJson(c, err)
		return
	} else {
		// Returning success and logging if message status updated successfully
		services.Log.Info("Message Statuses Updated ")
		c.JSON(http.StatusOK, gin.H{
			"message": "Message Statuses Updated successfully",
		})
	}
}

// HttpDeleteMessage : Calls the service to delete the message
func HttpDeleteMessage(c *gin.Context) {
	// Returning error and logging if failed to delete message
	if err := services.HttpDeleteMessage(c); err != nil {
		SendErrorToJson(c, err)
		return
	} else {
		// Returning success and logging if message deleted successfully
		services.Log.Info("Message Deleted successfully " + c.Param("messageId"))
		c.JSON(http.StatusOK, gin.H{
			"message": "Message Deleted successfully",
		})
	}
}

// HttpQueryMessages : Calls the service to query the messages
func HttpQueryMessages(c *gin.Context) {
	// Returning error and logging if failed to query messages
	messages, err := services.HttpQueryMessages(c)
	if err != nil {
		SendErrorToJson(c, err)
		return
	} else {
		// Returning success and logging if messages queried successfully
		services.Log.Info("Messages queried successfully!")
		c.JSON(http.StatusOK, messages)
	}
}
func (s *MessageServer) SendMessageByScene(ctx context.Context, req *pb.C2S_SendMessageBySceneReqT) (*pb.S2C_SendMessageBySceneRpnT, error) {
	id := primitive.NewObjectID()
	id.UnmarshalText([]byte(req.String()))
	// Converting message from proto to struct
	scene := services.SceneRelation{
		SceneId:    req.SceneId,
		AppId:      req.AppId,
		TenantId:   req.TenantId,
		InstanceId: req.InstanceId,
	}

	// Returning error and logging if failed to send message
	resp, err := services.SendMessageByScene(ctx, &scene, req.Arguments)
	if err != nil {
		return nil, err
	}

	var success bool
	var msg string
	if len(resp.FailedTargets) > 0 {
		success = false
		msg = "Message sent with some failed targets!"
	} else {
		success = true
		msg = "Message sent successfully!"
	}
	// Returning success and logging if message sent successfully
	log.Info(resp)
	return &pb.S2C_SendMessageBySceneRpnT{
		MessageId:     resp.MessageId,
		Message:       msg,
		Success:       success,
		FailedTargets: resp.FailedTargets,
	}, nil
}
