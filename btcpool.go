package main

import (
	"os"
	"os/signal"
	"syscall"
	"fmt"
	"flag"
	"io/ioutil"
	"gopkg.in/yaml.v2"
	"github.com/golang/glog"
	"github.com/btcpool/database"
	"github.com/btcpool/sserver"
	"github.com/btcpool/config"
	"github.com/btcrpcclient"
	"github.com/btcpool/gbtmaker"
)

func usage() {
	fmt.Fprintln(os.Stderr, "Usage:\n\tbtcpool -c \"config.yml\" -l \"log_dir\"\n")
}

func main() {
	args := os.Args
	if len(args) <= 1 {
		usage()
		os.Exit(1)
	}

	c := flag.String("c", "config.yml", "config file")
	l := flag.String("l", "log_dir", "log directory")
	h := flag.Bool("h", false, "help")

	flag.Parse()

	if *h {
		usage()
		os.Exit(0)
	}

	flag.Lookup("log_dir").Value.Set(*l)
	// Log messages at a level >= this flag are automatically sent to
	// stderr in addition to log files.
	flag.Lookup("stderrthreshold").Value.Set("3") // 3: FATAL
	// no max_log_size, ogbuflevel and stop_logging_if_full_disk in glob of go version

	// Read the file. If there is an error, report it and exit.
	bytes, err := ioutil.ReadFile(*c)
	if err != nil {
		glog.Error("read file failed: ", err)
		os.Exit(1)
	}
	cfg := config.BtcPoolConfig{}
	err = yaml.Unmarshal(bytes, &cfg)
	if err != nil {
		glog.Error("sserver config parse failed: ", err)
		os.Exit(1)
	}

	// check if we are using testnet3
	if cfg.Testnet {
		// todo set chain network to testnet
		glog.Warning("using bitcoin testnet3")
	} else {
		// todo set chain network to main
	}

	if !(cfg.Sserver.Id >= 1 && cfg.Sserver.Id <= 255) {
		glog.Error("invalid server id, range: [1, 255]")
		os.Exit(1)
	}

	btc_rpc_client_config := &btcrpcclient.ConnConfig{
		Host: cfg.Bitcoind.Rpc_addr,
		User: cfg.Bitcoind.Rpc_user,
		Pass: cfg.Bitcoind.Rpc_password,
		HTTPPostMode: true, // Bitcoin core only supports HTTP POST mode
		DisableTLS: true, // Bitcoin core does not provide TLS by default
	}
	btc_rpc_client, err := btcrpcclient.New(btc_rpc_client_config, nil)
	if err != nil {
		glog.Error("bitcoind rpc connect failed: ", err)
		os.Exit(1)
	}

	// todo try lock file

	var stratum_server *sserver.StratumServer
	var gbt_maker *gbtmaker.GbtMaker



	// create mysql connection and stratum server
	mysql_connection := database.NewMysqlConnection(cfg.Pooldb)
	stratum_server = sserver.NewStratumServer(cfg.Sserver, cfg.Kafka.Brokers, cfg.Users.List_id_api_url, mysql_connection)
	if err := stratum_server.Init(); err != nil {
		glog.Error(err)
		os.Exit(1)
	}
	if err := stratum_server.Run(); err != nil {
		glog.Error(err)
		os.Exit(1)
	}

	gbt_maker = gbtmaker.NewGbtMaker(cfg.GbtMaker, cfg.Bitcoind, cfg.Kafka.Brokers, btc_rpc_client)
	if err := gbt_maker.Init(); err != nil {
		glog.Error(err)
		os.Exit(1)
	}
	go func() {
		if err := gbt_maker.Run(); err != nil {
			glog.Error(err)
		}
	}()


	// wait ctrl+c or terminal
	sigs := make(chan os.Signal, 1)
	sig_done := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs
		if stratum_server != nil {
			if err := stratum_server.Stop(); err != nil {
				glog.Error(err)
			}
		}
		signal.Stop(sigs)
		sig_done <- true
	}()

	<-sig_done
	glog.Flush()
}
