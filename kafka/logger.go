package kafka

import (
	"github.com/golang/glog"
	"github.com/Shopify/sarama"
)

type KafkaLogger struct {

}

func (logger *KafkaLogger) Print(v ...interface{}) {
	glog.Info(v...)
}

func (logger *KafkaLogger) Printf(format string, v ...interface{}) {
	glog.Infof(format, v...)
}

func (logger *KafkaLogger) Println(v ...interface{}) {
	glog.Infoln(v...)
}

func init() {
	sarama.Logger = &KafkaLogger{}
}