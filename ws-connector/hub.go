// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"sync"
	"sync/atomic"
	"time"

	jsoniter "github.com/json-iterator/go"
)

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	// Registered clients.
	clients        map[string]*Client
	clientIDs      map[string]map[string]*Client
	clientsRWMutex *sync.RWMutex

	// Inbound messages from the clients.
	broadcast chan []byte

	// Register requests from the clients.
	registerChan chan *Client

	// UnregisterChan requests from clients.
	unregisterChan chan *Client

	hubClosed chan int

	inMsgHandlerPool  *RunGoPool
	outMsgHandlerPool *RunGoPool
}

type gCmdType uint32

const (
	gCmdPush gCmdType = iota
	gCmdKick
	gCmdClose
)

type gCommand struct {
	cmdType  gCmdType
	clientID string
	message  []byte
}

func newHub() *Hub {
	hub := &Hub{
		hubClosed:      make(chan int, 10),
		broadcast:      make(chan []byte, 1000),
		registerChan:   make(chan *Client, 1000),
		unregisterChan: make(chan *Client, 1000),
		clients:        make(map[string]*Client),
		clientIDs:      make(map[string]map[string]*Client),
		clientsRWMutex: new(sync.RWMutex),
	}

	hub.inMsgHandlerPool = NewRunGoPool("hub.inMsgHandlerPool", 500, time.Millisecond*100, inMsgHandler)
	hub.inMsgHandlerPool.Start()
	hub.outMsgHandlerPool = NewRunGoPool("hub.outMsgHandlerPool", 500, time.Millisecond*100, outMsgHandler)
	hub.outMsgHandlerPool.Start()

	return hub
}

type inMsg struct {
	h       *Hub
	c       *Client
	msgType int
	msg     []byte
}

func inMsgHandler(data interface{}) {
	m, ok := data.(*inMsg)
	if !ok {
		return
	}
	log.Infof("inMsgHandler: from client[%s] msgType[%d] msg: %s\n", m.c.Cid, m.msgType, m.msg)

	jsonObj := &ackStruct{}
	err := jsoniter.Unmarshal(m.msg, jsonObj)
	if err == nil {
		if len(jsonObj.Aid) > 0 {
			log.Info("inMsgHandler, handle ACK = ", jsonObj.Aid)
			jsonObj.Cid = m.c.Cid
			jsonObj.UserID = m.c.UserID
			pBroker.Broadcast(cgBroadcastAck, jsonObj)
			return
		}
	}

	m.h.clientsRWMutex.RLock()
	for _, client := range m.h.clients {
		log.Info("inMsgHandler: check client = ", client.Cid)
		client.send(m.msg)
	}
	m.h.clientsRWMutex.RUnlock()

	//test only
	msgStr := string(m.msg)
	if msgStr == "close" {
		log.Warn("Hub Close all client on test close message")
		m.h.clientsRWMutex.RLock()
		for _, client := range m.h.clients {
			client.close()
		}
		m.h.clientsRWMutex.RUnlock()
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
	msg []byte
}

func outMsgHandler(data interface{}) {
	m, ok := data.(*outMsg)
	if !ok {
		return
	}

	// log.Infof("Hub outMsgHandler from client[%s] msgType[%d] msg: %s\n", m.c.Cid, m.msgType, m.msg)
	for _, clientID := range m.ids {
		// log.Infof("Hub outMsgHandler to client[%s]\n", clientID)
		m.h.clientsRWMutex.RLock()
		if client, ok := m.h.clients[clientID]; ok {
			client.send(m.msg)
		}
		if clientIDs, ok := m.h.clientIDs[clientID]; ok {
			for _, v := range clientIDs {
				v.send(m.msg)
			}
		}
		m.h.clientsRWMutex.RUnlock()
	}
}

func outMsgSchedulerMonitor(incomingReqsDiff, processedReqsDiff, diff, currentGotoutines int64) {
	if incomingReqsDiff != 0 || processedReqsDiff != 0 {
		log.Printf("outMsgSchedulerMonitor: %d, %d, %d, %d\n", incomingReqsDiff, processedReqsDiff, diff, currentGotoutines)
	}
}

func (h *Hub) close() {
	for _, client := range h.clients {
		client.close()
	}
	for index := 0; index < 10; index++ {
		h.hubClosed <- 1
	}
}

var gSendMessageCount uint64

func (h *Hub) sendMessage(ids []string, msg []byte) {
	atomic.AddUint64(&gSendMessageCount, 1)
	if atomic.LoadUint64(&gSendMessageCount)%10000 == 0 {
		log.Info("Hub gSendMessageCount: ", gSendMessageCount)
	}
	// log.Info("Hub sendMessage ids: ", ids)
	// log.Info("Hub sendMessage msg: ", msg)
	// log.Info("Hub sendMessage msg: ", string(msg))

	// for _, clientID := range ids {
	// 	h.clientsRWMutex.RLock()
	// 	if client, ok := h.clients[clientID]; ok {
	// 		client.sendChan <- msg
	// 	}
	// 	h.clientsRWMutex.RUnlock()
	// }

	h.outMsgHandlerPool.Add(&outMsg{
		h:   h,
		ids: ids,
		msg: msg,
	})

}

func (h *Hub) handleClientMessage(client *Client, msgType int, msg []byte) {

	// go func() {
	// 	log.Infof("Hub handleMessage from client[%s] msgType[%d] msg: %s\n", client.ID, msgType, msg)
	// 	for _, client := range h.clients {
	// 		client.send(msg)
	// 	}
	// }()

	h.inMsgHandlerPool.Add(&inMsg{
		h:       h,
		c:       client,
		msgType: msgType,
		msg:     msg,
	})

	//test only
	msgStr := string(msg)
	if msgStr == "close" {
		log.Warn("Hub Close all client on test close message")
		for _, client := range h.clients {
			client.close()
		}
	}

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
				h.clientsRWMutex.Lock()

				//userID_platform save
				h.clients[client.Cid] = client

				//userID save
				_, ok := h.clientIDs[client.UserID]
				if !ok {
					h.clientIDs[client.UserID] = make(map[string]*Client)
				}
				clientIDs, ok := h.clientIDs[client.UserID]
				if ok {
					clientIDs[client.Cid] = client
				}

				h.clientsRWMutex.Unlock()
			case <-h.hubClosed:
				return
			}
		}
	}()
	go func() {
		for {
			select {
			case client := <-h.unregisterChan:
				h.clientsRWMutex.RLock()
				if _, ok := h.clients[client.Cid]; ok {
					h.clientsRWMutex.Lock()

					//userID_platform delete
					delete(h.clients, client.Cid)

					//userID delete
					clientIDs, ok := h.clientIDs[client.UserID]
					if ok {
						if _, ok := clientIDs[client.Cid]; ok {
							delete(clientIDs, client.Cid)
						}
						if len(clientIDs) < 1 {
							delete(h.clientIDs, client.UserID)
						}
					}

					h.clientsRWMutex.Unlock()
					client.close()
				}
				h.clientsRWMutex.RUnlock()
			case <-h.hubClosed:
				return
			}
		}
	}()
	// go func() {
	// 	for {
	// 		select {
	// 		case message := <-h.broadcast:
	// 			for _, client := range h.clients {
	// 				select {
	// 				case client.send <- message:
	// 				default:
	// 					close(client.send)
	// 					delete(h.clients, client.ID)
	// 				}
	// 			}
	// 		case <-h.hubClosed:
	// 			return
	// 		}
	// 	}
	// }()
	// for {
	// 	select {
	// 	case client := <-h.registerChan:
	// 		h.clients[client] = true
	// 	case client := <-h.unregisterChan:
	// 		if _, ok := h.clients[client]; ok {
	// 			delete(h.clients, client)
	// 			client.close()
	// 			// close(client.send)
	// 		}
	// 	case message := <-h.broadcast:
	// 		for client := range h.clients {
	// 			select {
	// 			case client.send <- message:
	// 			default:
	// 				close(client.send)
	// 				delete(h.clients, client)
	// 			}
	// 		}
	// 	}
	// }
}

func (h *Hub) register(c *Client) {
	h.registerChan <- c
	log.Info("Hub register: client = ", c)
	pBroker.Broadcast(cgBroadcastUserOnline, c)
}

func (h *Hub) unregister(c *Client) {
	h.unregisterChan <- c
	log.Info("Hub unregister: client = ", c)
	pBroker.Broadcast(cgBroadcastUserOffline, c)
}
