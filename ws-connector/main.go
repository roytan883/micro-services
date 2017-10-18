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

}

func setDebug() {
	if gIsDebug > 0 {
		log.SetLevel(logrus.DebugLevel)
		if gWriteLogToFile > 0 {
			os.Mkdir("logs", os.ModePerm)
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
		}
	} else {
		log.SetLevel(logrus.WarnLevel)
		if gWriteLogToFile > 0 {
			os.Mkdir("logs", os.ModePerm)
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
}

//TLS:
//openssl genrsa -out key_go.pem 2048
//openssl req -new -x509 -key key_go.pem -out cert_go.pem -days 36500

// NOTE: Use tls scheme for TLS, e.g. nats-req -s tls://demo.nats.io:4443 foo hello
// go run .\define.go .\main.go .\ws-connector.go .\hub.go .\client.go .\pool.go -s nats://192.168.1.69:12008
// ws-connector -s nats://192.168.1.69:12008
// ws-connector -s nats://127.0.0.1:4222
func usage() {
	log.Fatalf("Usage: ws-connector [-s server (%s)] [-p port (12020)] [-i nodeID (0)] [-d debug (0)] [-r RPS (2500)] [-m MaxClients (500000 (20G) //400MB~10K user)] [-fe FastExit (0)] [-wf WriteLogToFile (0)]\n", nats.DefaultURL)
}

/*
pro:
./ws-connector -s nats://127.0.0.1:12008 -p 12020 -i 0 -d 0 -fe 0 -wf 1
dev:
./ws-connector -s nats://127.0.0.1:12008 -p 12020 -i 0 -d 1 -fe 1 -wf 0
./ws-connector -s nats://127.0.0.1:12008 -p 12021 -i 1 -d 1 -fe 1 -wf 0
*/
func main() {
	closer.Bind(cleanupFunc)

	//get NATS server host
	_gUrls := flag.String("s", nats.DefaultURL, "The nats server URLs (separated by comma, default localhost:4222)")
	_gPort := flag.Int("p", 12020, "listen websocket port")
	_gID := flag.Int("i", 0, "ID of the service on this machine")
	_gRPS := flag.Int("r", 2500, "max request per second")
	_gMaxClients := flag.Int("m", 500000, "max clients")
	_gIsDebug := flag.Int("d", 0, "is debug")
	_gFastExit := flag.Int("fe", 0, "fast exit")
	_gWriteLogToFile := flag.Int("wf", 0, "write log to file")
	flag.Usage = usage
	flag.Parse()

	gUrls = *_gUrls
	gPort = *_gPort
	gID = *_gID
	gRPS = *_gRPS
	gMaxClients = *_gMaxClients
	gIsDebug = *_gIsDebug
	gFastExit = *_gFastExit
	gWriteLogToFile = *_gWriteLogToFile

	setDebug()

	log.Infof("Start %s ...\n", AppName)

	gNatsHosts = strings.Split(gUrls, ",")
	log.Warnf("gUrls : %v\n", gUrls)
	log.Warnf("gNatsHosts : %v\n", gNatsHosts)
	log.Warnf("gPort : %v\n", gPort)
	log.Warnf("gID : %v\n", gID)
	log.Warnf("gIsDebug : %v\n", gIsDebug)
	log.Warnf("gMaxClients : %v\n", gMaxClients)

	gNodeID += "-" + strconv.Itoa(gID)
	log.Warnf("gNodeID : %v\n", gNodeID)

	//init service and broker
	config := &moleculer.ServiceBrokerConfig{
		NatsHost:              gNatsHosts,
		NodeID:                gNodeID,
		DefaultRequestTimeout: time.Second * 2,
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
	demoWsString := "you can connect to the server by weboscket >>> wss://x.x.x.x:" + strconv.Itoa(gPort) + "/ws?userID=uaaa&platform=web&version=0.1.0&timestamp=1507870585757&token=73ce0b2d7b47b4af75f38dcabf8e3ce9894e6e6e"
	log.Warn(demoWsString)

	closer.Hold()
}

func cleanupFunc() {
	if gFastExit > 0 {
		log.Warn("=================== fast exit =================== ")
		os.Exit(0)
	}
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
