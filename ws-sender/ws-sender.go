package main

import (

	// _ "net/http/pprof" //https://localhost:12020/debug/pprof

	"errors"
	"strings"

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
	gMoleculerService.Actions["send"] = actionSend

	//init listen events handlers
	// gMoleculerService.Events[cWsConnectorOutOnline] = eventWsConnectorOutOnline
	// gMoleculerService.Events[cWsConnectorOutOffline] = eventWsConnectorOutOffline
	// gMoleculerService.Events[cWsConnectorOutSyncUsersInfo] = eventWsConnectorOutSyncUsersInfo

	return *gMoleculerService
}

func actionSend(req *protocol.MsRequest) (interface{}, error) {

	log.Info("run actionSend, req.Params = ", req.Params)
	jsonByte, err := jsoniter.Marshal(req.Params)
	if err != nil {
		log.Warn("run actionPush, parse req.Data to jsonByte error: ", err)
		return nil, err
	}
	jsonObj := &pushMsgStruct{}
	err = jsoniter.Unmarshal(jsonByte, jsonObj)
	if err != nil {
		log.Warn("run actionSend, parse jsonByte to jsonObj error: ", err)
		return nil, err
	}
	log.Info("run actionSend, jsonObj = ", jsonObj)

	// log.Info("type:", reflect.TypeOf(jsonObj.IDs))

	ids := make([]string, 0)
	switch jsonObj.IDs.(type) {
	case []string:
		ids = jsonObj.IDs.([]string)
	case []interface{}:
		_ids := jsonObj.IDs.([]interface{})
		for _, v := range _ids {
			if sv, ok := v.(string); ok {
				ids = append(ids, sv)
			}
		}
	case string:
		ids = strings.Split(jsonObj.IDs.(string), ",")
	default:
		log.Info("can't parse jsonObj.IDs")
		return nil, errors.New("can't parse ids")
	}
	doSend(ids, jsonObj.Data)
	log.Info("actionSend ids = ", ids)
	log.Info("actionSend data = ", jsonObj.Data)
	return nil, nil
}

func doSend(ids []string, data *pushMsgDataStruct) {
	go func() {
		res, err := pBroker.Call(cWsOnlineActionOnlineStatusBulk, &idsStruct{
			IDs: ids,
		}, nil)
		log.Info("doSend res = ", res)
		log.Info("doSend err = ", err)
		jsonByte, err := jsoniter.Marshal(res)
		if err != nil {
			log.Warn("run doSend, parse res to jsonByte error: ", err)
			return
		}
		jsonObj := &onlineStatusBulkStruct{}
		err = jsoniter.Unmarshal(jsonByte, jsonObj)
		if err != nil {
			log.Warn("run doSend, parse jsonByte to onlineStatusBulkStruct error: ", err)
			return
		}
		log.Info("run doSend, onlineStatusBulkStruct = ", jsonObj)
		wsConnectorNodes := make(map[string]struct{})
		for _, onlineStatus := range jsonObj.OnlineStatusBulk {
			for _, clientInfo := range onlineStatus.RealOnlineInfos {
				if _, ok := wsConnectorNodes[clientInfo.NodeID]; !ok {
					wsConnectorNodes[clientInfo.NodeID] = struct{}{}
					break
				}
			}
		}
		log.Info("run doSend, wsConnectorNodes = ", wsConnectorNodes)
		for nodeID := range wsConnectorNodes {
			pBroker.Call(cWsConnectorActionPush, &pushMsgStruct{
				IDs:  ids,
				Data: data,
			}, &moleculer.CallOptions{
				NodeID: nodeID,
			})
		}
	}()
}

// func eventWsConnectorOutOffline(req *protocol.MsEvent) {
// 	log.Info("run eventWsConnectorOutOffline")
// }

// func eventWsConnectorOutOnline(req *protocol.MsEvent) {
// 	log.Info("run eventWsConnectorOutOnline")
// }

// func eventWsConnectorOutSyncUsersInfo(req *protocol.MsEvent) {
// 	log.Info("run eventWsConnectorOutSyncUsersInfo")
// }
