package main

import (
	"flag"
	"strings"
	"time"

	logrus "github.com/Sirupsen/logrus"
	nats "github.com/nats-io/go-nats"
	moleculer "github.com/roytan883/moleculer-go"

	"github.com/xlab/closer"
)

const (
	//AppName ...
	AppName = "ws-connector"
)

func init() {
	initLog()
}

var log *logrus.Logger

func initLog() {
	log = logrus.New()
	log.Formatter = &logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "01-02 15:04:05.000000",
	}
	log.WithFields(logrus.Fields{"package": AppName, "file": "main"})
}

// NOTE: Use tls scheme for TLS, e.g. nats-req -s tls://demo.nats.io:4443 foo hello
// ws-connector -s nats://192.168.1.69:12008
// ws-connector -s nats://127.0.0.1:4222
func usage() {
	log.Fatalf("Usage: ws-connector [-s server (%s)] \n", nats.DefaultURL)
}

var pBroker *moleculer.ServiceBroker

//debug: go run .\main.go .\ws-connector.go
func main() {
	closer.Bind(cleanupFunc)
	log.Infof("Start %s ...\n", AppName)

	//get NATS server host
	var urls = flag.String("s", nats.DefaultURL, "The nats server URLs (separated by comma)")
	flag.Usage = usage
	flag.Parse()
	var hosts = strings.Split(*urls, ",")
	log.Printf("hosts : '%v'\n", hosts)

	//init service and broker
	config := &moleculer.ServiceBrokerConfig{
		NatsHost: hosts,
		NodeID:   "ws-connector",
		// LogLevel: moleculer.DebugLevel,
		LogLevel: moleculer.ErrorLevel,
		Services: make(map[string]moleculer.Service),
	}
	config.Services["ws-connector"] = createService()
	broker, err := moleculer.NewServiceBroker(config)
	if err != nil {
		log.Fatalf("NewServiceBroker err: %v\n", err)
	}
	pBroker = broker
	broker.Start()

	closer.Hold()
}

func cleanupFunc() {
	log.Infof("Hang on! %s is closing ...", AppName)
	log.Warn("=================== exit start =================== ")
	if pBroker != nil {
		pBroker.Stop()
	}
	time.Sleep(time.Second * 1)
	log.Warn("=================== exit end   =================== ")
	log.Infof("%s is closed", AppName)
}
