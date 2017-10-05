// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

// hub maintains the set of active clients and broadcasts messages to the
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
	return &Hub{
		hubClosed:  make(chan int, 10),
		broadcast:  make(chan []byte, 1000),
		register:   make(chan *Client, 1000),
		unregister: make(chan *Client, 1000),
		clients:    make(map[string]*Client),
		// cmdChan:    make(chan *gCommand, 1),
		// cmdClose:   make(chan int)
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
	go func() {
		log.Infof("Hub handleMessage from client[%s] msgType[%d] msg: %s\n", client.ID, msgType, msg)
		for _, client := range h.clients {
			client.send(msg)
		}

		//test only
		msgStr := string(msg)
		if msgStr == "close" {
			log.Warn("Hub Close all client on test close message")
			for _, client := range h.clients {
				client.close()
			}
		}
	}()
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
