package models

// SendMessageV2Request : Send message request
type SendMessageV2Request struct {
	ClassId               string   `json:"classId,omitempty"`
	SubClassId            string   `json:"subclassId,omitempty"`
	MessageTemplateId     string   `json:"messageTemplateId,omitempty"`
	Params                []string `json:"params,omitempty"`
	Content               string   `json:"content,omitempty"`
	Source                string   `json:"source,omitempty"`
	Targets               []string `json:"targets,omitempty"`
	CustomEmailRequest    `json:"customEmailRequest,omitempty"`
	CustomSmsRequest      `json:"customSmsRequest,omitempty"`
	CustomFeishuRequestV2 `json:"customFeishuRequestV2,omitempty"`
}

// SendMessageV2Response : Send message response
type SendMessageV2Response struct {
	MessageId     string   `json:"messageId,omitempty"`
	FailedTargets []string `json:"failedTargets,omitempty"`
}

// MessageStatus : Model of message status to change message status
type MessageStatus struct {
	Status string `json:"status,omitempty"`
	OpenId string `json:"openId,omitempty"`
}

// QueryMessageParams : Query message parameters
type QueryMessageParams struct {
	AppId       string `form:"appId"`
	OpenId      string `form:"openId"`
	Source      string `form:"source"`
	ClassId     string `form:"classId"`
	ClassIds    string `form:"classIds"`
	SubclassId  string `form:"subclassId"`
	SubclassIds string `form:"subclassIds"`
	Status      string `form:"status"`
	Page        int64  `form:"page"`
	Limit       int64  `form:"limit"`
	Order       string `form:"order"`
	Sort        string `form:"sort"`
}

// QueryMessageResponse : Query message response
type QueryMessageResponse struct {
	Page    int64       `json:"page"`
	Limit   int64       `json:"limit"`
	Total   int64       `json:"total"`
	Results interface{} `json:"results"`
}

type CustomEmailRequest struct {
	Subject     string   `json:"subject" bson:"subject"`
	BodyType    string   `json:"bodyType" bson:"bodyType"`
	Cc          []string `json:"cc" bson:"cc"`
	Bcc         []string `json:"bcc" bson:"bcc"`
	Attachments []string `json:"attachments" bson:"attachments"`
}

type CustomSmsRequest struct {
	TemplateParams []string `json:"templateParams" bson:"templateParams"`
}

type CustomFeishuRequest struct {
	ReceiveIdType string `json:"receive_id_type" bson:"receive_id_type"`
	MsgType       string `json:"msg_type" bson:"msg_type"`
	CompanyId     string `json:"companyId" bson:"companyId"`
}

type CustomFeishuRequestV2 struct {
	MsgType string `json:"msg_type" bson:"msg_type"`
}

type CustomDingtalkRequest struct {
	CompanyId string `json:"companyId" bson:"companyId"`
}

type FeishuConfigData struct {
	AppId             string `json:"appId" bson:"appId"`
	AppSecret         string `json:"appSecret" bson:"appSecret"`
	AppType           string `json:"appType" bson:"appType"`
	AppTicket         string `json:"appTicket" bson:"appTicket"`
	TenantAccessToken string `json:"tenantAccessToken" bson:"tenantAccessToken"`
	TenantKey         string `json:"tenantKey" bson:"tenantKey"`
	CompanyData       map[string]FeishuCompany
}

type FeishuCompany struct {
	TenantAccessToken string `json:"tenantAccessToken" bson:"tenantAccessToken"`
}

type DingTalkConfigData struct {
	AppId       string                     `json:"appId" bson:"appId"`
	SuiteKey    string                     `json:"suiteKey" bson:"suiteKey"`
	SuiteSecret string                     `json:"suiteSecret" bson:"suiteSecret"`
	SuiteTicket string                     `json:"suiteTicket" bson:"suiteTicket"`
	SuiteId     string                     `json:"suiteId" bson:"suiteId"`
	AuthCorpId  string                     `json:"authCorpId" bson:"authCorpId"`
	AccessToken string                     `json:"accessToken" bson:"accessToken"`
	CompanyData map[string]DingTalkCompany `json:"companyData" bson:"companyData"`
	AgentId     string                     `json:"agentId" bson:"agentId"`
}

type DingTalkCompany struct {
	AccessToken string `json:"accessToken" bson:"accessToken"`
	AgentId     string `json:"agentId" bson:"agentId"`
}
