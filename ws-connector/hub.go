// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"time"

	scheduler "github.com/singchia/go-scheduler"
)

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	// Registered clients.
	clients map[string]*Client

	// Inbound messages from the clients.
	broadcast chan []byte

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client

	hubClosed chan int

	inMsgHandlerPool  *scheduler.Scheduler
	outMsgHandlerPool *scheduler.Scheduler
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
		hubClosed:  make(chan int, 10),
		broadcast:  make(chan []byte, 1000),
		register:   make(chan *Client, 1000),
		unregister: make(chan *Client, 1000),
		clients:    make(map[string]*Client),
	}
	hub.inMsgHandlerPool = scheduler.NewScheduler()
	hub.inMsgHandlerPool.Interval = time.Millisecond * 200
	hub.inMsgHandlerPool.SetMaxGoroutines(10)
	hub.inMsgHandlerPool.SetMaxProcessedReqs(600) //1000 / 200 * 600 = 3000 r/s
	hub.inMsgHandlerPool.SetDefaultHandler(inMsgHandler)
	hub.inMsgHandlerPool.SetMonitor(inMsgSchedulerMonitor)
	hub.inMsgHandlerPool.StartSchedule()
	return hub
}

func inMsgSchedulerMonitor(incomingReqsDiff, processedReqsDiff, diff, currentGotoutines int64) {
	if incomingReqsDiff != 0 || processedReqsDiff != 0 {
		log.Printf("inMsgSchedulerMonitor: %d, %d, %d, %d\n", incomingReqsDiff, processedReqsDiff, diff, currentGotoutines)
	}
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
	log.Infof("Hub handleMessage from client[%s] msgType[%d] msg: %s\n", m.c.ID, m.msgType, m.msg)
	for _, client := range m.h.clients {
		client.send(m.msg)
	}

	//test only
	msgStr := string(m.msg)
	if msgStr == "close" {
		log.Warn("Hub Close all client on test close message")
		for _, client := range m.h.clients {
			client.close()
		}
	}
}

func (h *Hub) close() {
	for index := 0; index < 10; index++ {
		h.hubClosed <- 1
	}
	for _, client := range h.clients {
		client.close()
	}
}

func (h *Hub) handleClientMessage(client *Client, msgType int, msg []byte) {

	h.inMsgHandlerPool.PublishRequest(&scheduler.Request{Data: &inMsg{
		h:       h,
		c:       client,
		msgType: msgType,
		msg:     msg,
	}})
	// go func() {
	// 	log.Infof("Hub handleMessage from client[%s] msgType[%d] msg: %s\n", client.ID, msgType, msg)
	// 	for _, client := range h.clients {
	// 		client.send(msg)
	// 	}

	// 	//test only
	// 	msgStr := string(msg)
	// 	if msgStr == "close" {
	// 		log.Warn("Hub Close all client on test close message")
	// 		for _, client := range h.clients {
	// 			client.close()
	// 		}
	// 	}
	// }()
}

func (h *Hub) run() {
	go func() {
		for {
			select {
			case client := <-h.register:
				h.clients[client.ID] = client
			case <-h.hubClosed:
				return
			}
		}
	}()
	go func() {
		for {
			select {
			case client := <-h.unregister:
				if _, ok := h.clients[client.ID]; ok {
					delete(h.clients, client.ID)
					client.close()
				}
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
	// 	case client := <-h.register:
	// 		h.clients[client] = true
	// 	case client := <-h.unregister:
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
