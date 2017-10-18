package main

import (
	"flag"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	rotatelogs "github.com/lestrrat/go-file-rotatelogs"
	nats "github.com/nats-io/go-nats"
	"github.com/rifflock/lfshook"
	moleculer "github.com/roytan883/moleculer-go"
	logrus "github.com/sirupsen/logrus"
	"github.com/xlab/closer"
)

const (
	//AppName ...
	AppName = "ws-sender"
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
	os.Mkdir("logs", os.ModePerm)
	if gIsDebug > 0 {
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

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func usage() {
	log.Fatalf("Usage: ws-online [-s The nats server URLs (nats://192.168.1.69:12008)] [-i nodeID (0)] [-d debug (0)] [-w WaitAckSeconds (10)]\n")
}

var gCloseChan chan int

//./ws-sender -s nats://192.168.1.69:12008
func main() {

	gCloseChan = make(chan int, 1)

	closer.Bind(cleanupFunc)

	_gUrls := flag.String("s", nats.DefaultURL, "The nats server URLs (separated by comma, default localhost:4222)")
	_gID := flag.Int("i", 0, "ID of the service on this machine")
	_gWaitAckSeconds := flag.Int("w", 10, "wait 10s ack")
	_gIsDebug := flag.Int("d", 0, "is debug")
	// _gTestCount := flag.Int("c", 1, "test send message RPS")
	// _gTestUserName := flag.String("u", "gotest-user-", "TestUserName prefix")
	// _gTestUserNameRange := flag.Int("ur", 9999, "TestUserName range")
	flag.Usage = usage
	flag.Parse()

	gUrls = *_gUrls
	gID = *_gID
	gIsDebug = *_gIsDebug
	gWaitAckSeconds = *_gWaitAckSeconds
	// gTestCount = *_gTestCount
	// gTestUserName = *_gTestUserName
	// gTestUserNameRange = *_gTestUserNameRange

	setDebug()

	log.Infof("Start %s ...\n", AppName)

	gNatsHosts = strings.Split(gUrls, ",")

	gNodeID += "-" + strconv.Itoa(gID)
	log.Warnf("gNodeID : %v\n", gNodeID)
	log.Warnf("gIsDebug : %v\n", gIsDebug)
	log.Warnf("gWaitAckSeconds : %v\n", gWaitAckSeconds)

	//init service and broker
	config := &moleculer.ServiceBrokerConfig{
		NatsHost:              gNatsHosts,
		NodeID:                gNodeID,
		DefaultRequestTimeout: time.Second * 3,
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

	log.Warn("================= Server Started ================= ")

	closer.Hold()
}

func cleanupFunc() {
	log.Infof("Hang on! %s is closing ...", AppName)
	log.Warn("=================== exit start =================== ")
	time.Sleep(time.Second * 1)
	log.Warn("=================== exit end   =================== ")
	log.Infof("%s is closed", AppName)
}
