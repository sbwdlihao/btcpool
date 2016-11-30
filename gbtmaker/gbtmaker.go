package gbtmaker

import (
	"github.com/btcpool/kafka"
	"github.com/btcpool/config"
	"time"
	"github.com/btcrpcclient"
	"github.com/golang/glog"
	"errors"
	"github.com/pebbe/zmq4"
	"fmt"
	"encoding/hex"
	"github.com/btcsuite/btcd/btcjson"
	"encoding/json"
	"github.com/Shopify/sarama"
	"github.com/btcpool/util"
)

const (
	//
	// bitcoind zmq pub msg type: "hashblock", "hashtx", "rawblock", "rawtx"
	//
	BitcoindZMQHashBlock string = "hashblock"
	BitcoindZMQHashTX string = "hashtx"

	//
	// namecoind zmq pub msg type: "hashblock", "hashtx", "rawblock", "rawtx"
	//
	NamecoindZMQHashBlock string = "hashblock"
	NamecoindZMQHashTX string = "hashtx"
)

type GbtMakerInterface interface {
	Init() error
	Stop() error
	Run() error
}

type GbtMaker struct {
	zmqBitcoindAddr string
	kRpcCallInterval time.Duration
	isCheckZmq bool
	kafkaProducer *kafka.KafkaProducer
	bitcoindRpcClient *btcrpcclient.Client
	lastCheckTime time.Time
	checkTimeUpdate chan struct{}
	done chan struct{}
}

func NewGbtMaker(gbtMakerConfig config.GbtMakerConfig, zmqBitcoindAddr string, kafkaBrokers string, bitcoindRpcClient *btcrpcclient.Client) *GbtMaker {
	return &GbtMaker{
		zmqBitcoindAddr: zmqBitcoindAddr,
		kRpcCallInterval: time.Duration(gbtMakerConfig.Rpcinterval) * time.Second,
		isCheckZmq: gbtMakerConfig.Is_check_zmq,
		kafkaProducer: kafka.NewKafkaProducer(kafkaBrokers, kafka.KafkaTopicRawGBT),
		bitcoindRpcClient: bitcoindRpcClient,
		checkTimeUpdate: make(chan struct{}),
		done: make(chan struct{}),
	}
}

func (maker *GbtMaker) Init() error {
	// deliver msg as soon as possible see sarama/produce_set.go readyToFlush
	maker.kafkaProducer.Conf.Producer.Flush.Frequency = 0
	maker.kafkaProducer.Conf.Producer.Flush.Bytes = 0
	maker.kafkaProducer.Conf.Producer.Flush.Messages = 0
	if err := maker.kafkaProducer.Setup(); err != nil {
		return err
	}

	if err := maker.checkBitcoind(); err != nil {
		return err
	}

	if maker.isCheckZmq {
		if err := maker.checkBitcoindZMQ(); err != nil {
			return err
		}
	}
	return nil
}

func (maker *GbtMaker) Run() error {
	glog.Info("Gbt Running ......")

	recv_zmq_done := make(chan struct{})
	go func() {
		for {
			select {
			case err := <-maker.recvZmqMessage(BitcoindZMQHashBlock):
				if err != nil {
					glog.Error(err)
					break // break select statement
				}
				if err := maker.submitRawGbtMsg(false); err != nil {
					glog.Error(err)
				}
			case <-recv_zmq_done:
				glog.Info("recv_zmq_done")
				return
			}
		}

	}()

	ticker := time.NewTicker(maker.kRpcCallInterval)
	check_time_done := make(chan struct{})
	go func() {
		for {
			select {
			case <- ticker.C:
				maker.submitRawGbtMsg(true)
			case <- maker.checkTimeUpdate:
				ticker.Stop()
				ticker = time.NewTicker(maker.kRpcCallInterval)
			case <- check_time_done:
				ticker.Stop()
				return
			}
		}
	}()

	<- maker.done
	recv_zmq_done <- struct {}{}
	check_time_done <- struct{}{}
	return nil
}

func (maker *GbtMaker) Stop() error {
	glog.Info("GbtMaker stop")
	maker.done <- struct {}{}
	return nil
}

func (maker *GbtMaker) checkBitcoind() error {
	info, err := maker.bitcoindRpcClient.GetInfo()
	if err != nil {
		return err
	}
	glog.Infof("bitcoind getinfo: %+v", info)

	if info.Connections <= 0 {
		return errors.New("bitcoind connections is zero")
	}
	return nil
}

// call this method will block go routine
func (maker *GbtMaker) checkBitcoindZMQ() error {
	return <-maker.recvZmqMessage(BitcoindZMQHashTX)
}

func (maker *GbtMaker) recvZmqMessage(msgType string) chan error {
	result := make(chan error)
	go func() {
		// zmq socket is not thread safe
		subscriber, err := zmq4.NewSocket(zmq4.SUB)
		if err != nil {
			result <- err
			return
		}
		defer subscriber.Close()

		if err := subscriber.Connect(maker.zmqBitcoindAddr); err != nil {
			result <- err
			return
		}
		subscriber.SetSubscribe(msgType)
		glog.Infof("waiting for zmq message '%s'...", msgType)
		bytes, err := subscriber.RecvMessageBytes(0)
		if err != nil {
			result <- err
			return
		}
		recvType := string(bytes[0])
		if recvType != msgType {
			result <- fmt.Errorf("invalid message type: %s", recvType)
			return
		}
		hash := hex.EncodeToString(bytes[1])
		glog.Infof("bitcoind zmq recv %s: %s", msgType, hash)
		result <- nil
		close(result)
	}()
	return result
}

func (maker *GbtMaker) submitRawGbtMsg(checkTime bool) error {
	if checkTime && time.Now().Sub(maker.lastCheckTime) < maker.kRpcCallInterval {
		glog.Warning("submit raw bgt msg too often")
		return nil
	}

	raw_gbt_msg, err := maker.makeRawGbtMsg()
	if err != nil {
		return err
	}
	// submit to Kafka
	glog.Info("sumbit to Kafka, msg len: ", len(raw_gbt_msg))
	if err := maker.kafkaProducer.Produce(sarama.ByteEncoder(raw_gbt_msg)); err != nil {
		return err
	}
	go func() {
		maker.lastCheckTime = time.Now()
		maker.checkTimeUpdate <- struct {}{}
	}()
	return nil
}

func (maker *GbtMaker) makeRawGbtMsg() ([]byte, error) {
	request := &btcjson.TemplateRequest{}
	result, err := maker.bitcoindRpcClient.GetBlockTemplate(request)
	if err != nil {
		return "", err
	}
	gbthash, err := util.HashBlockTemplateResult(result)
	if err != nil {
		return "", err
	}
	glog.Infof("gbt height: %d, " +
		"prev_hash: %s, " +
		"coinbase_value: %v, " +
		"bits: %s, " +
		"mintime: %d, " +
		"version: %d|0x%x, " +
		"gbthash: %s",
		result.Height,
		result.PreviousHash,
		*result.CoinbaseValue,
		result.Bits,
		result.MinTime,
		result.Version, result.Version,
		gbthash)
	msg, err := json.Marshal(RawGbt{
		CreatedAt: time.Now(),
		BlockTemplate: result,
		Hash: gbthash,
	})
	if err != nil {
		return "", err
	}
	return msg, nil
}

type RawGbt struct {
	CreatedAt time.Time
	BlockTemplate btcjson.GetBlockTemplateResult
	Hash string
}