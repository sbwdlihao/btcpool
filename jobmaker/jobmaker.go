package jobmaker

import (
	"github.com/btcpool/kafka"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil"
	"time"
	"github.com/Shopify/sarama"
	"github.com/ethereum/go-ethereum/logger/glog"
	"encoding/json"
	"github.com/btcpool/gbtmaker"
)

type JobMakerInterface interface {
	Init() error
	Run() error
	Stop() error
}

type JobMaker struct {
	kafkaProducer *kafka.KafkaProducer
	kafkaConsumer *kafka.KafkaConsumer
	poolPayoutAddr btcutil.Address
	done chan struct{}
}

func NewJobMaker(chainParams *chaincfg.Params, payoutAddress string, brokers string) (*JobMaker, error) {
	poolPayoutAddr, err := btcutil.DecodeAddress(payoutAddress, chainParams)
	if err != nil {
		return nil, err
	}
	return &JobMaker{
		kafkaProducer: kafka.NewKafkaProducer(brokers, kafka.KafkaTopicStratumJob),
		kafkaConsumer: kafka.NewKafkaConsumer(brokers, kafka.KafkaTopicRawGBT, 0),
		poolPayoutAddr: poolPayoutAddr,
		done: make(chan struct{}),
	}, nil
}

func (maker *JobMaker) Init() error {
	// deliver msg as soon as possible see sarama/produce_set.go readyToFlush
	maker.kafkaProducer.Conf.Producer.Flush.Frequency = 0
	maker.kafkaProducer.Conf.Producer.Flush.Bytes = 0
	maker.kafkaProducer.Conf.Producer.Flush.Messages = 0
	if err := maker.kafkaProducer.Setup(); err != nil {
		return err
	}

	maker.kafkaConsumer.Conf.Consumer.MaxWaitTime = 5 * time.Millisecond
	if err := maker.kafkaConsumer.Setup(); err != nil {
		return err
	}
	
	return nil
}

func (maker *JobMaker) Run() error {

	pc, err := maker.kafkaConsumer.Consume(sarama.OffsetNewest)
	if err != nil {
		return err
	}

	consume_done := make(chan struct{})
	go func() {
		for {
			select {
			case msg := <-pc.Messages():
				maker.consumeRawGbtMsg(msg)
			case consume_done:
				return
			}
		}
	}()

	<-maker.done
	consume_done <- struct {}{}
	return nil
}

func (maker *JobMaker) Stop() error {
	maker.done <- struct {}{}
	return nil
}

func (maker *JobMaker) consumeRawGbtMsg(msg *sarama.ConsumerMessage)  {
	glog.Info("received rawgbt message, len: ", len(msg.Value))
}

func (maker *JobMaker) addRawgbt(msgValue []byte) error {
	var rawgbt gbtmaker.RawGbt
	if err := json.Unmarshal(msgValue, &rawgbt); err != nil {
		return err
	}

	return nil
}