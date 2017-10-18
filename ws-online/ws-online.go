package main

import (
	"errors"
	"strconv"
	"strings"
	"time"

	// _ "net/http/pprof" //https://localhost:12020/debug/pprof

	"sync"

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
	gMoleculerService.Actions["onlineStatus"] = actionOnlineStatus
	gMoleculerService.Actions["onlineStatusBulk"] = actionOnlineStatusBulk

	//init listen events handlers
	gMoleculerService.Events[cWsConnectorOutOnline] = eventWsConnectorOutOnline
	gMoleculerService.Events[cWsConnectorOutOffline] = eventWsConnectorOutOffline
	gMoleculerService.Events[cWsConnectorOutSyncUsersInfo] = eventWsConnectorOutSyncUsersInfo

	gShortOnlineHub = &ShortOnlineHub{
		Users:        &sync.Map{},
		AbandonUsers: &sync.Map{},
		hubClosed:    make(chan int, 1),
	}
	gShortOnlineHub.runCheckAbandonUsers()

	time.AfterFunc(time.Second*time.Duration(gSyncDelaySeconds), func() {
		pBroker.Broadcast(cWsConnectorInSyncUsersInfo, nil)
	})

	return *gMoleculerService
}

//mol $ call ws-online.onlineStatus --userID gotest-user-0
func actionOnlineStatus(req *protocol.MsRequest) (interface{}, error) {

	userID := parseUserID(req)
	if len(userID) < 1 {
		return nil, errors.New("Parse userID error")
	}

	log.Info("run actionOnlineStatus: ", userID)

	return getOnlineStatus(userID), nil
}

//mol $ call ws-online.onlineStatusBulk --ids gotest-user-0,gotest-user-1
func actionOnlineStatusBulk(req *protocol.MsRequest) (interface{}, error) {

	jsonByte, err := jsoniter.Marshal(req.Params)
	if err != nil {
		log.Warn("run actionOnlineStatusBulk, parse req.Data to jsonByte error: ", err)
		return nil, err
	}
	jsonObj := &idsStruct{}
	err = jsoniter.Unmarshal(jsonByte, jsonObj)
	if err != nil {
		log.Warn("run actionOnlineStatusBulk, parse req.Data to jsonObj idsStruct error: ", err)
		return nil, err
	}

	log.Info("run actionOnlineStatusBulk jsonObj.IDs: ", jsonObj.IDs)

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

	onlineStatusBulk := &onlineStatusBulkStruct{
		OnlineStatusBulk: make([]*onlineStatusStruct, 0),
	}
	for _, userID := range ids {
		onlineStatus := getOnlineStatus(userID)
		onlineStatusBulk.OnlineStatusBulk = append(onlineStatusBulk.OnlineStatusBulk, onlineStatus)
	}

	return onlineStatusBulk, nil
}

func getOnlineStatus(userID string) *onlineStatusStruct {
	onlineStatus := &onlineStatusStruct{
		UserID:          userID,
		RealOnlineInfos: make([]*ClientInfo, 0),
	}

	if userInfo, ok := gShortOnlineHub.Users.Load(userID); ok {
		onlineStatus.IsShortOnline = true
		userInfoObj, ok := userInfo.(*UserInfo)
		if ok {
			realOnlineInfos := make([]*ClientInfo, 0)
			userInfoObj.Clients.Range(func(key, value interface{}) bool {
				if clientInfo, ok := value.(*ClientInfo); ok {
					if clientInfo.IsOnline {
						onlineStatus.IsRealOnline = true
					}
					realOnlineInfos = append(realOnlineInfos, clientInfo)
				}
				return true
			})
			onlineStatus.RealOnlineInfos = realOnlineInfos
		}
	}
	return onlineStatus
}

func parseUserID(req *protocol.MsRequest) string {
	jsonByte, err := jsoniter.Marshal(req.Params)
	if err != nil {
		log.Warn("run parseUserID, parse req.Data to jsonByte error: ", err)
		return ""
	}
	jsonObj := &userIDStruct{}
	err = jsoniter.Unmarshal(jsonByte, jsonObj)
	if err != nil {
		log.Warn("run parseUserID, parse req.Data to jsonObj userIDStruct error: ", err)
		return ""
	}
	return jsonObj.UserID
}

func eventWsConnectorOutOffline(req *protocol.MsEvent) {
	log.Info("run eventWsConnectorOutOffline")
	handlerClientInfo(req)
}

func eventWsConnectorOutOnline(req *protocol.MsEvent) {
	log.Info("run eventWsConnectorOutOnline")
	handlerClientInfo(req)
}

func eventWsConnectorOutSyncUsersInfo(req *protocol.MsEvent) {
	log.Info("run eventWsConnectorOutSyncUsersInfo")
	handlerClientInfo(req)
}

//UserInfo ...
type UserInfo struct {
	UserID          string
	LastOnlineTime  time.Time
	LastOfflineTime time.Time
	LastClientInfo  *ClientInfo
	Clients         *sync.Map //~= sync.Map[string(Cid)]*ClientInfo //only real online clientInfos

}

var gShortOnlineHub *ShortOnlineHub

//ShortOnlineHub ...
type ShortOnlineHub struct {
	Users        *sync.Map //~= sync.Map[string(UserID)]*UserInfo
	AbandonUsers *sync.Map //~= sync.Map[string(UserID)]time.Time(UserInfo.LastOfflineTime)
	hubClosed    chan int
}

func handlerClientInfo(req *protocol.MsEvent) {
	jsonByte, err := jsoniter.Marshal(req.Data)
	if err != nil {
		log.Warn("handlerClientInfo, parse req.Data to jsonByte error: ", err)
		return
	}
	clientInfo := &ClientInfo{}
	err = jsoniter.Unmarshal(jsonByte, clientInfo)
	if err != nil {
		log.Warn("handlerClientInfo, parse req.Data to ClientInfo error: ", err)
		return
	}
	if clientInfo == nil || len(clientInfo.NodeID) < 1 || len(clientInfo.Cid) < 1 || len(clientInfo.UserID) < 1 || len(clientInfo.ConnectTime) < 1 || len(clientInfo.DisconnectTime) < 1 {
		return
	}

	onlineTimestamp, err := strconv.Atoi(clientInfo.ConnectTime)
	if err != nil {
		log.Warn("handlerClientInfo, parseclientInfo.ConnectTime error: ", err)
		return
	}
	offlineTimestamp, err := strconv.Atoi(clientInfo.DisconnectTime)
	if err != nil {
		log.Warn("handlerClientInfo, parseclientInfo.ConnectTime error: ", err)
		return
	}
	onlineTime := time.Unix(int64(onlineTimestamp/1e3), int64(onlineTimestamp%1e3*1e6))
	offlineTime := time.Unix(int64(offlineTimestamp/1e3), int64(offlineTimestamp%1e3*1e6))

	newUserInfo := &UserInfo{
		UserID:          clientInfo.UserID,
		LastOnlineTime:  onlineTime,
		LastOfflineTime: offlineTime,
		Clients:         &sync.Map{},
	}
	log.Info("handlerClientInfo newUserInfo = ", newUserInfo)

	isOnline := newUserInfo.LastOnlineTime.After(newUserInfo.LastOfflineTime)
	clientInfo.IsOnline = isOnline

	userInfo, isOld := gShortOnlineHub.Users.LoadOrStore(clientInfo.UserID, newUserInfo)
	userInfoObj, ok := userInfo.(*UserInfo)
	if !ok {
		log.Warn("handlerClientInfo, LoadOrStore cast to userInfo error")
		return
	}

	if !isOld && isOnline {
		pBroker.Broadcast(cWsOnlineOutOnline, clientInfo)
	}

	userInfoObj.LastClientInfo = clientInfo

	if isOnline {
		userInfoObj.LastOnlineTime = newUserInfo.LastOnlineTime
		userInfoObj.Clients.Store(clientInfo.Cid, clientInfo)
		gShortOnlineHub.AbandonUsers.Delete(userInfoObj.UserID)
	} else {
		userInfoObj.LastOfflineTime = newUserInfo.LastOfflineTime
		userInfoObj.Clients.Store(clientInfo.Cid, clientInfo)
		gShortOnlineHub.AbandonUsers.Store(clientInfo.Cid, &abandonStruct{
			UserID:          userInfoObj.UserID,
			Cid:             clientInfo.Cid,
			LastOfflineTime: newUserInfo.LastOfflineTime,
		})
		// userInfoObj.Clients.Delete(clientInfo.Cid)
		// count := 0
		// userInfoObj.Clients.Range(func(key, value interface{}) bool {
		// 	if info, ok := value.(*ClientInfo); ok {
		// 		if info.IsOnline {
		// 			count++
		// 		}
		// 	}
		// 	return true
		// })
		// if count < 1 {
		// 	userInfoObj.LastOfflineTime = newUserInfo.LastOfflineTime
		// 	gShortOnlineHub.AbandonUsers.Store(map[string]string{
		// 		"UserID": userInfoObj.UserID,
		// 		"Cid":    clientInfo.Cid,
		// 	}, userInfoObj.LastOfflineTime)
		// }
	}
}

//Close ...
func (h *ShortOnlineHub) Close() {
	h.hubClosed <- 1
}

func (h *ShortOnlineHub) runCheckAbandonUsers() {
	go func() {
		ticker := time.NewTicker(time.Minute * 1)
		// ticker := time.NewTicker(time.Second * 1)
		for {
			select {
			case <-ticker.C:
				now := time.Now()
				gShortOnlineHub.AbandonUsers.Range(func(key, value interface{}) bool {
					// userID, ok := key.(string)
					abandon, ok := value.(*abandonStruct)
					if ok {
						diff := now.Sub(abandon.LastOfflineTime)
						// if diff > time.Second*10 {
						if diff > (time.Minute * time.Duration(gAbandonMinutes)) {
							log.Warn("Abandon User Cid: ", abandon.Cid)
							gShortOnlineHub.AbandonUsers.Delete(abandon.Cid)
							userInfo, ok := gShortOnlineHub.Users.Load(abandon.UserID)
							if ok {
								userInfoObj, ok := userInfo.(*UserInfo)
								if ok {
									userInfoObj.Clients.Delete(abandon.Cid)
									hasOtherClients := false
									userInfoObj.Clients.Range(func(key, value interface{}) bool {
										hasOtherClients = true
										return false
									})
									if !hasOtherClients {
										gShortOnlineHub.Users.Delete(abandon.UserID)
										pBroker.Broadcast(cWsOnlineOutOffline, userInfoObj.LastClientInfo)
									}
								}
							}
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
