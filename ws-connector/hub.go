// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"errors"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	jsoniter "github.com/json-iterator/go"
)

// Hub maintains the set of active clients
type Hub struct {
	// Registered clients.
	clients     sync.Map //~= sync.Map[string(Cid)]*Client
	userID2Cids sync.Map //~= sync.Map[string(UserID)]*sync.Map[string(Cid)]*Client

	// Register requests from the clients.
	registerChan chan *Client

	// UnregisterChan requests from clients.
	unregisterChan chan *Client

	hubClosed chan int

	inMsgHandlerPool  *RunGoPool
	outMsgHandlerPool *RunGoPool

	isDoingSyncUsersInfo int32
}

func newHub() *Hub {
	hub := &Hub{
		hubClosed:      make(chan int, 10),
		registerChan:   make(chan *Client, 1000),
		unregisterChan: make(chan *Client, 1000),
	}

	hub.inMsgHandlerPool = NewRunGoPool("hub.inMsgHandlerPool", 500, time.Millisecond*100, inMsgHandler)
	hub.inMsgHandlerPool.Start()
	hub.outMsgHandlerPool = NewRunGoPool("hub.outMsgHandlerPool", 500, time.Millisecond*100, outMsgHandler)
	hub.outMsgHandlerPool.Start()

	return hub
}

// 枚举
type inMsgType int

const (
	clientMsg     inMsgType = iota // value --> 0
	syncUsersInfo                  // value --> 1
)

func (it inMsgType) String() string {
	switch it {
	case clientMsg:
		return "clientMsg"
	case syncUsersInfo:
		return "syncUsersInfo"
	default:
		return "Unknow"
	}
}

type inMsg struct {
	h   *Hub
	c   *Client
	t   inMsgType
	msg []byte
}

func inMsgHandler(data interface{}) {
	m, ok := data.(*inMsg)
	if !ok {
		return
	}
	log.Infof("inMsgHandler: from client[%s] msgType[%s] msg: %s\n", m.c.Cid, m.t, m.msg)

	if m.t == clientMsg {
		jsonObj := &ackStruct{}
		err := jsoniter.Unmarshal(m.msg, jsonObj)
		if err == nil {
			if len(jsonObj.Aid) > 0 {
				log.Info("inMsgHandler, handle ACK = ", jsonObj.Aid)
				jsonObj.Cid = m.c.Cid
				jsonObj.UserID = m.c.UserID
				pBroker.Broadcast(cgBroadcastAck, jsonObj)
			}
			return
		}
		m.h.clients.Range(func(key, value interface{}) bool {
			value.(*Client).send(m.msg)
			return true
		})

		//test only
		msgStr := string(m.msg)
		if msgStr == "close" {
			log.Warn("Hub Close all client on test close message")
			m.h.clients.Range(func(key, value interface{}) bool {
				value.(*Client).close()
				return true
			})
		}

	}

	if m.t == syncUsersInfo {
		pBroker.Broadcast(cgBroadcastSyncUsersInfo, m.c)
		return
	}

}

func inMsgSchedulerMonitor(incomingReqsDiff, processedReqsDiff, diff, currentGotoutines int64) {
	if incomingReqsDiff != 0 || processedReqsDiff != 0 {
		log.Printf("inMsgSchedulerMonitor: %d, %d, %d, %d\n", incomingReqsDiff, processedReqsDiff, diff, currentGotoutines)
	}
}

type outMsg struct {
	h   *Hub
	ids []string
	msg interface{}
}

func outMsgHandler(data interface{}) {
	m, ok := data.(*outMsg)
	if !ok {
		return
	}

	// log.Infof("Hub outMsgHandler from client[%s] msgType[%d] msg: %s\n", m.c.Cid, m.msgType, m.msg)
	for _, clientID := range m.ids {
		// log.Infof("Hub outMsgHandler to client[%s]\n", clientID)
		if client, ok := m.h.clients.Load(clientID); ok {
			client.(*Client).send(m.msg)
		}
		if userID2Cids, ok := m.h.userID2Cids.Load(clientID); ok {
			userID2Cids.(*sync.Map).Range(func(key, value interface{}) bool {
				value.(*Client).send(m.msg)
				return true
			})
		}
	}
}

func outMsgSchedulerMonitor(incomingReqsDiff, processedReqsDiff, diff, currentGotoutines int64) {
	if incomingReqsDiff != 0 || processedReqsDiff != 0 {
		log.Printf("outMsgSchedulerMonitor: %d, %d, %d, %d\n", incomingReqsDiff, processedReqsDiff, diff, currentGotoutines)
	}
}

func (h *Hub) close() {
	h.clients.Range(func(key, value interface{}) bool {
		value.(*Client).close()
		return true
	})
	for index := 0; index < 10; index++ {
		h.hubClosed <- 1
	}
}

var gSendMessageCount uint64

func (h *Hub) sendMessage(ids []string, msg interface{}) {
	atomic.AddUint64(&gSendMessageCount, 1)
	if atomic.LoadUint64(&gSendMessageCount)%10000 == 0 {
		log.Info("Hub gSendMessageCount: ", gSendMessageCount)
	}
	h.outMsgHandlerPool.Add(&outMsg{
		h:   h,
		ids: ids,
		msg: msg,
	})

}

func (h *Hub) handleClientMessage(client *Client, msgType int, msg []byte) {
	h.inMsgHandlerPool.Add(&inMsg{
		h:   h,
		c:   client,
		t:   clientMsg,
		msg: msg,
	})
}

func (h *Hub) run() {

	//test out to client msg performance
	// go func() {
	// 	ticker := time.NewTicker(time.Millisecond * 1)
	// 	for {
	// 		select {
	// 		case <-ticker.C:
	// 			// log.Info("ALL Goroutine: ", runtime.NumGoroutine())
	// 			testData := map[string]interface{}{
	// 				"ids":  "uaaa, bbb",
	// 				"data": "abc1133",
	// 			}
	// 			pBroker.Broadcast(AppName+".push", testData)
	// 			pBroker.Broadcast(AppName+".push", testData)
	// 			pBroker.Broadcast(AppName+".push", testData)
	// 		case <-h.hubClosed:
	// 			return
	// 		}
	// 	}
	// }()

	go func() {
		for {
			select {
			case client := <-h.registerChan:
				//userID_platform save
				h.clients.Store(client.Cid, client)

				//userID save
				h.userID2Cids.LoadOrStore(client.UserID, &sync.Map{})
				userID2Cids, ok := h.userID2Cids.Load(client.UserID)
				if ok {
					userID2Cids.(*sync.Map).Store(client.Cid, client)
				}
			case client := <-h.unregisterChan:
				h.clients.Delete(client.Cid)
				userID2Cids, ok := h.userID2Cids.Load(client.UserID)
				if ok {
					userID2Cids.(*sync.Map).Delete(client.Cid)
				}
				count := 0
				userID2Cids.(*sync.Map).Range(func(key, value interface{}) bool {
					count++
					return true
				})
				if count == 0 {
					h.userID2Cids.Delete(client.UserID)
				}
			case <-h.hubClosed:
				return
			}
		}
	}()
}

func (h *Hub) register(c *Client) {
	if oldClient, ok := h.clients.Load(c.Cid); ok {
		log.Info("Hub: kick old when register client: ", c.Cid)
		oldClient.(*Client).kick()
	}
	h.registerChan <- c
	log.Warn("Hub register <<< client = ", c.String())
	pBroker.Broadcast(cgBroadcastUserOnline, c)
}

func (h *Hub) unregister(c *Client) {
	h.unregisterChan <- c
	c.DisconnectTime = strconv.Itoa(int(time.Now().UnixNano() / 1e6))
	log.Warn("Hub unregister ### client = ", c.String())
	pBroker.Broadcast(cgBroadcastUserOffline, c)
}

//call ws-connector.count
func (h *Hub) count() int {
	count := 0
	h.clients.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

//emit ws-connector.in.kickClient --cid uaaa_web3
func (h *Hub) kickClient(cid string) {
	log.Info("Hub kickClient: cid = ", cid)
	if oldClient, ok := h.clients.Load(cid); ok {
		oldClient.(*Client).kick()
	}
}

//emit ws-connector.in.kickUser --userID uaaa
func (h *Hub) kickUser(userID string) {
	log.Info("Hub kickUser: userID = ", userID)
	if userID2Cids, ok := h.userID2Cids.Load(userID); ok {
		userID2Cids.(*sync.Map).Range(func(key, value interface{}) bool {
			value.(*Client).kick()
			return true
		})
	}
}

//call ws-connector.getUserOnlineInfo --userID uaaa
func (h *Hub) getUserOnlineInfo(userID string) ([]*Client, error) {
	log.Info("Hub getUserOnlineInfo: userID = ", userID)
	clientInfo := make([]*Client, 0)
	if userID2Cids, ok := h.userID2Cids.Load(userID); ok {
		userID2Cids.(*sync.Map).Range(func(key, value interface{}) bool {
			client := value.(*Client)
			clientInfo = append(clientInfo, client)
			return true
		})
		return clientInfo, nil
	}
	return nil, errors.New("Cant find userID: " + userID)
}

//emit ws-connector.in.syncUsersInfo
func (h *Hub) syncUsersInfo() {
	log.Info("Hub syncUsersInfo")
	go func() {
		if atomic.CompareAndSwapInt32(&h.isDoingSyncUsersInfo, 0, 1) {
			h.clients.Range(func(key, value interface{}) bool {
				client := value.(*Client)
				h.inMsgHandlerPool.Add(&inMsg{
					h: h,
					c: client,
					t: syncUsersInfo,
				})
				time.Sleep(time.Millisecond * 1) //Do not send too fast
				return true
			})
			atomic.StoreInt32(&h.isDoingSyncUsersInfo, 0)
		} else {
			log.Warn("Hub syncUsersInfo already running")
		}
	}()
}
