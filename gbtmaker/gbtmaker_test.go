package gbtmaker

import (
	"testing"
	"os"
	"github.com/golang/glog"
	"flag"
	"github.com/btcrpcclient"
)

var maker *GbtMaker

func TestMain(m *testing.M) {
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
	maker = &GbtMaker{
		zmqBitcoindAddr: "tcp://127.0.0.1:28332",
		bitcoindRpcClient: btc_rpc_client,
	}
	os.Exit(m.Run())
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

