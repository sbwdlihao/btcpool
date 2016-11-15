package kafka

import (
	"testing"
	"github.com/Shopify/sarama"
	"fmt"
	"strconv"
)

func TestKafkaProducer_Produce(t *testing.T) {
	producer := NewKafkaProducer("localhost:9092,localhost:9093,localhost:9094", "testKafka")
	defer func() {
		if err := producer.Close(); err != nil {
			fmt.Println(err)
		}
	}()
	if err := producer.Setup(); err != nil{
		fmt.Println(err)
		return
	}
	for i := 0; i < 3; i++ {
		producer.Produce(sarama.StringEncoder("this is a kafka test" + strconv.Itoa(i)))
	}
}

