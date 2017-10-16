package main

import (
	"flag"
	"math/rand"
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
	AppName = "ws-connector-test-sender"
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

//TLS:
//openssl genrsa -out key_go.pem 2048
//openssl req -new -x509 -key key_go.pem -out cert_go.pem -days 36500

// NOTE: Use tls scheme for TLS, e.g. nats-req -s tls://demo.nats.io:4443 foo hello
// go run .\define.go .\main.go .\ws-connector.go .\hub.go .\client.go .\pool.go -s nats://192.168.1.69:12008
// ws-connector -s nats://192.168.1.69:12008
// ws-connector -s nats://127.0.0.1:4222
func usage() {
	log.Fatalf("Usage: ws-connector-test [-s The nats server URLs (nats://192.168.1.69:12008)] [-d debug (0)] [-c TestCount (1)] [-u TestUserName (gotest-user-)] [-ur TestUserName range (9999)]\n")
}

var gCloseChan chan int

//.\ws-connector-test-sender.exe -s nats://192.168.1.69:12008 -c 500
func main() {

	gCloseChan = make(chan int, 1)

	closer.Bind(cleanupFunc)

	_gUrls := flag.String("s", nats.DefaultURL, "The nats server URLs (separated by comma, default localhost:4222)")
	_gIsDebug := flag.Int("d", 0, "is debug")
	_gTestCount := flag.Int("c", 1, "test send message RPS")
	_gTestUserName := flag.String("u", "gotest-user-", "TestUserName prefix")
	_gTestUserNameRange := flag.Int("ur", 9999, "TestUserName range")
	flag.Usage = usage
	flag.Parse()

	gUrls = *_gUrls
	gIsDebug = *_gIsDebug
	gTestCount = *_gTestCount
	gTestUserName = *_gTestUserName
	gTestUserNameRange = *_gTestUserNameRange

	setDebug()

	log.Infof("Start %s ...\n", AppName)

	gNatsHosts = strings.Split(gUrls, ",")

	//init service and broker
	config := &moleculer.ServiceBrokerConfig{
		NatsHost:              gNatsHosts,
		NodeID:                AppName,
		DefaultRequestTimeout: time.Second * 2,
		// LogLevel: moleculer.DebugLevel,
		LogLevel: moleculer.ErrorLevel,
		Services: make(map[string]moleculer.Service),
	}
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

	go func() {
		ticker := time.NewTicker(time.Second * 1)
		for {
			select {
			case <-ticker.C:
				// log.Info("ALL Goroutine: ", runtime.NumGoroutine())
				for index := 0; index < gTestCount; index++ {
					id := gTestUserName + strconv.Itoa(rand.Intn(gTestUserNameRange))
					mid := id + "-" + strconv.Itoa(int(time.Now().UnixNano())) + "-" + strconv.Itoa(rand.Intn(gTestUserNameRange))
					testData := map[string]interface{}{
						"ids": id,
						"data": map[string]interface{}{
							"mid": mid,
							"msg": map[string]interface{}{
								"aaa": "hello world",
								"bbb": 123,
								"ccc": true,
							},
						},
					}
					pBroker.Broadcast("ws-connector.in.push", testData)
				}
			case <-gCloseChan:
				return
			}
		}
	}()

	log.Warn("================= Server Started ================= ")

	closer.Hold()
}

func cleanupFunc() {
	log.Infof("Hang on! %s is closing ...", AppName)
	log.Warn("=================== exit start =================== ")
	gCloseChan <- 1
	time.Sleep(time.Second * 1)
	log.Warn("=================== exit end   =================== ")
	log.Infof("%s is closed", AppName)
}
