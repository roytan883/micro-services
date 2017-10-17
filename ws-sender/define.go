package main

import (
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
var gNodeID = AppName

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
	IDs  interface{}        `json:"ids"`
	Data *pushMsgDataStruct `json:"data"`
}

type pushMsgDataStruct struct {
	Mid string      `json:"mid"`
	Msg interface{} `json:"msg"`
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
