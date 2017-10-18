package main

import (
	"fmt"
	"sync"
	"time"

	// _ "net/http/pprof" //https://localhost:12020/debug/pprof

	"errors"
	"strings"

	jsoniter "github.com/json-iterator/go"
	moleculer "github.com/roytan883/moleculer-go"
	"github.com/roytan883/moleculer-go/protocol"
)

var pBroker *moleculer.ServiceBroker
var gMoleculerService *moleculer.Service

func setupMoleculerService() {
	//init service and broker
	config := &moleculer.ServiceBrokerConfig{
		NatsHost:              gNatsHosts,
		NodeID:                gNodeID,
		DefaultRequestTimeout: time.Second * 3,
		// LogLevel: moleculer.DebugLevel,
		LogLevel: moleculer.ErrorLevel,
		Services: make(map[string]moleculer.Service),
	}
	moleculerService := createMoleculerService()
	config.Services[ServiceName] = moleculerService
	broker, err := moleculer.NewServiceBroker(config)
	if err != nil {
		log.Fatalf("NewServiceBroker err: %v\n", err)
	}
	pBroker = broker
	err = broker.Start()
	if err != nil {
		log.Fatalf("exit process, broker.Start err: %v\n", err)
		return
	}
}

func createMoleculerService() moleculer.Service {
	gMoleculerService = &moleculer.Service{
		ServiceName: ServiceName,
		Actions:     make(map[string]moleculer.RequestHandler),
		Events:      make(map[string]moleculer.EventHandler),
	}

	//init actions handlers
	gMoleculerService.Actions["send"] = actionSend

	//init listen events handlers
	gMoleculerService.Events[cWsConnectorOutAck] = eventWsConnectorOutAck
	// gMoleculerService.Events[cWsConnectorOutOffline] = eventWsConnectorOutOffline
	// gMoleculerService.Events[cWsConnectorOutSyncUsersInfo] = eventWsConnectorOutSyncUsersInfo

	gLocalSaveHub = &LocalSaveHub{
		waitAckMsgs: &sync.Map{},
		hubClosed:   make(chan int, 1),
	}

	gLocalSaveHub.runCheckLocalSaveSend()

	return *gMoleculerService
}

//mol $ call ws-sender.send --ids gotest-user-0,gotest-user-1 --data.mid m123 --data.msg.a abc --data.msg.b 111 --data.msg.c true
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
		if err != nil {
			log.Warn("run doSend, get ids OnlineStatusBulk err: ", err)
			return
		}
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
		// log.Info("run doSend, onlineStatusBulkStruct = ", jsonObj)
		wsConnectorNodes := make(map[string][]string)
		for _, onlineStatus := range jsonObj.OnlineStatusBulk {
			for _, clientInfo := range onlineStatus.RealOnlineInfos {
				if clientInfo.IsOnline {
					// log.Info("run doSend, IsOnline = ", clientInfo)
					if nodes, ok := wsConnectorNodes[clientInfo.NodeID]; !ok {
						wsConnectorNodes[clientInfo.NodeID] = make([]string, 0)
						wsConnectorNodes[clientInfo.NodeID] = append(wsConnectorNodes[clientInfo.NodeID], clientInfo.UserID)
					} else {
						nodes = append(nodes, clientInfo.UserID)
						wsConnectorNodes[clientInfo.NodeID] = nodes
					}
					gLocalSaveHub.save(clientInfo.UserID, clientInfo.Cid, data.Mid, data)
				} else {
					saveToRemoteCache(clientInfo.UserID, clientInfo.Cid, data.Mid, data)
				}
			}
		}
		log.Info("run doSend, wsConnectorNodes = ", wsConnectorNodes)
		for nodeID, realIds := range wsConnectorNodes {
			log.Infof("run doSend, nodeID[%s] realIds[%v]", nodeID, realIds)
			pBroker.Call(cWsConnectorActionPush, &pushMsgStruct{
				IDs:  realIds,
				Data: data,
			}, &moleculer.CallOptions{
				NodeID: nodeID,
			})
		}
	}()
}

var gLocalSaveHub *LocalSaveHub

//LocalSaveHub ...
type LocalSaveHub struct {
	waitAckMsgs *sync.Map
	hubClosed   chan int
}

func (h *LocalSaveHub) save(userID string, cid string, mid string, data *pushMsgDataStruct) {
	key := fmt.Sprintf("%s.%s.%s", mid, userID, cid)
	h.waitAckMsgs.Store(key, &waitAckStruct{
		UserID:   userID,
		Cid:      cid,
		Mid:      mid,
		Data:     data,
		SendTime: time.Now(),
	})
}

func (h *LocalSaveHub) runCheckLocalSaveSend() {
	go func() {
		ticker := time.NewTicker(time.Second * 5)
		for {
			select {
			case <-ticker.C:
				now := time.Now()
				h.waitAckMsgs.Range(func(key, value interface{}) bool {
					if waitAck, ok := value.(*waitAckStruct); ok {
						diff := now.Sub(waitAck.SendTime)
						if diff > time.Second*time.Duration(gWaitAckSeconds) {
							h.waitAckMsgs.Delete(key)
							log.Info("move waitAckMsg to remote cache: ", key)
							saveToRemoteCache(waitAck.UserID, waitAck.Cid, waitAck.Mid, waitAck.Data)
						}
					}
					return true
				})
			case <-h.hubClosed:
				return
			}
		}
	}()
}

func saveToRemoteCache(userID string, cid string, mid string, data *pushMsgDataStruct) {
	//TODO: save msg to remote cache process
	log.Info("saveToRemoteCache userID = ", userID)
	log.Info("saveToRemoteCache cid = ", cid)
	log.Info("saveToRemoteCache mid = ", mid)
	log.Info("saveToRemoteCache data = ", data)
}

func eventWsConnectorOutAck(req *protocol.MsEvent) {
	log.Info("run eventWsConnectorOutAck")

	jsonByte, err := jsoniter.Marshal(req.Data)
	if err != nil {
		log.Warn("run eventWsConnectorOutAck, parse req.Data to jsonByte error: ", err)
		return
	}
	jsonObj := &ackStruct{}
	err = jsoniter.Unmarshal(jsonByte, jsonObj)
	if err != nil {
		log.Warn("run eventWsConnectorOutAck, parse jsonByte to jsonObj ackStruct error: ", err)
		return
	}
	if len(jsonObj.Aid) > 0 {
		key := fmt.Sprintf("%s.%s.%s", jsonObj.Aid, jsonObj.UserID, jsonObj.Cid)
		gLocalSaveHub.waitAckMsgs.Delete(key)
		log.Info("finish ACK: ", key)
	}
}

// func eventWsConnectorOutOnline(req *protocol.MsEvent) {
// 	log.Info("run eventWsConnectorOutOnline")
// }

// func eventWsConnectorOutSyncUsersInfo(req *protocol.MsEvent) {
// 	log.Info("run eventWsConnectorOutSyncUsersInfo")
// }
