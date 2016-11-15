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
)

func usage() {
	fmt.Fprintln(os.Stderr, "Usage:\n\tbtcpool -c \"config.yml\" -l \"log_dir\"\n")
}

type config struct {
	Testnet bool
	Kafka   struct {
			Brokers string
		}
	Sserver sserver.SserverConfig
	Users struct {
			List_id_api_url string
		}
	Pooldb database.MysqlConnectInfo
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
	cfg := config{}
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

	// todo try lock file

	var stratum_server *sserver.StratumServer

	// wait ctrl+c or terminal
	sigs := make(chan os.Signal, 1)
	sig_done := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	stopSignal := func() {
		signal.Stop(sigs)
		sig_done <- true
	}

	go func() {
		<-sigs
		fmt.Println("manual stop")
		if stratum_server != nil {
			if err := stratum_server.Stop(); err != nil {
				glog.Error(err)
			}
		}
		stopSignal()
	}()

	// create mysql connection and stratum server
	mysqlConnection := database.NewMysqlConnection(&cfg.Pooldb)
	stratum_server = sserver.NewStratumServer(&cfg.Sserver, cfg.Kafka.Brokers, cfg.Users.List_id_api_url, mysqlConnection)

	go func() {
		if err := stratum_server.Init(); err != nil {
			glog.Error(err)
			stopSignal()
			return
		}
		if err := stratum_server.Run(); err != nil {
			glog.Error(err)
			stopSignal()
		}
	}()

	<-sig_done
	glog.Flush()
}
