package gbtmaker

import (
	"testing"
	"os"
	"github.com/golang/glog"
	"flag"
	"github.com/btcrpcclient"
	"github.com/btcpool/config"
	"time"
	"sync"
)

var maker *GbtMaker

func TestMain(m *testing.M) {
	flag.Parse()
	flag.Lookup("logtostderr").Value.Set("true")
	btc_rpc_client_config := &btcrpcclient.ConnConfig{
		Host: "127.0.0.1:19001",
		User: "admin1",
		Pass: "123",
		HTTPPostMode: true,
		DisableTLS: true,
	}
	btc_rpc_client, err := btcrpcclient.New(btc_rpc_client_config, nil)
	if err != nil {
		glog.Error("bitcoind rpc connect failed: ", err)
		os.Exit(1)
	}
	gbkmakerConfig := config.GbtMakerConfig{
		Rpcinterval: 5,
		Is_check_zmq: true,
	}
	maker = NewGbtMaker(gbkmakerConfig, "tcp://127.0.0.1:28332", "localhost:9092,localhost:9093,localhost:9094", btc_rpc_client)
	code := m.Run()
	glog.Flush()
	os.Exit(code)
}

func TestCheckBitcoind(t *testing.T)  {
	if err := maker.checkBitcoind(); err != nil {
		glog.Error(err)
	}
}

func TestCheckBitcoindZMQ(t *testing.T) {
	if err := maker.checkBitcoindZMQ(); err != nil {
		glog.Error(err)
	}
}

func TestMakeRawGbtMsg(t *testing.T)  {
	msg, err := maker.makeRawGbtMsg()
	if err != nil {
		glog.Error(err)
	} else {
		glog.Info(msg)
	}
}

func TestGbtMaker_Run(t *testing.T) {
	if err := maker.Init(); err != nil {
		glog.Error(err)
		return
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		if err := maker.Run(); err != nil {
			glog.Error(err)
		}
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		time.Sleep(60 * time.Second)
		maker.Stop()
		wg.Done()
	} ()

	wg.Wait()
}

