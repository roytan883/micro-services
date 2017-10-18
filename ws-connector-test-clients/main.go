package main

import (
	"flag"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	rotatelogs "github.com/lestrrat/go-file-rotatelogs"
	"github.com/rifflock/lfshook"
	logrus "github.com/sirupsen/logrus"
	"github.com/xlab/closer"
)

const (
	//AppName ...
	AppName = "ws-connector-test-clients"
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
	log.Fatalf("Usage: ws-connector-test [-s server (127.0.0.1:12020)] [-d debug (0)] [-c TestCount (1)] [-u TestUserName (gotest-user-)] \n")
}

// .\ws-connector-test-clients.exe -s 192.168.1.69:12020 -c 10000
func main() {
	closer.Bind(cleanupFunc)

	_gUrls := flag.String("s", "127.0.0.1:12020", "The websocket server host address")
	_gIsDebug := flag.Int("d", 0, "is debug")
	_gTestCount := flag.Int("c", 1, "test websocket count")
	_gTestUserName := flag.String("u", "gotest-user-", "TestUserName prefix")
	flag.Usage = usage
	flag.Parse()

	gUrls = *_gUrls
	gIsDebug = *_gIsDebug
	gTestCount = *_gTestCount

	gTestUserName = *_gTestUserName

	setDebug()

	log.Infof("Start %s ...\n", AppName)

	urlString := "wss://" + gUrls + "/ws?"
	userID := gTestUserName
	platform := "gotest"
	version := "0.0.1"
	timestamp := "111"
	token := "gotesttoken"

	for index := 0; index < gTestCount; index++ {
		id := userID + strconv.Itoa(index)
		theURLString := urlString + "userID=" + id + "&platform=" + platform + "&version=" + version + "&timestamp=" + timestamp + "&token=" + token
		time.Sleep(time.Microsecond * 500)
		go NewWsClient(id, theURLString)
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
