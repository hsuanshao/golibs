package entities

import "github.com/hsuanshao/golibs/ctx"

type MQTTClient interface {
	Connect(ctx ctx.CTX) MQTTError
	Disconnect(ctx ctx.CTX) MQTTError
	Publish(ctx ctx.CTX, topic string, payload []byte) MQTTError
	Subscribe(ctx ctx.CTX, topic string, handler func(ctx ctx.CTX, topic string, payload []byte) MQTTError) MQTTError
}
