package main

import (
	"flag"
	"os"
	"strconv"
	"strings"
	"time"

	rotatelogs "github.com/lestrrat/go-file-rotatelogs"
	nats "github.com/nats-io/go-nats"
	"github.com/rifflock/lfshook"
	moleculer "github.com/roytan883/moleculer-go"
	logrus "github.com/sirupsen/logrus"
	"github.com/xlab/closer"
)

const (
	//AppName ...
	AppName = "ws-connector"
	//ServiceName ...
	ServiceName = AppName
	//IsDebug change logLevel here for dev and pro
	IsDebug = true
)

func init() {
	initLog()
}

func initLog() {
	log = logrus.New()
	log.Formatter = &logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "01-02 15:04:05.000000",
	}
	log.WithFields(logrus.Fields{"package": AppName})

	os.Mkdir("logs", os.ModePerm)
	if IsDebug {
		log.SetLevel(logrus.DebugLevel)
		debugLogPath := "logs/debug.log"
		warnLogPath := "logs/warn.log"
		debugLogWriter, err := rotatelogs.New(
			debugLogPath+".%Y%m%d%H%M%S",
			rotatelogs.WithLinkName(debugLogPath),
			rotatelogs.WithMaxAge(time.Hour*24*7),
			rotatelogs.WithRotationTime(time.Hour*24),
		)
		if err != nil {
			log.Printf("failed to create rotatelogs debugLogWriter : %s", err)
			return
		}
		warnLogWriter, err := rotatelogs.New(
			warnLogPath+".%Y%m%d%H%M%S",
			rotatelogs.WithLinkName(warnLogPath),
			rotatelogs.WithMaxAge(time.Hour*24*7),
			rotatelogs.WithRotationTime(time.Hour*24),
		)
		if err != nil {
			log.Printf("failed to create rotatelogs warnLogWriter : %s", err)
			return
		}
		log.Hooks.Add(lfshook.NewHook(lfshook.WriterMap{
			logrus.DebugLevel: debugLogWriter,
			logrus.InfoLevel:  debugLogWriter,
			logrus.WarnLevel:  warnLogWriter,
			logrus.ErrorLevel: warnLogWriter,
			logrus.FatalLevel: warnLogWriter,
			logrus.PanicLevel: warnLogWriter,
		}))
	} else {
		log.SetLevel(logrus.WarnLevel)
		warnLogPath := "logs/warn.log"
		warnLogWriter, err := rotatelogs.New(
			warnLogPath+".%Y%m%d%H%M%S",
			rotatelogs.WithLinkName(warnLogPath),
			rotatelogs.WithMaxAge(time.Hour*24*7),
			rotatelogs.WithRotationTime(time.Hour*24),
		)
		if err != nil {
			log.Printf("failed to create rotatelogs warnLogWriter : %s", err)
			return
		}
		log.Hooks.Add(lfshook.NewHook(lfshook.WriterMap{
			logrus.WarnLevel:  warnLogWriter,
			logrus.ErrorLevel: warnLogWriter,
			logrus.FatalLevel: warnLogWriter,
			logrus.PanicLevel: warnLogWriter,
		}))
	}
}

//TLS:
//openssl genrsa -out key_go.pem 2048
//openssl req -new -x509 -key key_go.pem -out cert_go.pem -days 36500

// NOTE: Use tls scheme for TLS, e.g. nats-req -s tls://demo.nats.io:4443 foo hello
// go run .\define.go .\main.go .\ws-connector.go .\hub.go .\client.go .\pool.go -s nats://192.168.1.69:12008
// ws-connector -s nats://192.168.1.69:12008
// ws-connector -s nats://127.0.0.1:4222
func usage() {
	log.Fatalf("Usage: ws-connector [-s server (%s)] [-p port (12020)] [-i nodeID (0)] \n", nats.DefaultURL)
}

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
	log.Warnf("gUrls : %v\n", gUrls)
	log.Warnf("gNatsHosts : %v\n", gNatsHosts)
	log.Warnf("gPort : %v\n", gPort)
	log.Warnf("gID : %v\n", gID)

	gNodeID += "-" + strconv.Itoa(gID)
	log.Warnf("gNodeID : %v\n", gNodeID)

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

	log.Warn("================= Server Started ================= ")
	log.Error("================= Error test ================= ")
	demoWsString := "you can connect to the server by weboscket >>> wss://x.x.x.x:" + strconv.Itoa(gPort) + "/ws?userID=uaaa&platform=web&version=0.1.0&timestamp=1507870585757&token=73ce0b2d7b47b4af75f38dcabf8e3ce9894e6e6e"
	log.Warn(demoWsString)

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
