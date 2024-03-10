package bus

import (
	"encoding/json"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"testing"
)

// 环境变量 CONFIG_PATH=../../../bin/conf
// invalid character 'ä' looking for beginning of value"  还差一个这个问题。
func TestConsume(t *testing.T) {
	go comsumTest()

	f := &ZixelMqBusDo{}
	//需要对父类函数进行赋值
	f.ZixelMqBus.Consume = f.consumeTestEEE
	f.InitZixel("eee", "", "", "messengerConsumer2")
}

func comsumTest() {
	e := &ZixelMqBusDo{}
	//需要对父类函数进行赋值
	e.ZixelMqBus.Consume = e.consumeTestDemo
	e.InitZixel("ThreePart", "", "", "messengerConsumer")
}

func (d *ZixelMqBusDo) consumeTestDemo(msg *primitive.MessageExt) consumer.ConsumeResult {
	var data map[string]interface{}
	err := json.Unmarshal(msg.Body, &data)
	log.Info("========收到消息 :", msg)
	if err != nil {
		log.Error("consumeTestDemo>>>", err.Error())
		//return consumer.ConsumeRetryLater, nil
	}
	return consumer.ConsumeSuccess
}

func (d *ZixelMqBusDo) consumeTestEEE(msg *primitive.MessageExt) consumer.ConsumeResult {
	var data map[string]interface{}
	err := json.Unmarshal(msg.Body, &data)
	log.Info("========收到消息 :", msg)
	if err != nil {
		log.Error("consumeTestDemo>>>", err.Error())
		//return consumer.ConsumeRetryLater, nil
	}
	return consumer.ConsumeSuccess
}
