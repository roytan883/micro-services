package main

import (

	// _ "net/http/pprof" //https://localhost:12020/debug/pprof

	moleculer "github.com/roytan883/moleculer-go"
	"github.com/roytan883/moleculer-go/protocol"
)

var gMoleculerService *moleculer.Service

func createMoleculerService() moleculer.Service {
	gMoleculerService = &moleculer.Service{
		ServiceName: ServiceName,
		Actions:     make(map[string]moleculer.RequestHandler),
		Events:      make(map[string]moleculer.EventHandler),
	}

	//init actions handlers
	gMoleculerService.Actions[cWsOnlineActionUserInfo] = actionUserInfo

	//init listen events handlers
	gMoleculerService.Events[cWsConnectorOutOnline] = eventWsConnectorOutOnline
	gMoleculerService.Events[cWsConnectorOutOffline] = eventWsConnectorOutOffline
	gMoleculerService.Events[cWsConnectorOutSyncUsersInfo] = eventWsConnectorOutSyncUsersInfo

	return *gMoleculerService
}

func actionUserInfo(req *protocol.MsRequest) (interface{}, error) {
	log.Info("run actionUserInfo")
	return nil, nil
}

func eventWsConnectorOutOffline(req *protocol.MsEvent) {
	log.Info("run eventWsConnectorOutOffline")
}

func eventWsConnectorOutOnline(req *protocol.MsEvent) {
	log.Info("run eventWsConnectorOutOnline")
}

func eventWsConnectorOutSyncUsersInfo(req *protocol.MsEvent) {
	log.Info("run eventWsConnectorOutSyncUsersInfo")
}
