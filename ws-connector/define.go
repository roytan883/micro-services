package main

import (
	"time"

	moleculer "github.com/roytan883/moleculer-go"
	logrus "github.com/sirupsen/logrus"
)

var log *logrus.Logger

var pBroker *moleculer.ServiceBroker

var gUrls string
var gNatsHosts []string
var gPort int
var gID int
var gIsDebug int
var gNodeID = AppName

var gHub *Hub

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 200 * time.Second * 1

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	// maxMessageSize = 1024 * 4
)

type gCmdType uint32

const (
	cgListenPush             = AppName + ".in.push"
	cgListenKickClient       = AppName + ".in.kickClient"
	cgListenKickUser         = AppName + ".in.kickUser"
	cgListeSyncUsersInfo     = AppName + ".in.syncUsersInfo"
	cgBroadcastUserOnline    = AppName + ".out.userOnline"
	cgBroadcastUserOffline   = AppName + ".out.userOffline"
	cgBroadcastAck           = AppName + ".out.ack"
	cgBroadcastSyncUsersInfo = AppName + ".out.syncUsersInfo"
)

type ackStruct struct {
	Aid    string `json:"aid"`
	Cid    string `json:"cid"`
	UserID string `json:"userID"`
}

type pushMsgStruct struct {
	IDs interface{} `json:"ids"`
	Mid string      `json:"mid"`
	Msg interface{} `json:"msg"`
}

type kickClientStruct struct {
	Cid string `json:"cid"`
}

type kickUserStruct struct {
	UserID string `json:"userID"`
}

type getUserOnlineInfoStruct struct {
	UserID string `json:"userID"`
}
