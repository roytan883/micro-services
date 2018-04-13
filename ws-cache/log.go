package main

import (
	"os"
	"time"

	rotatelogs "github.com/lestrrat/go-file-rotatelogs"
	"github.com/rifflock/lfshook"
	logrus "github.com/sirupsen/logrus"
)

var log *logrus.Logger

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
			log.Hooks.Add(lfshook.NewHook(
				lfshook.WriterMap{
					logrus.DebugLevel: debugLogWriter,
					logrus.InfoLevel:  debugLogWriter,
					logrus.WarnLevel:  warnLogWriter,
					logrus.ErrorLevel: warnLogWriter,
					logrus.FatalLevel: warnLogWriter,
					logrus.PanicLevel: warnLogWriter,
				},
				&logrus.JSONFormatter{},
			))
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
			log.Hooks.Add(lfshook.NewHook(
				lfshook.WriterMap{
					logrus.WarnLevel:  warnLogWriter,
					logrus.ErrorLevel: warnLogWriter,
					logrus.FatalLevel: warnLogWriter,
					logrus.PanicLevel: warnLogWriter,
				},
				&logrus.JSONFormatter{},
			))
		}
	}
}
