package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/apache/rocketmq-client-go/v2/producer"
	"github.com/go-playground/validator"
	"gitlab.zixel.cn/go/framework/logger"
)

var (
	validate = validator.New()
	log      = logger.Get()
)

type OperationRecordReq struct {
	ProjectCode  string `json:"project_code" validate:"required,max=50"` //项目code（标识项目系统的操作日志）
	ServiceCode  string `json:"service_code" validate:"required,max=50"` //服务code（标识具体微服务模块）
	UserId       string `json:"user_id" validate:"required,max=50"`      //用户的唯一标识
	RequestId    string `json:"request_id" validate:"required,max=100"`  //请求id（每次请求的唯一ID）
	OperateLevel string `json:"operate_level" validate:"max=50"`         //操作日志级别
	OperateType  string `json:"operate_type" validate:"max=50"`          //操作类型（日志类型）
	Message      string `json:"message" validate:"required,max=500"`     //操作内容
	OperateTime  string `json:"operate_time" validate:"required"`        //
	OperateOrder int32  `json:"operate_order"`                           //操作顺序
	DeviceId     string `json:"device_id" validate:"max=200"`            //设备id
	Ip           string `json:"ip" validate:"max=20""`                   //操作ip
}

type KlogProducer struct {
	NameServers []string
}

func NewKlogProducer(NameServers []string) *KlogProducer {
	return &KlogProducer{NameServers: NameServers}
}

var topic = "topicOperationLog"

// 异步发送操作日志消息
func (Produce *KlogProducer) AsyncSendOperationLog(operationRecordReq *OperationRecordReq, tag string) error {
	var (
		err error
	)
	err = validate.Struct(operationRecordReq)
	if err != nil {
		return err
	}
	body, err := json.Marshal(operationRecordReq)
	if err != nil {
		return err
	}
	p, err := rocketmq.NewProducer(producer.WithNameServer(Produce.NameServers), producer.WithRetry(2))
	if err != nil {
		return err
	}
	if err = p.Start(); err != nil {
		return err
	}
	if tag != "" {
		topic = topic + ":" + tag
	}
	res, err := p.SendSync(context.Background(), primitive.NewMessage(topic, body))
	if err != nil {
		return nil
	} else {
		return err
	}
	if err = p.Shutdown(); err != nil {
		return err
	}
	fmt.Printf("send message success. result=%s\n", res.String())
	return nil
}
