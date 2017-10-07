// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 200 * time.Second * 1

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	// maxMessageSize = 1024 * 4
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	ID       string
	platform string
	token    string

	hub *Hub

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	sendChan chan []byte

	sendPongChan chan int

	closed int32
}

func (c *Client) send(data []byte) {
	if atomic.LoadInt32(&c.closed) > 0 {
		log.Warnf("client[%s] already closed, can't send: %s\n", c.ID, string(data))
		return //already closed
	}
	log.Printf("client[%s] send: %s\n", c.ID, string(data))
	c.sendChan <- data
}

func (c *Client) close() {
	if atomic.LoadInt32(&c.closed) > 0 {
		return //already closed
	}
	atomic.AddInt32(&c.closed, 1)
	log.Printf("client[%s] close start", c.ID)
	c.conn.SetWriteDeadline(time.Now().Add(time.Second * 3))
	c.conn.WriteMessage(websocket.CloseMessage, []byte{})
	close(c.sendChan)
	close(c.sendPongChan)
	c.conn.Close()
	c.hub.unregister <- c
	log.Printf("client[%s] close end", c.ID)
}

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *Client) readPump() {
	defer func() {
		c.close()
	}()
	// c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		log.Infof("client[%s] PongHandler\n", c.ID)
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	c.conn.SetPingHandler(func(string) error {
		log.Infof("client[%s] PingHandler\n", c.ID)
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		c.sendPongChan <- 1 //use writePump to send pong, avoid write conflict
		return nil
	})
	for {
		if atomic.LoadInt32(&c.closed) > 0 {
			log.Warnf("client[%s] exit readPump, c.closed already set\n", c.ID)
			return //already closed
		}
		msgType, message, err := c.conn.ReadMessage()
		if err != nil {
			log.Warnf("client[%s] exit readPump, ReadMessage error: %v", c.ID, err)
			return
		}
		if msgType == websocket.TextMessage {
			message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
			c.hub.handleClientMessage(c, msgType, message)
		}
	}
}

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.close()
	}()
	for {
		if atomic.LoadInt32(&c.closed) > 0 {
			log.Warnf("client[%s] exit writePump, c.closed already set\n", c.ID)
			return //already closed
		}
		select {
		case message, ok := <-c.sendChan:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				log.Warnf("client[%s] exit writePump, c.sendChan was closed\n", c.ID)
				return
			}

			err := c.conn.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				log.Warnf("client[%s] exit writePump, WriteMessage error = %v\n", c.ID, err)
				return
			}

		case _, ok := <-c.sendPongChan:
			if !ok {
				log.Warnf("client[%s] exit writePump, c.sendPongChan was closed\n", c.ID)
				return
			}

			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			err := c.conn.WriteMessage(websocket.PongMessage, []byte{})
			if err != nil {
				log.Warnf("client[%s] exit writePump, sendPong error = %v\n", c.ID, err)
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				log.Warnf("client[%s] exit writePump, Write PingMessage error = %v\n", c.ID, err)
				return
			}
		}
	}
}

var gClientID uint64

// serveWs handles websocket requests from the peer.
func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request) {

	log.Printf("serveWs url = %v", r.URL)

	queryValues := r.URL.Query()

	log.Printf("serveWs queryValues = %v", queryValues)
	if len(queryValues) < 1 {
		w.WriteHeader(401)
		w.Write([]byte("Need Query Values in URL"))
		return
	}
	userID := queryValues.Get("userID")
	if len(userID) < 1 {
		w.WriteHeader(401)
		w.Write([]byte("Need Query Values [userID] in URL"))
		return
	}

	platform := queryValues.Get("platform")
	if len(platform) < 1 {
		w.WriteHeader(401)
		w.Write([]byte("Need Query Values [platform] in URL"))
		return
	}

	token := queryValues.Get("token")
	if len(token) < 1 {
		w.WriteHeader(401)
		w.Write([]byte("Need Query Values [token] in URL"))
		return
	}
	//TODO: just parse id and platform from token, validate token, return fail

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Warn("websocket Upgrade connection err: ", err)
		// log.Println(err)
		return
	}

	atomic.AddUint64(&gClientID, 1)
	log.Warn("websocket connection times: ", atomic.LoadUint64(&gClientID))

	client := &Client{
		// ID:           strconv.Itoa(int(gClientID)),
		ID:           userID,
		platform:     platform,
		token:        token,
		hub:          hub,
		conn:         conn,
		sendChan:     make(chan []byte, 10),
		sendPongChan: make(chan int, 10),
	}
	client.hub.register <- client

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writePump()
	go client.readPump()
}