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
	pongWait = 240 * time.Second * 1

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = 210 * time.Second * 1

	maxConcurrentAccept = 500

	// Maximum message size allowed from peer.
	// maxMessageSize = 1024 * 4
)

type gCmdType uint32

const (
	cWsConnectorActionPush       = "ws-connector.push"              //in: pushMsgStruct || out: null, err
	cWsConnectorActionCount      = "ws-connector.count"             //in: null || out: count, err
	cWsConnectorActionMetrics    = "ws-connector.metrics"           //in: null || out: metricsStruct, err
	cWsConnectorActionUserInfo   = "ws-connector.userInfo"          //in: userIDStruct || out: []ClientInfo, err
	cWsConnectorInPush           = "ws-connector.in.push"           //pushMsgStruct
	cWsConnectorInKickClient     = "ws-connector.in.kickClient"     //cidStruct
	cWsConnectorInKickUser       = "ws-connector.in.kickUser"       //userIDStruct
	cWsConnectorOutOnline        = "ws-connector.out.online"        //ClientInfo
	cWsConnectorOutOffline       = "ws-connector.out.offline"       //ClientInfo
	cWsConnectorInSyncUsersInfo  = "ws-connector.in.syncUsersInfo"  //null
	cWsConnectorOutSyncUsersInfo = "ws-connector.out.syncUsersInfo" //ClientInfo
	cWsConnectorOutAck           = "ws-connector.out.ack"           //ackStruct
	cWsConnectorInSyncMetrics    = "ws-connector.in.syncMetrics"    //null
	cWsConnectorOutSyncMetrics   = "ws-connector.out.syncMetrics"   //metricsStruct

	cWsTokenActionVerify = "ws-token.verify"

	cWsOnlineActionOnlineStatus     = "ws-online.onlineStatus"     //in: userIDStruct `json:"userID"` || out:onlineStatusStruct, err
	cWsOnlineActionOnlineStatusBulk = "ws-online.onlineStatusBulk" //in: idsStruct `json:"ids"` || out:onlineStatusBulkStruct, err
	cWsOnlineOutOnline              = "ws-online.out.online"       //ClientInfo
	cWsOnlineOutOffline             = "ws-online.out.offline"      //ClientInfo

	cWsSenderActionSend = "ws-sender.send" //in: pushMsgStruct || out: null, err
	cWsCacheActionSave  = "ws-cache.save"  //in: cacheMsgStruct || out: null, err
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

type cidStruct struct {
	Cid string `json:"cid"`
}

type userIDStruct struct {
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
