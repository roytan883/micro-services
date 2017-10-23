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
var gWriteLogToFile int
var gTestCount int
var gTestUserName string
var gTestUserNameRange int
var gAbandonMinutes int
var gSyncDelaySeconds int
var gNodeID = AppName

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
	cWsCacheActionSave  = "ws-cache.save"  //in: cacheStruct || out: null, err
)

type cacheStruct struct {
	UserID    string      `json:"userID"`
	Cid       string      `json:"cid"`
	Timestamp string      `json:"timestamp"`
	Mid       string      `json:"mid"`
	Msg       interface{} `json:"msg"`
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
	IsOnline       bool   `json:"isOnline"`
}

type userIDStruct struct {
	UserID string `json:"userID"`
}

type idsStruct struct {
	IDs interface{} `json:"ids"`
}

type onlineStatusStruct struct {
	UserID          string        `json:"userID"`
	IsShortOnline   bool          `json:"isShortOnline"`
	IsRealOnline    bool          `json:"isRealOnline"`
	RealOnlineInfos []*ClientInfo `json:"realOnlineInfos"`
}

type onlineStatusBulkStruct struct {
	OnlineStatusBulk []*onlineStatusStruct `json:"onlineStatusBulk"`
}

type abandonStruct struct {
	UserID          string
	Cid             string
	LastOfflineTime time.Time
}
