package models

import (
	"google.golang.org/protobuf/types/known/structpb"
)

// Policy : Policy of message class configuration
type Policy struct {
	Kafka string `json:"kafka" bson:"kafka"`
	Http  string `json:"http"  bson:"http"`
}

// ActionModel : Model of action
type ActionModel struct {
	Name       string        `json:"name" validate:"required" bson:"name"`
	Tips       string        `json:"tips" validate:"required" bson:"tips"`
	Action     ActionTrigger `json:"action,omitempty" validate:"omitempty" bson:"action,omitempty"`
	NextAction string        `json:"nextAction,omitempty" validate:"omitempty" bson:"nextAction"`
}

// ActionTrigger : Event Trigger of Action
type ActionTrigger struct {
	Type   string           `json:"type,omitempty"   bson:"type,omitempty"`
	URL    string           `json:"url,omitempty"    bson:"url,omitempty"`
	Method string           `json:"method,omitempty"  bson:"method,omitempty"`
	Query  *structpb.Struct `json:"query,omitempty"`
	Body   *structpb.Struct `json:"body,omitempty"`
}

type CustomEmailConfiguration struct {
	EmailAccount  string `json:"emailAccount" bson:"emailAccount"`
	EmailPassword string `json:"emailPassword" bson:"emailPassword"`
}

type CustomSmsConfiguration struct {
	Huawei Huawei `json:"huawei,omitempty" bson:"huawei,omitempty"`
}

type CustomFeishuConfiguration struct {
	AppId     string `json:"appId" bson:"appId"`
	AppSecret string `json:"appSecret" bson:"appSecret"`
	Type      string `json:"type" bson:"type"`
}

type Huawei struct {
	Channel    string `json:"channel" bson:"channel"`
	Signature  string `json:"signature" bson:"signature"`
	TemplateId string `json:"templateId" bson:"templateId"`
}

type CustomDingTalkConfiguration struct {
	SuiteId     string `json:"suiteId" bson:"suiteId"`
	AppId       string `json:"appId" bson:"appId"`
	MiniAppId   string `json:"miniAppId" bson:"miniAppId"`
	SuiteKey    string `json:"suiteKey" bson:"suiteKey"`
	SuiteSecret string `json:"suiteSecret" bson:"suiteSecret"`
	TemplateId  string `json:"templateId" bson:"templateId"`
	AgentId     string `json:"agentId" bson:"agentId"`
}

// QueryConfigurationsParams : Get Configuration Params
type QueryConfigurationsParams struct {
	Name     string `form:"name"`
	Type     string `form:"type"`
	Mode     string `form:"mode"`
	ParentId string `form:"parentId"`
	AppId    string `form:"appId"`
	Page     int64  `form:"page"`
	Limit    int64  `form:"limit"`
	Order    string `form:"order"`
	Sort     string `form:"sort"`
}

// QueryConfigurationResponse : Get Configuration Response
type QueryConfigurationResponse struct {
	Page    int64       `json:"page"`
	Limit   int64       `json:"limit"`
	Total   int64       `json:"total"`
	Results interface{} `json:"results"`
}

// CreateConfigurationRequest : Model of message class configuration
type CreateConfigurationRequest struct {
	ClassId                     string                      `json:"classId" validate:"required" bson:"classId"`
	ParentId                    string                      `json:"parentId" validate:"required" bson:"parentId"`
	AppId                       string                      `json:"appId" bson:"appId"`
	Name                        string                      `json:"name" bson:"name"`
	Persist                     bool                        `json:"persist" bson:"persist"`
	States                      []string                    `json:"states" bson:"states"`
	Type                        string                      `json:"type" bson:"type"`
	Mode                        string                      `json:"mode" bson:"mode"`
	StatusCallbacks             Policy                      `json:"statusCallbacks" bson:"statusCallbacks"`
	Actions                     []ActionModel               `json:"actions,omitempty" bson:"actions,omitempty"`
	CustomEmailConfiguration    CustomEmailConfiguration    `json:"customEmailConfiguration,omitempty" bson:"customEmailConfiguration,omitempty"`
	CustomSmsConfiguration      CustomSmsConfiguration      `json:"customSmsConfiguration,omitempty" bson:"customSmsConfiguration,omitempty"`
	CustomFeishuConfiguration   CustomFeishuConfiguration   `json:"customFeishuConfiguration,omitempty" bson:"customFeishuConfiguration,omitempty"`
	CustomDingTalkConfiguration CustomDingTalkConfiguration `json:"customDingTalkConfiguration,omitempty" bson:"customDingTalkConfiguration,omitempty"`
}

type UpdateConfigurationRequest struct {
	ClassId                     string                      `json:"classId" validate:"required" bson:"classId"`
	AppId                       string                      `json:"appId" bson:"appId"`
	Name                        string                      `json:"name" bson:"name"`
	Persist                     bool                        `json:"persist" bson:"persist"`
	States                      []string                    `json:"states" bson:"states"`
	Type                        string                      `json:"type" bson:"type"`
	Mode                        string                      `json:"mode" bson:"mode"`
	StatusCallbacks             Policy                      `json:"statusCallbacks" bson:"statusCallbacks"`
	Actions                     []ActionModel               `json:"actions,omitempty" bson:"actions,omitempty"`
	CustomEmailConfiguration    CustomEmailConfiguration    `json:"customEmailConfiguration,omitempty" bson:"customEmailConfiguration,omitempty"`
	CustomSmsConfiguration      CustomSmsConfiguration      `json:"customSmsConfiguration,omitempty" bson:"customSmsConfiguration,omitempty"`
	CustomFeishuConfiguration   CustomFeishuConfiguration   `json:"customFeishuConfiguration,omitempty" bson:"customFeishuConfiguration,omitempty"`
	CustomDingTalkConfiguration CustomDingTalkConfiguration `json:"customDingTalkConfiguration,omitempty" bson:"customDingTalkConfiguration,omitempty"`
}
