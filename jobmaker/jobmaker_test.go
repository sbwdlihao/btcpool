package jobmaker

import (
	"testing"
	"flag"
	"github.com/btcpool/config"
	"gopkg.in/redis.v5"
	"github.com/btcsuite/btcd/chaincfg"
	"os"
	"github.com/golang/glog"
	"sync"
	"time"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcpool/util"
	"github.com/btcpool/gbtmaker"
	"encoding/json"
)

var maker *JobMaker

func TestMain(t *testing.M) {
	flag.Parse()
	flag.Lookup("logtostderr").Value.Set("true")
	jobConf := &config.JobMakerConfig{
		Pool_coinbase: "/BTC.COM/",
		Block_version: 2,
	}
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		Password: "",
		DB: 0,
	})
	maker, _ = NewJobMaker(jobConf, redisClient, &chaincfg.MainNetParams, "3NA8hsjfdgVkmmVS9moHmkZsVCoLxUkvvv", "localhost:9092,localhost:9093,localhost:9094")
	if err := maker.Init(); err != nil {
		glog.Error(err)
		return
	}
	os.Exit(t.Run())
}

func TestConsumeRawGbtMsg(t *testing.T)  {
	var cv int64 = 5000000000
	bk := &btcjson.GetBlockTemplateResult{
		Bits: "207fffff",
		CurTime: 1481205398,
		Height: 442457,
		PreviousHash: "53dae1de0cf115cbaec8d48893b891231f3907ff5878ede3cde6959000f52076",
		SigOpLimit: 20000,
		SizeLimit: 1000000,
		Transactions: []btcjson.GetBlockTemplateResultTx{
			{
				Data: "01000000013d6171c20312373e7e2bb7ea07d3f2176abc1dd114af92666e0570c8f7192177000000004948304502210092b7951fde9c5fa4161abb22265a9c7647d3774a22874ff251d9fa99a21e894c022010d6c2f0eae1d701bda97ff9cbfe29f55a5e982b337d5ae9a3d8379693d7f36501feffffff0200021024010000001976a914f8b500f578e602cd9021e6809ac9f22687a363b888ac00e1f505000000001976a914991d0461842b386b74616f79011979f4e666ff3788ac42000000",
				Hash: "605adc0cfd155da6c9902753123160bb7a8934d7105d6d346eb619f39098195f",
				Depends: []int64{},
				Fee: 3840,
				SigOps: 2,
			},
			{
				Data: "01000000015f199890f319b66e346d5d10d734897abb603112532790c9a65d15fd0cdc5a60000000006a47304402201dba47fb8ab224f0eb8192975d54118f897e47fe433e4921040281d71ecc570702205ddd3ff8c5849131829ca60a07dff80e2a859123a6f14e3924aae11ebcb653df0121026b8aa8e9305fd900d487b71eb4467f35c680da6d5f9244b60a265574ddd02d5efeffffff026c0f1a1e010000001976a914bf1fd67e3bfa4fa7257e0adfbc068b19d95da86488ac00e1f505000000001976a914991d0461842b386b74616f79011979f4e666ff3788ac66000000",
				Hash: "6aa04f0d94d2a7965c3d036eb03a0e90ef7cc62aa9e4bb80efa72b9796f0d513",
				Depends: []int64{1},
				Fee: 4500,
				SigOps: 2,
			},
		},
		Version: 536870912,
		CoinbaseAux: &btcjson.GetBlockTemplateResultAux{
			Flags: "0a2f454231362f4144342f",
		},
		CoinbaseValue: &cv,
		LongPollID: "53dae1de0cf115cbaec8d48893b891231f3907ff5878ede3cde6959000f5207623",
		Target: "7fffff0000000000000000000000000000000000000000000000000000000000",
		MinTime: 1480851746,
		Mutable: []string{"time","transactions","prevblock"},
		NonceRange: "00000000ffffffff",
		Capabilities: []string{"proposal"},
	}
	hash, _ := util.HashBlockTemplateResult(bk)
	rawGbt := &gbtmaker.RawGbt{
		CreatedAt: time.Now(),
		BlockTemplate: bk,
		Hash: hash,
	}
	value, _ := json.Marshal(rawGbt)
	if err := maker.consumeRawGbtMsg(value); err != nil {
		glog.Error(err)
	}
	maker.kafkaProducer.Close()
}

func TestJobMaker_Run(t *testing.T) {
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
