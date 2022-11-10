package rabbitmq

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/streadway/amqp"
)

// 消费者
type consumers struct {
	channel   *amqp.Channel
	queueName string
}

// 消费者队列
type ConsumersQueue struct {
	RoutingKey    string                          // 路由key
	payLoadStruct interface{}                     // 要解析的结构体
	CallBack      func(payload interface{}) error // 回调函数
}

// durable可持久化
func RegisterConsumers(url string, durable bool, exchangeName string, cQueue []*ConsumersQueue) error {
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
			exchangeName, // name
			"topic",      // type
			durable,      // durable
			false,        // auto-deleted
			false,        // internal
			false,        // no-wait
			nil,          // arguments
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
			exchangeName,
			false,
			nil,
		)
		if err != nil {
			return fmt.Errorf("queue[%v]绑定exchange[%v]错误:%v", q.Name, exchangeName, err)
		}

		go consume(ch, q.Name, c)
	}

	return nil
}

func consume(ch *amqp.Channel, queueName string, c *ConsumersQueue) error {

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
			s := reflect.TypeOf(c.payLoadStruct)
			data := reflect.New(s)

			err := json.Unmarshal(d.Body, data.Interface())
			if err != nil {
				fmt.Printf("json Unmarshal[%v] is err[%v]", d.Body, err)
			}
			err = c.CallBack(data.Interface())
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
