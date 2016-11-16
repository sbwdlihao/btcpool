package config

type BtcPoolConfig struct {
	Testnet  bool
	Kafka    struct {
			Brokers string
		}
	Sserver  SserverConfig
	Users    struct {
			List_id_api_url string
		}
	Pooldb   MysqlConnectConfig
	GbtMaker GbtMakerConfig
	Bitcoind BitcoindConfig
}

type SserverConfig struct {
	Ip                          string
	Port                        uint32
	Id                          uint8
	File_last_notify_time       string
	Enable_simulator            bool
	Enable_submit_invalid_block bool
}

type MysqlConnectConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	Dbname   string
}

type GbtMakerConfig struct {
	Rpcinterval uint
	Is_check_zmq bool
}

type BitcoindConfig struct {
	Zmq_addr string
	Rpc_addr string
	Rpc_user string
	Rpc_password string
}
