package models

// QueryTemplateParams : Query template parameters
type QueryTemplateParams struct {
	AppId     string `form:"appId"`
	Name      string `form:"name"`
	Type      string `form:"type"`
	CreatedBy string `form:"createdBy"`
	Language  string `form:"language"`
	Page      int64  `form:"page"`
	Limit     int64  `form:"limit"`
	Order     string `form:"order"`
	Sort      string `form:"sort"`
}

// QueryTemplateResponse : Query template response
type QueryTemplateResponse struct {
	Page    int64       `json:"page"`
	Limit   int64       `json:"limit"`
	Total   int64       `json:"total"`
	Results interface{} `json:"results"`
}
