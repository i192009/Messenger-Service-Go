package sdk

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"time"
)

type CreateLogRequest struct {
	Logs []LogRecordRequest `json:"logs"`
}

type CreateLogResponse struct {
	Logs       []LogRecord `json:"logs"`
	FailedLogs int         `json:"failedLogs"`
}

type LogRecordRequest struct {
	LogType    string                 `json:"logType" bson:"logType" validate:"required"`
	SubLogType string                 `json:"subLogType" bson:"subLogType"`
	LogPath    string                 `json:"logPath" bson:"logPath" validate:"required"`
	Source     string                 `json:"source" bson:"source" validate:"required"`
	Timestamp  int                    `json:"timestamp" bson:"timestamp" validate:"required"`
	Tags       []string               `json:"tags" bson:"tags"`
	EventData  map[string]interface{} `json:"eventData" bson:"eventData"`
}

type KLogHeaders struct {
	ApplicationId  string `json:"applicationId"`
	InstanceId     string `json:"instanceId"`
	UserId         string `json:"userId"`
	OrganizationId string `json:"organizationId"`
	EmployeeId     string `json:"employeeId"`
	OpenId         string `json:"openId"`
	DeviceId       string `json:"deviceId"`
}

type LogRecord struct {
	AppId          string                 `bson:"appId"`
	InstanceId     string                 `bson:"instanceId"`
	UserId         string                 `bson:"userId"`
	OrganizationId string                 `bson:"organizationId"`
	EmployeeId     string                 `bson:"employeeId"`
	OpenId         string                 `bson:"openId"`
	LogType        string                 `bson:"logType"`
	SubLogType     string                 `bson:"subLogType"`
	LogPath        string                 `bson:"logPath"`
	Source         string                 `bson:"source"`
	Timestamp      int                    `bson:"timestamp"`
	Tags           []string               `bson:"tags"`
	EventData      map[string]interface{} `bson:"eventData"`
}

func StoreOneLogInKLogService(request LogRecordRequest, headers KLogHeaders) (*LogRecord, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	record, err := StoreOneLogInKLogServiceCtx(ctx, request, headers)
	if err != nil {
		log.Error("StoreOneLogInKLogService error: %v", err)
		return nil, err
	} else {
		return record, nil
	}
}

func StoreOneLogInKLogServiceCtx(ctx context.Context, requestBody LogRecordRequest, requestHeaders KLogHeaders) (*LogRecord, error) {
	createLogRequest := CreateLogRequest{
		Logs: []LogRecordRequest{requestBody},
	}

	reqUrl := url.URL{
		Scheme: "http",
		Host:   "klog-svc:8573",
		Path:   "/klog/v1/logs",
	}

	headers := http.Header{
		"Zixel-Application-Id":  []string{requestHeaders.ApplicationId},
		"Zixel-Instance-Id":     []string{requestHeaders.InstanceId},
		"Zixel-User-Id":         []string{requestHeaders.UserId},
		"Zixel-Organization-Id": []string{requestHeaders.OrganizationId},
		"Zixel-Employee-Id":     []string{requestHeaders.EmployeeId},
		"Zixel-Open-Id":         []string{requestHeaders.OpenId},
		"Zixel-Device-Id":       []string{requestHeaders.DeviceId},
	}

	body, err := json.Marshal(createLogRequest)
	if err != nil {
		log.Error("StoreOneLogInKLogServiceCtx error: %v", err)
		return nil, err
	}

	req := &http.Request{
		Method: http.MethodPost,
		URL:    &reqUrl,
		Body:   io.NopCloser(bytes.NewReader(body)),
		Header: headers,
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Error("StoreOneLogInKLogServiceCtx error: %v", err)
		return nil, err
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(resp.Body)

	// Read Response Body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error("StoreOneLogInKLogServiceCtx error: %v" + err.Error())
		return nil, err
	}

	// Parse Response Body
	var response CreateLogResponse
	err = json.Unmarshal(respBody, &response)
	if err != nil {
		log.Error("StoreOneLogInKLogServiceCtx error: %v" + err.Error())
		return nil, err
	}
	if response.FailedLogs == 1 {
		log.Error("StoreOneLogInKLogServiceCtx error: %v" + err.Error())
		return nil, errors.New("failed to validate log")
	} else {
		return &response.Logs[0], nil
	}
}

func StoreBatchLogInKLogService(request CreateLogRequest, headers KLogHeaders) (*CreateLogResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	record, err := StoreBatchLogInKLogServiceCtx(ctx, request, headers)
	if err != nil {
		log.Error("StoreOneLogInKLogService error: %v", err)
		return nil, err
	} else {
		return record, nil
	}
}

func StoreBatchLogInKLogServiceCtx(ctx context.Context, requestBody CreateLogRequest, requestHeaders KLogHeaders) (*CreateLogResponse, error) {
	reqUrl := url.URL{
		Scheme: "http",
		Host:   "klog-svc:8573",
		Path:   "/klog/v1/logs",
	}

	headers := http.Header{
		"Zixel-Application-Id":  []string{requestHeaders.ApplicationId},
		"Zixel-Instance-Id":     []string{requestHeaders.InstanceId},
		"Zixel-User-Id":         []string{requestHeaders.UserId},
		"Zixel-Organization-Id": []string{requestHeaders.OrganizationId},
		"Zixel-Employee-Id":     []string{requestHeaders.EmployeeId},
		"Zixel-Open-Id":         []string{requestHeaders.OpenId},
		"Zixel-Device-Id":       []string{requestHeaders.DeviceId},
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		log.Error("StoreOneLogInKLogServiceCtx error: %v", err)
		return nil, err
	}

	req := &http.Request{
		Method: http.MethodPost,
		URL:    &reqUrl,
		Body:   io.NopCloser(bytes.NewReader(body)),
		Header: headers,
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Error("StoreOneLogInKLogServiceCtx error: %v", err)
		return nil, err
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(resp.Body)

	// Read Response Body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error("StoreOneLogInKLogServiceCtx error: %v" + err.Error())
		return nil, err
	}

	// Parse Response Body
	var response CreateLogResponse
	err = json.Unmarshal(respBody, &response)
	if err != nil {
		log.Error("StoreOneLogInKLogServiceCtx error: %v" + err.Error())
		return nil, err
	}
	if response.FailedLogs > 0 {
		log.Error("StoreOneLogInKLogServiceCtx error: %v" + err.Error())
		return nil, errors.New("failed to validate log")
	} else {
		return &response, nil
	}
}

func NewLogRecord(
	logType, subLogType, logPath, source string,
	timestamp int, tags []string, eventData map[string]interface{}) *LogRecordRequest {
	var logRecord LogRecordRequest
	logRecord.LogType = logType
	logRecord.SubLogType = subLogType
	logRecord.LogPath = logPath
	logRecord.Source = source
	logRecord.Timestamp = timestamp
	logRecord.Tags = tags
	logRecord.EventData = eventData
	return &logRecord
}

func NewLogRecordRequest(logs []LogRecordRequest) *CreateLogRequest {
	var createLogRequest CreateLogRequest
	createLogRequest.Logs = logs
	return &createLogRequest
}
