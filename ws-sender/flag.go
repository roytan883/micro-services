package main

import (
	"flag"
	"strconv"
	"strings"

	nats "github.com/nats-io/go-nats"
)

var gUrls string
var gNatsHosts []string
var gPort int
var gID int
var gFastExit int
var gIsDebug int
var gWriteLogToFile int
var gNodeID = AppName
var gWaitAckSeconds int

func initFlag() {
	_gUrls := flag.String("s", nats.DefaultURL, "The nats server URLs (separated by comma, default localhost:4222)")
	_gID := flag.Int("i", 0, "ID of the service on this machine")
	_gWaitAckSeconds := flag.Int("w", 10, "wait 10s ack")

	_gFastExit := flag.Int("fe", 0, "fast exit")
	_gIsDebug := flag.Int("d", 0, "is debug")
	_gWriteLogToFile := flag.Int("wf", 0, "write log to file")

	flag.Usage = usage
	flag.Parse()

	gUrls = *_gUrls
	gID = *_gID

	gIsDebug = *_gIsDebug
	gFastExit = *_gFastExit
	gWriteLogToFile = *_gWriteLogToFile

	gWaitAckSeconds = *_gWaitAckSeconds

	gNatsHosts = strings.Split(gUrls, ",")

	gNodeID += "-" + strconv.Itoa(gID)

}

func printFlag() {
	log.Warnf("gIsDebug : %v\n", gIsDebug)
	log.Warnf("gWriteLogToFile : %v\n", gWriteLogToFile)
	log.Warnf("gNodeID : %v\n", gNodeID)
	log.Warnf("gUrls : %v\n", gUrls)
	log.Warnf("gNatsHosts : %v\n", gNatsHosts)
	log.Warnf("gWaitAckSeconds : %v\n", gWaitAckSeconds)
}
