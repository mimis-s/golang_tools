package rabbitmq

import (
	"fmt"

	"github.com/streadway/amqp"
	"google.golang.org/protobuf/proto"
)

// 生产者
type producers struct {
	channel      *amqp.Channel
	exchangeName string
	routingKey   string
	persistent   uint8 // 可持久化(和durable一样, 不过这个值是uint8的)
}

// 初始化生产者,连接mq服务器
func InitProducers(url, exchangeName, routingKey string, durable bool) (*producers, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("连接mq错误:%v", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("打开channel错误:%v", err)
	}

	err = ch.ExchangeDeclare(
		exchangeName, // name
		"topic",      // type
		durable,      // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		return nil, fmt.Errorf("定义exchange错误:%v", err)
	}

	persistent := uint8(0)
	if durable {
		persistent = amqp.Persistent
	} else {
		persistent = amqp.Transient
	}

	return &producers{ch, exchangeName, routingKey, persistent}, nil
}

// 发送消息
func (p *producers) Publish(payload interface{}) error {
	msg, err := proto.Marshal(payload.(proto.Message))
	if err != nil {
		return err
	}
	err = p.channel.Publish(
		p.exchangeName,
		p.routingKey,
		false,
		false,
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "text/plain",
			Body:         msg,
		},
	)
	if err != nil {
		return fmt.Errorf("mq send msg[%v] is err:%v", msg, err)
	}
	return nil
}
