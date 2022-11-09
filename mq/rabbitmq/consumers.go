package rabbitmq

import (
	"fmt"

	"github.com/streadway/amqp"
	"google.golang.org/protobuf/proto"
)

// 消费者
type consumers struct {
	channel   *amqp.Channel
	queueName string
}

func InitConsumers(url, exchangeName, routingKey string) (*consumers, error) {
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

	// 消费者队列
	q, err := ch.QueueDeclare(
		"",    // name
		false, // durable
		false, // delete when usused
		true,  // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return nil, fmt.Errorf("定义queue错误:%v", err)
	}

	// 绑定队列到exchange，通过routing路由到消费者
	err = ch.QueueBind(
		q.Name,
		routingKey,
		exchangeName,
		false,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("queue[%v]绑定exchange[%v]错误:%v", q.Name, exchangeName, err)
	}

	return &consumers{ch, q.Name}, nil
}

func (c *consumers) RegisterConsume(callBack func(payload interface{}) error) error {

	errChannel := make(chan error)

	msgs, err := c.channel.Consume(
		c.queueName,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("接收mq消息错误:%v", err)
	}

	go func() {
		for d := range msgs {
			var pb proto.Message
			err := proto.Unmarshal(d.Body, pb)
			if err != nil {
				errChannel <- err
				return
			}
			err = callBack(pb)
			if err != nil {
				errChannel <- err
				return
			}
		}
	}()
	select {
	case err = <-errChannel:
		return err
	}

}
