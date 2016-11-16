package gbtmaker

import (
	"testing"
	"fmt"
)

func TestGbtMaker_CheckBitcoindZMQ(t *testing.T) {
	maker := &GbtMaker{
		zmqBitcoindAddr: "tcp://127.0.0.1:28332",
	}
	if err := maker.CheckBitcoindZMQ(); err != nil {
		fmt.Println(err)
	}
	maker.submitRawGbtMsg()
}

