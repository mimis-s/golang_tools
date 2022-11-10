package rabbitmq

import (
	"fmt"
	"testing"
)

type TestStruct struct {
	NickName string
}

// 生产者
func testProducers(url string, exchangeName string, routingKey string, durable bool) {
	p, err := InitProducers(url, exchangeName, routingKey, durable)
	if err != nil {
		fmt.Printf("err:%v", err)
		return
	}

	data := &TestStruct{
		"123",
	}

	err = p.Publish(data)
	if err != nil {
		fmt.Printf("err:%v", err)
		return
	}
}

// 消费者
func testConsume(url string, exchangeName string, routingKey string, durable bool) {
	cQueue := make([]*ConsumersQueue, 0)
	cQueue = append(cQueue, &ConsumersQueue{
		RoutingKey:    routingKey,
		payLoadStruct: &TestStruct{},
		CallBack:      callBack,
	})

	RegisterConsumers(url, durable, exchangeName, cQueue)
}

func TestRabbitMQ(t *testing.T) {
	url := "amqp://dev:dev123@localhost:5672/"
	exchangeName := "test.producers"
	routingKey := "info"
	durable := false

	// 消费者
	testConsume(url, exchangeName, routingKey, durable)

	// 生产者
	testProducers(url, exchangeName, routingKey, durable)
	select {}
}

func callBack(payload interface{}) error {
	data := payload.(map[string]string)
	fmt.Printf("接收到:%v\n", data)
	return nil
}
