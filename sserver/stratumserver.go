package sserver

import (
	"sync/atomic"
	"errors"
	"github.com/golang/glog"
	"github.com/Shopify/sarama"
	"github.com/btcpool/database"
	"github.com/btcpool/kafka"
)

type SserverConfig struct {
	Ip                          string
	Port                        uint32
	Id                          uint8
	File_last_notify_time       string
	Enable_simulator            bool
	Enable_submit_invalid_block bool
}

type StratumServerInterface interface {
	Init() error
	Stop() error
	Run() error
}

type StratumServer struct {
	running int32
	config *SserverConfig
	kafkaBrokers string
	userAPIUrl string
	mysqlConnection *database.MysqlConnection

	// kafka producers
	kafkaProducerShareLog *kafka.KafkaProducer
	kafkaProducerSolvedShare *kafka.KafkaProducer
	kafkaProducerNamecoinSolvedShare *kafka.KafkaProducer
	kafkaProducerCommonEvents *kafka.KafkaProducer

	JobRepository *JobRepository
}

func NewStratumServer(config *SserverConfig, kafkaBrokers string, userAPIUrl string, mysqlConnection *database.MysqlConnection) *StratumServer {
	return &StratumServer{
		config: config,
		kafkaBrokers: kafkaBrokers,
		userAPIUrl: userAPIUrl,
		mysqlConnection: mysqlConnection,
	}
}

func (server *StratumServer) Init() error {
	if server.config.Enable_simulator {
		glog.Warning("Simulator is enabled, all share will be accepted")
	}
	if server.config.Enable_submit_invalid_block {
		glog.Warning("submit invalid block is enabled, all block will be submited")
	}

	server.kafkaProducerSolvedShare = kafka.NewKafkaProducer(server.kafkaBrokers, kafka.KafkaTopicSolvedShare)
	server.kafkaProducerNamecoinSolvedShare = kafka.NewKafkaProducer(server.kafkaBrokers, kafka.KafkaTopicNMCSolvedShare)
	server.kafkaProducerShareLog = kafka.NewKafkaProducer(server.kafkaBrokers, kafka.KafkaTopicShareLog)
	server.kafkaProducerCommonEvents = kafka.NewKafkaProducer(server.kafkaBrokers, kafka.KafkaTopicCommonEvents)

	server.JobRepository = NewJobRepository(server.kafkaBrokers, server.config.File_last_notify_time, server)
	if err := server.JobRepository.SetupThreadConsume(); err != nil {
		return err
	}


	return nil
}

func (server *StratumServer) Stop() error {
	return nil
}

func (server *StratumServer) Run() error {
	if atomic.LoadInt32(&server.running) == 1 {
		return errors.New("server is running")
	}
	atomic.AddInt32(&server.running, 1)
	return nil
}

type JobRepositoryInterface interface {
	Stop()
	SetupThreadConsume() error
	MarkAllJobsAsStale()
}

type JobRepository struct {
	kafkaConsumer *kafka.KafkaConsumer
	server *StratumServer
	fileLastNotifyTime string
	done chan struct{}
}

func NewJobRepository(kafkaBrokers string, fileLastNotifyTime string, server *StratumServer) *JobRepository {
	return &JobRepository{
		kafkaConsumer: kafka.NewKafkaConsumer(kafkaBrokers, kafka.KafkaTopicStratumJob, 0),
		server: server,
		fileLastNotifyTime: fileLastNotifyTime,
		done: make(chan struct{}),
	}
}

func (jobRepository *JobRepository) Stop() {
	jobRepository.done <- struct {}{}
	glog.Info("stop job repository")
}

func (jobRepository *JobRepository) SetupThreadConsume() error {
	jobRepository.kafkaConsumer.Setup()
	partitionConsumer, err := jobRepository.kafkaConsumer.Consume(sarama.OffsetNewest)
	if err != nil {
		return err
	}
	go func() {
		glog.Info("start job repository consume thread")
		out:
			for {
				select {
				case message := <- partitionConsumer.Messages():
					jobRepository.consumeStratumJob(message)
					jobRepository.checkAndSendMiningNotify()
					jobRepository.tryCleanExpiredJobs()
				case <- jobRepository.done:
					break out
				}
			}
		glog.Info("stop job repository consume thread")
	}()
	return nil
}

func (jobRepository *JobRepository) MarkAllJobsAsStale() {

}

func (jobRepository *JobRepository) consumeStratumJob(message *sarama.ConsumerMessage) {
}

func (jobRepository *JobRepository) checkAndSendMiningNotify() {

}

func (jobRepository *JobRepository) tryCleanExpiredJobs() {

}

type StratumJobExInterface interface {
	MarkStale()
	IsStale()
}