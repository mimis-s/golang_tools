package rabbitmq

import (
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/streadway/amqp"
)

// 生产者
type producers struct {
	channel      *amqp.Channel
	exchangeName string
	routingKey   string
}

// 初始化生产者,连接mq服务器
func InitProducers(url, exchangeName, routingKey string) (*producers, error) {
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
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		return nil, fmt.Errorf("定义exchange错误:%v", err)
	}

	return &producers{ch, exchangeName, routingKey}, nil
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
