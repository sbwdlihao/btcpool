package kafka

import (
	"testing"
	"github.com/Shopify/sarama"
	"fmt"
	"os/signal"
	"os"
	"syscall"
	"flag"
)

// go test -v github.com/btcpool/kafka -run ^TestKafkaProducer_Produce$ -args world1
func TestKafkaProducer_Produce(t *testing.T) {
	flag.Parse()
	msg := flag.Arg(0)
	producer := NewKafkaProducer("localhost:9092,localhost:9093,localhost:9094", "testKafka2")
	defer func() {
		if err := producer.Close(); err != nil {
			fmt.Println(err)
		}
	}()
	if err := producer.Setup(); err != nil{
		fmt.Println(err)
		return
	}

	fmt.Println("produce msg: ", msg) // produce msg:  world1
	producer.Produce(sarama.StringEncoder(msg))
}


func TestKafkaConsumer_Consume(t *testing.T) {
	consumer := NewKafkaConsumer("localhost:9092,localhost:9093,localhost:9094", "testKafka2", 0)
	if err := consumer.Setup(); err != nil {
		fmt.Println(err)
		return
	}
	defer func() {
		if err := consumer.Close(); err != nil {
			fmt.Println(err)
		}
	}()
	// 如果kafka中消息的队列是0->hi0,1->hi1,2->hi2
	// offsetnewest读取最新的消息，OffsetOldest是从最老的消息开始读(hi0)
	// 如果指定offset为1，则从第1条消息(hi1)开始读取
	pc, err := consumer.Consume(sarama.OffsetNewest)
	if err != nil {
		fmt.Println(err)
		return
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT)

	for {
		select {
		case <-sigs:
			fmt.Println("sig exit")
			return
		case msg := <-pc.Messages():
			fmt.Println("msg: ", string(msg.Value))
		}
	}
}
