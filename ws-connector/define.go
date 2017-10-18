package main

import (
	"time"

	jsoniter "github.com/json-iterator/go"
	moleculer "github.com/roytan883/moleculer-go"
	logrus "github.com/sirupsen/logrus"
)

var log *logrus.Logger

var pBroker *moleculer.ServiceBroker

var gUrls string
var gNatsHosts []string
var gPort int
var gID int
var gRPS int
var gMaxClients int
var gIsDebug int
var gFastExit int
var gWriteLogToFile int
var gNodeID = AppName

var gHub *Hub

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 200 * time.Second * 1

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	maxConcurrentAccept = 500

	// Maximum message size allowed from peer.
	// maxMessageSize = 1024 * 4
)

type gCmdType uint32

const (
	cWsConnectorActionPush       = "ws-connector.push"
	cWsConnectorActionCount      = "ws-connector.count"
	cWsConnectorActionMetrics    = "ws-connector.metrics"
	cWsConnectorActionUserInfo   = "ws-connector.userInfo"
	cWsConnectorInPush           = "ws-connector.in.push"
	cWsConnectorInKickClient     = "ws-connector.in.kickClient"
	cWsConnectorInKickUser       = "ws-connector.in.kickUser"
	cWsConnectorOutOnline        = "ws-connector.out.online"
	cWsConnectorOutOffline       = "ws-connector.out.offline"
	cWsConnectorInSyncUsersInfo  = "ws-connector.in.syncUsersInfo"
	cWsConnectorOutSyncUsersInfo = "ws-connector.out.syncUsersInfo"
	cWsConnectorOutAck           = "ws-connector.out.ack"
	cWsConnectorInSyncMetrics    = "ws-connector.in.syncMetrics"
	cWsConnectorOutSyncMetrics   = "ws-connector.out.syncMetrics"

	cWsTokenActionVerify = "ws-token.verify"

	//in: userIDStruct `json:"userID"` || out:onlineStatusStruct
	cWsOnlineActionOnlineStatus = "ws-online.onlineStatus"
	//in: idsStruct `json:"ids"` || out:onlineStatusBulkStruct
	cWsOnlineActionOnlineStatusBulk = "ws-online.onlineStatusBulk"
	cWsOnlineOutOnline              = "ws-online.out.online"
	cWsOnlineOutOffline             = "ws-online.out.offline"

	cWsSenderActionSend = "ws-sender.send"
)

type ackStruct struct {
	Aid    string `json:"aid"`
	Cid    string `json:"cid"`
	UserID string `json:"userID"`
}

type pushMsgStruct struct {
	IDs  interface{} `json:"ids"`
	Data interface{} `json:"data"`
}

type pushMsgDataStruct struct {
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

type verifyTokenStruct struct {
	URL       string `json:"url"`
	UserID    string `json:"userID"`
	Platform  string `json:"platform"`
	Version   string `json:"version"`
	Timestamp string `json:"timestamp"`
	Token     string `json:"token"`
}

func (v *verifyTokenStruct) String() string {
	data, err := jsoniter.Marshal(v)
	if err != nil {
		return ""
	}
	return string(data)
}

type verifyTokenResultStruct struct {
	Invalid bool `json:"invalid"` //GO default bool is false, so use invalid == true to detect invalid token
}

//ClientInfo ...
type ClientInfo struct {
	NodeID         string `json:"nodeID"`
	Cid            string `json:"cid"`
	UserID         string `json:"userID"`
	Platform       string `json:"platform"`
	Version        string `json:"version"`
	Timestamp      string `json:"timestamp"`
	Token          string `json:"token"`
	ConnectTime    string `json:"connectTime"`
	DisconnectTime string `json:"disconnectTime"`
}

type metricsStruct struct {
	NodeID           string `json:"nodeID"`
	Port             int    `json:"port"`
	OnlineUsers      uint64 `json:"onlineUsers"`
	TotalTrySend     uint64 `json:"totalTrySend"`
	TotalSend        uint64 `json:"totalSend"`
	TotalTryAck      uint64 `json:"totalTryAck"`
	TotalAck         uint64 `json:"totalAck"`
	CurrentAccepting int64  `json:"currentAccepting"`
}

var gTotalTrySend uint64
var gTotalSend uint64
var gTotalTryAck uint64
var gTotalAck uint64
var gCurrentAccepting int64
var gCurrentClients int64
