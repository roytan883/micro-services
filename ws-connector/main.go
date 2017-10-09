package main

import (
	"flag"
	"strconv"
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
	//ServiceName ...
	ServiceName = AppName
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
	log.WithFields(logrus.Fields{"package": AppName})
}

//TLS:
//openssl genrsa -out key_go.pem 2048
//openssl req -new -x509 -key key_go.pem -out cert_go.pem -days 36500

// NOTE: Use tls scheme for TLS, e.g. nats-req -s tls://demo.nats.io:4443 foo hello
// go run .\main.go .\ws-connector.go .\hub.go .\client.go .\pool.go -s nats://192.168.1.69:12008
// ws-connector -s nats://192.168.1.69:12008
// ws-connector -s nats://127.0.0.1:4222
func usage() {
	log.Fatalf("Usage: ws-connector [-s server (%s)] [-p port (12020)] [-i nodeID (0)] \n", nats.DefaultURL)
}

var pBroker *moleculer.ServiceBroker

var gUrls string
var gNatsHosts []string
var gPort int
var gID int
var gNodeID = AppName

//debug: go run .\main.go .\ws-connector.go
func main() {
	closer.Bind(cleanupFunc)
	log.Infof("Start %s ...\n", AppName)

	//get NATS server host
	_gUrls := flag.String("s", nats.DefaultURL, "The nats server URLs (separated by comma, default localhost:4222)")
	_gPort := flag.Int("p", 12020, "listen websocket port")
	_gID := flag.Int("i", 0, "ID of the service on this machine")
	flag.Usage = usage
	flag.Parse()

	gUrls = *_gUrls
	gPort = *_gPort
	gID = *_gID

	gNatsHosts = strings.Split(gUrls, ",")
	log.Printf("gUrls : %v\n", gUrls)
	log.Printf("gNatsHosts : %v\n", gNatsHosts)
	log.Printf("gPort : %v\n", gPort)
	log.Printf("gID : %v\n", gID)

	gNodeID += "-" + strconv.Itoa(gID)
	log.Printf("gNodeID : %v\n", gNodeID)

	//init service and broker
	config := &moleculer.ServiceBrokerConfig{
		NatsHost: gNatsHosts,
		NodeID:   gNodeID,
		// LogLevel: moleculer.DebugLevel,
		LogLevel: moleculer.ErrorLevel,
		Services: make(map[string]moleculer.Service),
	}
	moleculerService := createMoleculerService()
	config.Services[moleculerService.ServiceName] = moleculerService
	broker, err := moleculer.NewServiceBroker(config)
	if err != nil {
		log.Fatalf("NewServiceBroker err: %v\n", err)
	}
	pBroker = broker
	err = broker.Start()
	if err != nil {
		log.Fatalf("exit process, broker.Start err: %v\n", err)
		return
	}

	startWsService()

	closer.Hold()
}

func cleanupFunc() {
	log.Infof("Hang on! %s is closing ...", AppName)
	log.Warn("=================== exit start =================== ")
	if pBroker != nil {
		pBroker.Stop()
	}
	stopWsService()
	time.Sleep(time.Second * 1)
	log.Warn("=================== exit end   =================== ")
	log.Infof("%s is closed", AppName)
}
