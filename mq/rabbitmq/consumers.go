package rabbitmq

import (
	"encoding/json"
	"fmt"

	"github.com/streadway/amqp"
	"google.golang.org/protobuf/proto"
)

// 消费者
type consumers struct {
	channel   *amqp.Channel
	queueName string
}

// 消费者队列
type ConsumersQueue struct {
	ExchangeName string
	RoutingKey   string
	CallBack     func(payload interface{}) error
}

// durable可持久化
func RegisterConsumers(url string, durable bool, cQueue []*ConsumersQueue) error {
	conn, err := amqp.Dial(url)
	if err != nil {
		return fmt.Errorf("连接mq错误:%v", err)
	}

	for _, c := range cQueue {

		ch, err := conn.Channel()
		if err != nil {
			return fmt.Errorf("打开channel错误:%v", err)
		}

		err = ch.ExchangeDeclare(
			c.ExchangeName, // name
			"topic",        // type
			durable,        // durable
			false,          // auto-deleted
			false,          // internal
			false,          // no-wait
			nil,            // arguments
		)
		if err != nil {
			return fmt.Errorf("定义exchange错误:%v", err)
		}

		// 消费者队列
		q, err := ch.QueueDeclare(
			"",      // name
			durable, // durable
			false,   // delete when usused
			true,    // exclusive
			false,   // no-wait
			nil,     // arguments
		)
		if err != nil {
			return fmt.Errorf("定义queue错误:%v", err)
		}

		// 绑定队列到exchange，通过routing路由到消费者
		err = ch.QueueBind(
			q.Name,
			c.RoutingKey,
			c.ExchangeName,
			false,
			nil,
		)
		if err != nil {
			return fmt.Errorf("queue[%v]绑定exchange[%v]错误:%v", q.Name, c.ExchangeName, err)
		}

		go consume(ch, q.Name, c.CallBack)
	}

	return nil
}

func consume(ch *amqp.Channel, queueName string, callBack func(payload interface{}) error) error {

	errChannel := make(chan *amqp.Error)

	msgs, err := ch.Consume(
		queueName,
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
	ch.NotifyClose(errChannel)

	go func() {
		for d := range msgs {
			var pb proto.Message
			err := json.Unmarshal(d.Body, pb)
			if err != nil {
				fmt.Printf("proto Unmarshal[%v] is err[%v]", d.Body, err)
			}
			err = callBack(pb)
			if err != nil {
				fmt.Printf("mq callback msg[%v] is err[%v]", d.Body, err)
			}
		}
	}()
	select {
	case mqErr := <-errChannel:
		return mqErr
	default:
		return nil
	}
}
