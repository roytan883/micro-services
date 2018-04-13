package main

import (

	// _ "net/http/pprof" //https://localhost:12220/debug/pprof

	jsoniter "github.com/json-iterator/go"
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
	// gMoleculerService.Actions["send"] = actionSend

	//init listen events handlers
	gMoleculerService.Events[cWsConnectorOutOnline] = eventAnalyze
	gMoleculerService.Events[cWsConnectorOutOffline] = eventAnalyze
	gMoleculerService.Events[cWsConnectorOutSyncUsersInfo] = eventAnalyze
	gMoleculerService.Events[cWsConnectorOutAck] = eventAnalyze
	gMoleculerService.Events[cWsConnectorOutSyncMetrics] = eventAnalyze
	gMoleculerService.Events[cWsOnlineOutOnline] = eventAnalyze
	gMoleculerService.Events[cWsOnlineOutOffline] = eventAnalyze
	// gMoleculerService.Events[cWsConnectorOutOffline] = eventWsConnectorOutOffline
	// gMoleculerService.Events[cWsConnectorOutSyncUsersInfo] = eventWsConnectorOutSyncUsersInfo

	return *gMoleculerService
}

// //mol $ call ws-sender.send --ids gotest-user-0,gotest-user-1 --data.mid m123 --data.msg.a abc --data.msg.b 111 --data.msg.c true
// func actionSend(req *protocol.MsRequest) (interface{}, error) {
// 	log.Info("run actionSend, req.Params = ", req.Params)
// 	return nil, nil
// }

func eventAnalyze(req *protocol.MsEvent) {
	jsonByte, err := jsoniter.Marshal(req)
	if err != nil {
		log.Warn("run eventAnalyze, parse req to jsonByte error: ", err)
		return
	}
	log.Info("eventAnalyze: ", string(jsonByte))
}

// func eventWsConnectorOutSyncUsersInfo(req *protocol.MsEvent) {
// 	log.Info("run eventWsConnectorOutSyncUsersInfo")
// }
