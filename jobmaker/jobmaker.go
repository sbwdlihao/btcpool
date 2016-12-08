package jobmaker

import (
	"encoding/json"
	"github.com/Shopify/sarama"
	"github.com/btcpool/gbtmaker"
	"github.com/btcpool/kafka"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil"
	"github.com/golang/glog"
	"time"
	"gopkg.in/redis.v5"
	"github.com/btcpool/config"
	"github.com/btcpool/sserver"
	"errors"
)

type JobMakerInterface interface {
	Init() error
	Run() error
	Stop() error
}

type JobMaker struct {
	config *config.JobMakerConfig
	kafkaProducer       *kafka.KafkaProducer
	kafkaRawGbtConsumer *kafka.KafkaConsumer
	redis *redis.Client
	poolPayoutAddr      btcutil.Address
	latestGbtHeight int64
	done                chan struct{}
}

func NewJobMaker(config *config.JobMakerConfig, redisClient *redis.Client, chainParams *chaincfg.Params, payoutAddress string, brokers string) (*JobMaker, error) {
	poolPayoutAddr, err := btcutil.DecodeAddress(payoutAddress, chainParams)
	if err != nil {
		return nil, err
	}
	return &JobMaker{
		config: config,
		redis: redisClient,
		kafkaProducer:       kafka.NewKafkaProducer(brokers, kafka.KafkaTopicStratumJob),
		kafkaRawGbtConsumer: kafka.NewKafkaConsumer(brokers, kafka.KafkaTopicRawGBT, 0),
		poolPayoutAddr:      poolPayoutAddr,
		latestGbtHeight: 0,
		done:                make(chan struct{}),
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

	//maker.kafkaRawGbtConsumer.Conf.Consumer.MaxWaitTime = 5 * time.Millisecond
	if err := maker.kafkaRawGbtConsumer.Setup(); err != nil {
		return err
	}

	return nil
}

func (maker *JobMaker) Run() error {
	pc, err := maker.kafkaRawGbtConsumer.Consume(sarama.OffsetNewest)
	if err != nil {
		return err
	}

	consume_done := make(chan struct{})
	go func() {
		for {
			select {
			case msg := <-pc.Messages():
				if err := maker.consumeRawGbtMsg(msg.Value); err != nil {
					glog.Error(err)
				}
			case <-consume_done:
				return
			}
		}
	}()

	<-maker.done
	consume_done <- struct{}{}
	return nil
}

func (maker *JobMaker) Stop() error {
	glog.Info("JobMaker stop")
	maker.done <- struct{}{}
	if err := maker.kafkaProducer.Close(); err != nil {
		glog.Error(err)
	}
	if err := maker.kafkaRawGbtConsumer.Close(); err != nil {
		glog.Error(err)
	}
	return nil
}

func (maker *JobMaker) consumeRawGbtMsg(value []byte) error {
	glog.Info("received rawgbt message, len: ", len(value))
	var rawgbt gbtmaker.RawGbt
	if err := json.Unmarshal(value, &rawgbt); err != nil {
		return err
	}
	
	if err := maker.checkRawGbt(&rawgbt); err != nil {
		return err
	}
	
	return maker.sendStratumJob(&rawgbt)
}

func (maker *JobMaker) checkRawGbt(rawgbt *gbtmaker.RawGbt) error {
	// 检查创建时间
	now := time.Now()
	if now.Add(-time.Minute).After(rawgbt.CreatedAt) {
		glog.Warningf("rawgbt diff time is more than 60, ingore it. now = %d, create at = %d", now, rawgbt.CreatedAt)
		return nil
	}
	if now.Add(-3 * time.Second).After(rawgbt.CreatedAt) {
		glog.Warning("rawgbt diff time is too large: ", now.Sub(rawgbt.CreatedAt).Seconds(), " seconds")
	}

	// 检查height
	if rawgbt.BlockTemplate.Height < maker.latestGbtHeight {
		glog.Warningf("gbt height: %d lower latest: %d", rawgbt.BlockTemplate.Height, maker.latestGbtHeight)
		return nil
	}

	// 检查hash
	_, err := maker.redis.Get(rawgbt.Hash).Result()
	if err == nil {
		return errors.New("duplicate gbt hash: " + rawgbt.Hash)
	}
	if err != redis.Nil {
		return err
	}
	maker.redis.Set(rawgbt.Hash, true, time.Hour)
	maker.latestGbtHeight = rawgbt.BlockTemplate.Height
	glog.Infof("receive rawgbt, height: %d, gbthash: %s, gbtTime(UTC): %d", rawgbt.BlockTemplate.Height, rawgbt.Hash, rawgbt.CreatedAt.Unix())
	return nil
}

func (maker *JobMaker) sendStratumJob(rawgbt *gbtmaker.RawGbt) error {
	job, err := sserver.InitFromGbt(rawgbt, maker.config.Pool_coinbase, maker.poolPayoutAddr, maker.config.Block_version)
	if err != nil {
		return err
	}
	glog.Info("init from gbt, job id = ", job.JobId)
	jobBytes, err := json.Marshal(job)
	if err != nil {
		return err
	}
	glog.Infof("submit to kafka topic = %s, msg len = %d", kafka.KafkaTopicStratumJob, len(jobBytes))
	return maker.kafkaProducer.Produce(sarama.ByteEncoder(jobBytes))
}