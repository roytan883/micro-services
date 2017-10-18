package main

import (
	"os"
	"time"

	"github.com/xlab/closer"
)

func usage() {
	log.Fatalf("Usage: ws-online [-s The nats server URLs (nats://192.168.1.69:12008)] [-i nodeID (0)] [-d debug (0)] [-w WaitAckSeconds (10)] [-fe FastExit (0)] [-wf WriteLogToFile (0)]\n")
}

//./ws-sender -s nats://192.168.1.69:12008
func main() {
	closer.Bind(cleanupFunc)

	initFlag()
	setDebug()
	printFlag()

	log.Warnf("Start Server: %s ...\n", AppName)

	setupMoleculerService()

	log.Warn("=================== Server Started ================= ")

	closer.Hold()
}

func cleanupFunc() {
	log.Warnf("Stop Server: %s ...\n", AppName)
	if gFastExit > 0 {
		log.Warn("================= fast exit  ================== ")
		os.Exit(0)
	}
	log.Warnf("Hang on! Server[%s] is closing ...", AppName)
	log.Warn("=================== exit start =================== ")
	time.Sleep(time.Second * 1)
	log.Warn("=================== exit end   =================== ")
	log.Warnf("Server[%s] is closed", AppName)
}
