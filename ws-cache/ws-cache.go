package main

import (
	"sync"
	"time"

	// _ "net/http/pprof" //https://localhost:12020/debug/pprof

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
	gMoleculerService.Actions["save"] = actionSave

	//init listen events handlers
	gMoleculerService.Events[cWsConnectorOutOnline] = eventWsConnectorOutOnline

	//init hub
	gMyHub = &MyHub{
		cachedMsgs:     &sync.Map{},
		userCachedMids: &sync.Map{},
		hubClosed:      make(chan int, 1),
	}
	gMyHub.run()

	return *gMoleculerService
}

//mol $ call ws-sender.send --ids gotest-user-0,gotest-user-1 --data.mid m123 --data.msg.a abc --data.msg.b 111 --data.msg.c true
func actionSave(req *protocol.MsRequest) (interface{}, error) {
	log.Info("run actionSave")

	jsonByte, err := jsoniter.Marshal(req.Params)
	if err != nil {
		log.Warn("run actionSave, parse req.Data to jsonByte error: ", err)
		return nil, err
	}
	jsonObj := &cacheMsgStruct{}
	err = jsoniter.Unmarshal(jsonByte, jsonObj)
	if err != nil {
		log.Warn("run actionSave, parse jsonByte to jsonObj cacheMsgStruct error: ", err)
		return nil, err
	}
	jsonObj.saveTime, err = timestampToTime(jsonObj.Timestamp)
	if err != nil {
		log.Warn("run actionSave, parse timestampToTime error: ", err)
		return nil, err
	}
	jsonObj.umid = genUniqueMid(jsonObj.UserID, jsonObj.Cid, jsonObj.Mid)
	log.Info("run actionSave, Store msg: ", jsonObj)
	gMyHub.cachedMsgs.Store(jsonObj.umid, jsonObj)

	newOneUserCachedMids := &sync.Map{}
	oneUserCachedMids, _ := gMyHub.userCachedMids.LoadOrStore(jsonObj.UserID, newOneUserCachedMids)
	if oneUserCachedMidsObj, ok := oneUserCachedMids.(*sync.Map); ok {
		oneUserCachedMidsObj.Store(jsonObj.umid, struct{}{})
	}

	return nil, nil
}

func eventWsConnectorOutOnline(req *protocol.MsEvent) {
	log.Info("run eventWsConnectorOutOnline")

	jsonByte, err := jsoniter.Marshal(req.Data)
	if err != nil {
		log.Warn("run eventWsConnectorOutOnline, parse req.Data to jsonByte error: ", err)
		return
	}
	jsonObj := &ClientInfo{}
	err = jsoniter.Unmarshal(jsonByte, jsonObj)
	if err != nil {
		log.Warn("run eventWsConnectorOutOnline, parse jsonByte to jsonObj ClientInfo error: ", err)
		return
	}

	log.Info("run eventWsConnectorOutOnline, online client : ", jsonObj)

	oneUserCachedMids, ok := gMyHub.userCachedMids.Load(jsonObj.UserID)
	if ok {
		if oneUserCachedMidsObj, ok := oneUserCachedMids.(*sync.Map); ok {
			oneUserCachedMidsObj.Range(func(key, value interface{}) bool {
				if umid, ok := key.(string); ok {
					cacheMsg, ok := gMyHub.cachedMsgs.Load(umid)
					if ok {
						gMyHub.cachedMsgs.Delete(umid)
						if cacheMsgObj, ok := cacheMsg.(*cacheMsgStruct); ok {
							pBroker.Call(cWsSenderActionSend, &pushMsgStruct{
								IDs: cacheMsgObj.UserID,
								Data: &pushMsgDataStruct{
									Mid: cacheMsgObj.Mid,
									Msg: cacheMsgObj.Msg,
								},
							}, nil)
							log.Infof("re-send: UserID[%s] Cid[%s] Mid[%s]\n", cacheMsgObj.UserID, cacheMsgObj.Cid, cacheMsgObj.Mid)
						}
					}
				}
				return true
			})
		}
	}
	gMyHub.userCachedMids.Delete(jsonObj.UserID)
}

var gMyHub *MyHub

//MyHub ...
type MyHub struct {
	cachedMsgs     *sync.Map //sync.Map[string(Mid)]*cacheMsgStruct
	userCachedMids *sync.Map //sync.Map[string(UserID)]*sync.Map[string(Mid)]struct{}
	hubClosed      chan int
}

func (h *MyHub) run() {
	go func() {
		ticker := time.NewTicker(time.Minute * 1)
		for {
			select {
			case <-ticker.C:
				now := time.Now()
				cacheCount := 0
				h.cachedMsgs.Range(func(key1, value interface{}) bool {
					cacheCount++
					if cacheMsg, ok := value.(*cacheMsgStruct); ok {
						diff := now.Sub(*cacheMsg.saveTime)
						if diff > time.Second*time.Duration(gMaxCacheSeconds) {
							cacheCount--
							h.cachedMsgs.Delete(key1)
							log.Info("delete timeout cached umid: ", cacheMsg)
							log.Info("delete timeout cached msg: ", cacheMsg)

							oneUserCachedMids, ok := gMyHub.userCachedMids.Load(cacheMsg.UserID)
							if ok {
								if oneUserCachedMidsObj, ok := oneUserCachedMids.(*sync.Map); ok {
									count := 0
									oneUserCachedMidsObj.Range(func(key2, value interface{}) bool {
										count++
										umid1, ok1 := key1.(string)
										umid2, ok2 := key2.(string)
										if ok1 && ok2 && umid1 == umid2 {
											oneUserCachedMidsObj.Delete(key2)
										}
										return true
									})
									if count == 0 {
										gMyHub.userCachedMids.Delete(cacheMsg.UserID)
									}
								}
							}
						}

						// else {
						// 	res, err := pBroker.Call(cWsOnlineActionOnlineStatus, &userIDStruct{
						// 		UserID: cacheMsg.UserID,
						// 	}, nil)
						// 	if err != nil {
						// 		log.Warn("get ws-online.onlineStatus err: ", err)
						// 		return true
						// 	}
						// 	jsonByte, err := jsoniter.Marshal(res)
						// 	if err != nil {
						// 		log.Warn("parse res to jsonByte error: ", err)
						// 		return true
						// 	}
						// 	jsonObj := &onlineStatusStruct{}
						// 	err = jsoniter.Unmarshal(jsonByte, jsonObj)
						// 	if err != nil {
						// 		log.Warn("parse jsonByte to onlineStatusStruct error: ", err)
						// 		return true
						// 	}
						// 	if jsonObj.IsRealOnline {
						// 		pBroker.Call(cWsSenderActionSend, &pushMsgStruct{
						// 			IDs: cacheMsg.UserID,
						// 			Data: &pushMsgDataStruct{
						// 				Mid: cacheMsg.Mid,
						// 				Msg: cacheMsg.Msg,
						// 			},
						// 		}, nil)
						// 		log.Info("re-send: ", key)
						// 	}
						// }
					}
					return true
				})
				if cacheCount > 0 {
					log.Warn("cacheCount = ", cacheCount)
				}
			case <-h.hubClosed:
				return
			}
		}
	}()
}
