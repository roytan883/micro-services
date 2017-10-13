package main

import (
	"flag"
	"strings"
	"sync/atomic"
	"time"

	nats "github.com/nats-io/go-nats"
	moleculer "github.com/roytan883/moleculer-go"
	logrus "github.com/sirupsen/logrus"

	"github.com/xlab/closer"
)

const (
	//AppName ...
	AppName = "micro-services-debug"
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
	log.Fatalf("Usage: dev [-s server (%s)] \n", nats.DefaultURL)
}

var pBroker *moleculer.ServiceBroker

//debug: go run .\main.go
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
		NodeID:   AppName,
		// LogLevel: moleculer.DebugLevel,
		LogLevel: moleculer.ErrorLevel,
		Services: make(map[string]moleculer.Service),
	}
	broker, err := moleculer.NewServiceBroker(config)
	if err != nil {
		log.Fatalf("NewServiceBroker err: %v\n", err)
	}
	pBroker = broker
	broker.Start()

	test2()

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

var targetService = "ws-connector"

func test() {
	//test call and emit
	go time.AfterFunc(time.Second*1, func() {
		log.Info("broker.Call push start")
		res, err := pBroker.Call(targetService+".push", map[string]interface{}{
			"arg1": "aaa",
			"arg2": 123,
		}, nil)
		log.Info("broker.Call push end, res: ", res)
		log.Info("broker.Call push end, err: ", err)

		// log.Info("broker.Emit user.create start")
		// err = pBroker.Emit("user.create", map[string]interface{}{
		// 	"user":   "userA",
		// 	"status": "create",
		// })
		// log.Info("broker.Emit user.create end, err: ", err)

	})
}

func test2() {
	var count int32
	//1.8G for 200K go goroutines
	var testCount int32 = 200000 * 1
	startTime := time.Now()
	log.Info("test2 startTime = ", startTime)
	var index int32
	for ; index <= testCount; index++ {
		if atomic.LoadInt32(&count) <= testCount {
			go func() {
				if atomic.LoadInt32(&count) >= testCount {
					endTime := time.Now()
					log.Info("test2 endTime = ", endTime)
					log.Info("test2 use time = ", endTime.Sub(startTime))
					return
				}

				atomic.AddInt32(&count, 1)
				time.Sleep(time.Second * 100)

				// log.Info("test2 count = ", count)
			}()
		}
	}
}
