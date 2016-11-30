package kafka

import (
	"github.com/Shopify/sarama"
	"time"
	"strings"
	"github.com/golang/glog"
	"errors"
)

const (
	ClientId string = "btcpool"
	KafkaTopicRawGBT string = "RawGbt"
	KafkaTopicStratumJob string = "StratumJob"
	KafkaTopicSolvedShare string = "SolvedShare"
	KafkaTopicShareLog string = "ShareLog"
	KafkaTopicCommonEvents = "CommonEvents"

	KafkaTopicNMCSolvedShare = "NMCSolvedShare"
)

type KafkaProducerInterface interface {
	Setup() error
	Produce(payload sarama.Encoder) error
	Close() error
}

type KafkaConsumerInterface interface {
	Setup() error
	Consume(offset int64) (sarama.PartitionConsumer, error)
	Close() error
}

type KafkaProducer struct {
	brokers        	[]string
	topic    	string
	producer 	sarama.AsyncProducer
	Conf     	*sarama.Config
}

type KafkaConsumer struct {
	brokers        []string
	topic          string
	partition      int32
	consumer       sarama.Consumer
	Conf           *sarama.Config
}

func NewKafkaProducer(brokers string, topic string) *KafkaProducer {
	producer := &KafkaProducer{
		brokers: strings.Split(brokers, ","),
		topic: topic,
	}
	producer.Conf = sarama.NewConfig()
	producer.Conf.ClientID = ClientId
	producer.Conf.Producer.MaxMessageBytes = 20000000
	producer.Conf.Producer.Compression = sarama.CompressionSnappy
	producer.Conf.Producer.Flush.MaxMessages = 100000
	producer.Conf.Producer.Flush.Frequency = 1000 * time.Millisecond
	producer.Conf.Producer.Flush.Messages = 1000

	return producer
}

func (producer *KafkaProducer) Setup() error {
	if producer.producer != nil {
		return errors.New("producer can only set up once")
	}
	// create producer
	k_producer, err := sarama.NewAsyncProducer(producer.brokers, producer.Conf)
	if err == nil {
		producer.producer = k_producer
		// we must call producer.producer.Close() before quit
		go func() {
			for err := range producer.producer.Errors() {
				glog.Error(err)
			}
		}()
	}

	return err
}

func (producer *KafkaProducer) Produce(payload sarama.Encoder) error {
	if producer.producer == nil {
		return errors.New("producer doesn't set up or closed already")
	}
	message := &sarama.ProducerMessage{Topic: producer.topic, Key: nil, Value: payload}
	producer.producer.Input() <- message
	return nil
}

func (producer *KafkaProducer) Close() error {
	if producer.producer == nil {
		return errors.New("producer doesn't set up or closed already")
	}
	if err := producer.producer.Close(); err != nil {
		return err
	}
	producer.producer = nil
	return nil
}

func NewKafkaConsumer(brokers string, topic string, partition int32) *KafkaConsumer{
	consumer := &KafkaConsumer{
		brokers:strings.Split(brokers, ","),
		topic:topic,
		partition:partition,
	}
	consumer.Conf = sarama.NewConfig()
	consumer.Conf.ClientID = ClientId
	consumer.Conf.Consumer.Fetch.Default = 20000000
	consumer.Conf.Consumer.MaxWaitTime = 10 * time.Millisecond

	return consumer
}

func (consumer *KafkaConsumer) Setup() error {
	if consumer.consumer != nil {
		return errors.New("consumer can only set up once")
	}
	k_consumer, err := sarama.NewConsumer(consumer.brokers, consumer.Conf)
	if err == nil {
		consumer.consumer = k_consumer
	}
	return err
}

func (consumer *KafkaConsumer) Consume(offset int64) (sarama.PartitionConsumer, error) {
	if consumer.consumer == nil {
		return nil, errors.New("consumer doesn't set up or closed already")
	}
	return consumer.consumer.ConsumePartition(consumer.topic, consumer.partition, offset)
}

func (consumer *KafkaConsumer) Close() error {
	if consumer.consumer == nil {
		return errors.New("consumer doesn't set up or closed already")
	}
	if err := consumer.consumer.Close(); err != nil {
		return err
	}
	consumer.consumer = nil
	return nil
}