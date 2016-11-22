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
	"crypto/sha256"
	"github.com/Shopify/sarama"
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
	bitcoindRpcAddr string
	bitcoindRpcUser string
	bitcoindRpcPassword string
	kafkaBrokers string
	kRpcCallInterval time.Duration
	isCheckZmq bool
	kafkaProducer *kafka.KafkaProducer
	bitcoindRpcClient *btcrpcclient.Client
	lastCheckTIme time.Time
	checkTimeOut chan struct{}
	checkTimeUpdate chan struct{}
	done chan struct{}
}

func NewGbtMaker(gbtMakerConfig config.GbtMakerConfig, bitcoindConfig config.BitcoindConfig, kafkaBrokers string, bitcoindRpcClient *btcrpcclient.Client) *GbtMaker {
	return &GbtMaker{
		zmqBitcoindAddr: bitcoindConfig.Zmq_addr,
		bitcoindRpcAddr: bitcoindConfig.Rpc_addr,
		bitcoindRpcUser: bitcoindConfig.Rpc_user,
		bitcoindRpcPassword: bitcoindConfig.Rpc_password,
		kafkaBrokers: kafkaBrokers,
		kRpcCallInterval: time.Duration(gbtMakerConfig.Rpcinterval) * time.Second,
		isCheckZmq: gbtMakerConfig.Is_check_zmq,
		kafkaProducer: kafka.NewKafkaProducer(kafkaBrokers, kafka.KafkaTopicRawGBT),
		bitcoindRpcClient: bitcoindRpcClient,
		checkTimeOut: make(chan struct{}),
		checkTimeUpdate: make(chan struct{}),
		done: make(chan struct{}),
	}
}

func (maker *GbtMaker) Init() error {
	// set to 1 (0 is an illegal value here), deliver msg as soon as possible.
	maker.kafkaProducer.Conf.Producer.Flush.Frequency = 1 * time.Millisecond
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
	subscriber, err := zmq4.NewSocket(zmq4.SUB)
	if err != nil {
		return err
	}
	defer subscriber.Close()

	if err := subscriber.Connect(maker.zmqBitcoindAddr); err != nil {
		return err
	}
	subscriber.SetSubscribe(BitcoindZMQHashBlock)

	msg_chan := make(chan [][]byte, 1)
	recvMsg := func() {
		msg, err := subscriber.RecvMessageBytes(zmq4.DONTWAIT)
		if err != nil {
			glog.Error(err)
			return
		}
		msg_chan <- msg
	}
	handleMsg := func(msg [][]byte) {
		if len(msg) != 2 {
			glog.Error(fmt.Errorf("unknown message: %s", msg))
			return
		}
		msg_type := string(msg[0])
		if msg_type != BitcoindZMQHashTX {
			glog.Error(fmt.Errorf("invalid message type: %s", msg_type))
			return
		}
		glog.Info("bitcoind recv hashblock: ", hex.EncodeToString(msg[1]))
		if err := maker.submitRawGbtMsg(false); err != nil {
			glog.Error(err)
			return
		}
	}

	ticker := time.NewTicker(maker.kRpcCallInterval)
	check_time_done := make(chan struct{})
	checkTime := func() {
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
	}

	go recvMsg()
	go checkTime()

	runDone:
		for {
			select {
			case msg, ok := <- msg_chan:
				if ok {
					handleMsg(msg)
					go recvMsg()
				}
			case <- maker.done:
				check_time_done <- struct{}{}
				break runDone
			}
		}

	return nil
}

func (maker *GbtMaker) Stop() error {
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
	subscriber, err := zmq4.NewSocket(zmq4.SUB)
	if err != nil {
		return err
	}
	defer subscriber.Close()

	if err := subscriber.Connect(maker.zmqBitcoindAddr); err != nil {
		return err
	}
	subscriber.SetSubscribe(BitcoindZMQHashTX)
	// wait for hashtx
	glog.Info("check bitcoind zmq, waiting for zmq message 'hashtx'...")
	bytes, err := subscriber.RecvBytes(0)
	if err != nil {
		return err
	}
	msg_type := string(bytes)
	if msg_type != BitcoindZMQHashTX {
		return fmt.Errorf("invalid message type: %s", msg_type)
	}
	bytes, err = subscriber.RecvBytes(0)
	if err != nil {
		return err
	}
	hashtx := hex.EncodeToString(bytes)
	glog.Info("bitcoind zmq recv hashtx: ", hashtx)
	return nil
}

func (maker *GbtMaker) submitRawGbtMsg(checkTime bool) error {
	if checkTime && time.Now().Sub(maker.lastCheckTIme) < maker.kRpcCallInterval {
		glog.Warning("submit raw bgt msg too often")
		return nil
	}

	raw_gbt_msg, err := maker.makeRawGbtMsg()
	if err != nil {
		return err
	}
	maker.lastCheckTIme = time.Now()
	maker.checkTimeUpdate <- struct {}{}

	// submit to Kafka
	glog.Info("sumbit to Kafka, msg len: ", len(raw_gbt_msg))
	if err := maker.kafkaProducer.Produce(sarama.StringEncoder(raw_gbt_msg)); err != nil {
		return err
	}
	return nil
}

func (maker *GbtMaker) makeRawGbtMsg() (string, error) {
	request := &btcjson.TemplateRequest{}
	result, err := maker.bitcoindRpcClient.GetBlockTemplate(request)
	if err != nil {
		return "", err
	}
	template, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	template_string := string(template)
	hash := sha256.New()
	hash.Write(template)
	gbthash := hex.EncodeToString(hash.Sum(nil))
	glog.Infof("gbt height: %d, " +
		"prev_hash: %s, " +
		"coinbase_value: %v, " +
		"bits: %s, " +
		"mintime: %d, " +
		"version: %d|0x%x, " +
		"gbthash: %s",
		result.Height,
		result.PreviousHash,
		result.CoinbaseValue,
		result.Bits,
		result.MinTime,
		result.Version, result.Version,
		gbthash)
	msg, err := json.Marshal(map[string]string{
		"created_at": time.Now().Format(time.RFC3339),
		"block_template": template_string,
		"gbthash": gbthash,
	})
	if err != nil {
		return "", err
	}
	return string(msg), nil
}