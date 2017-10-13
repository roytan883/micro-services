// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
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
	Cid            string `json:"cid"`
	UserID         string `json:"userID"`
	Platform       string `json:"platform"`
	Version        string `json:"version"`
	Timestamp      string `json:"timestamp"`
	Token          string `json:"token"`
	ConnectTime    string `json:"connectTime"`
	DisconnectTime string `json:"disconnectTime"`

	hub *Hub

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	sendChan chan []byte
	sendMu   sync.Mutex

	sendPongChan chan int

	closed int32
}

func (c *Client) send(data []byte) {
	if atomic.LoadInt32(&c.closed) > 0 {
		log.Warnf("client[%s] already closed, can't send: %s\n", c.Cid, string(data))
		return //already closed
	}
	log.Printf("client[%s] send: %s\n", c.Cid, string(data))
	c.sendChan <- data
}

func (c *Client) kick() {
	if atomic.LoadInt32(&c.closed) > 0 {
		return //already closed
	}
	log.Printf("client[%s] kick:", c.Cid)
	c.sendMu.Lock()
	c.conn.SetWriteDeadline(time.Now().Add(time.Second * 3))
	c.conn.WriteMessage(websocket.TextMessage, []byte("kicked"))
	c.sendMu.Unlock()
	c.close()
}

func (c *Client) close() {
	if !atomic.CompareAndSwapInt32(&c.closed, 0, 1) {
		return //already closed
	}
	atomic.AddInt32(&c.closed, 1)
	log.Printf("client[%s] close start", c.Cid)
	c.sendMu.Lock()
	c.conn.SetWriteDeadline(time.Now().Add(time.Second * 3))
	c.conn.WriteMessage(websocket.CloseMessage, []byte{})
	c.sendMu.Unlock()
	close(c.sendChan)
	close(c.sendPongChan)
	c.conn.Close()
	c.hub.unregister(c)
	log.Printf("client[%s] close end", c.Cid)
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
		// log.Infof("client[%s] PongHandler\n", c.Cid)
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	c.conn.SetPingHandler(func(string) error {
		// log.Infof("client[%s] PingHandler\n", c.Cid)
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		c.sendPongChan <- 1 //use writePump to send pong, avoid write conflict
		return nil
	})
	for {
		if atomic.LoadInt32(&c.closed) > 0 {
			log.Warnf("client[%s] exit readPump, c.closed already set\n", c.Cid)
			return //already closed
		}
		msgType, message, err := c.conn.ReadMessage()
		if err != nil {
			log.Warnf("client[%s] exit readPump, ReadMessage error: %v", c.Cid, err)
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
			log.Warnf("client[%s] exit writePump, c.closed already set\n", c.Cid)
			return //already closed
		}
		select {
		case message, ok := <-c.sendChan:

			if !ok {
				log.Warnf("client[%s] exit writePump, c.sendChan was closed\n", c.Cid)
				return
			}

			c.sendMu.Lock()
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			err := c.conn.WriteMessage(websocket.TextMessage, message)
			c.sendMu.Unlock()
			if err != nil {
				log.Warnf("client[%s] exit writePump, WriteMessage error = %v\n", c.Cid, err)
				return
			}

		case _, ok := <-c.sendPongChan:
			if !ok {
				log.Warnf("client[%s] exit writePump, c.sendPongChan was closed\n", c.Cid)
				return
			}

			c.sendMu.Lock()
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			err := c.conn.WriteMessage(websocket.PongMessage, []byte{})
			c.sendMu.Unlock()
			if err != nil {
				log.Warnf("client[%s] exit writePump, sendPong error = %v\n", c.Cid, err)
				return
			}

		case <-ticker.C:
			c.sendMu.Lock()
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			err := c.conn.WriteMessage(websocket.PingMessage, []byte{})
			c.sendMu.Unlock()
			if err != nil {
				log.Warnf("client[%s] exit writePump, Write PingMessage error = %v\n", c.Cid, err)
				return
			}
		}
	}
}
