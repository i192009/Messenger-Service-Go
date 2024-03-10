package bus

import (
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
)

type ZixelMqBus struct {
	Consume func(msg *primitive.MessageExt) consumer.ConsumeResult
}

type ZixelMqBusDo struct {
	ZixelMqBus
}
