package bus

import (
	"context"
	"time"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	rocketMqPrimitive "github.com/apache/rocketmq-client-go/v2/primitive"
	"gitlab.zixel.cn/go/framework/config"
)

func (e *ZixelMqBus) InitZixel(namespace string, service string, secretkey string, consumerGroup string) error {
	log.Infof(" initZixel " + namespace + " init-------------\n")

	// 订阅主题、消费
	endPoint := []string{config.GetString("messagebus.nameserver", "http://rocketmq-svc:9876")}

	// 创建一个consumer实例
	c, err := rocketmq.NewPushConsumer(
		consumer.WithNsResolver(rocketMqPrimitive.NewPassthroughResolver(endPoint)),
		consumer.WithConsumerModel(consumer.Clustering),
		consumer.WithGroupName(consumerGroup),
		consumer.WithConsumerPullTimeout(time.Duration(config.GetReal("messagebus.pulltimeout", 1))*time.Second),
	)

	if err != nil {
		log.Errorf("rocketmq.NewPushConsumer is error.err : %v \n", err)
		return err
	}
	// 订阅topic
	err = c.Subscribe(namespace, consumer.MessageSelector{},
		func(ctx context.Context, messages ...*rocketMqPrimitive.MessageExt) (consumer.ConsumeResult, error) {
			for _, msg := range messages {
				result := e.Consume(msg)
				if result != consumer.ConsumeSuccess {
					return result, nil
				}
			}
			return consumer.ConsumeSuccess, nil
		})

	if err != nil {
		log.Errorf("subscribe message error: %s\n", err.Error())
		return err
	}

	// 启动consumer
	err = c.Start()
	log.Debug("Subcribe Message  start-------------\n")
	if err != nil {
		log.Errorf("subscribe Start error: %s\n", err.Error())
		return err
	}

	defer func(c rocketmq.PushConsumer) {
		err := c.Shutdown()
		if err != nil {
			log.Errorf("subscribe Shutdown error: %s\n", err.Error())
		}
	}(c)

	select {}
}
