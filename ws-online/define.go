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
	cWsConnectorActionPush       = "push"
	cWsConnectorActionCount      = "count"
	cWsConnectorActionMetrics    = "metrics"
	cWsConnectorActionUserInfo   = "userInfo"
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
	cWsOnlineActionOnlineStatus = "onlineStatus"
	cWsOnlineOutOnline          = "ws-online.out.online"
	cWsOnlineOutOffline         = "ws-online.out.offline"
)

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

type userIDStruct struct {
	UserID string `json:"userID"`
}

type onlineStatusStruct struct {
	IsShortOnline   bool          `json:"isShortOnline"`
	IsRealOnline    bool          `json:"isRealOnline"`
	RealOnlineInfos []*ClientInfo `json:"realOnlineInfos"`
}
