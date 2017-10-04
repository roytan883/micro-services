// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"sync/atomic"
	// "log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second * 4

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
	ID uint64

	hub *Hub

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte
}

func (c *Client) close() {
	log.Printf("client[%d] close", c.ID)
	c.conn.SetWriteDeadline(time.Now().Add(time.Second * 3))
	c.conn.WriteMessage(websocket.CloseMessage, []byte{})
	_, ok := <-c.send
	if ok {
		close(c.send)
	}
	c.conn.Close()
	c.hub.unregister <- c
}

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *Client) readPump() {
	defer func() {
		// c.hub.unregister <- c
		// c.conn.Close()
		c.close()
	}()
	// c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				log.Printf("client[%d] CloseGoingAway error: %v", c.ID, err)
			}
			log.Printf("client[%d] exit readPump, ReadMessage error: %v", c.ID, err)
			return
			// break
		}
		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		messageStr := string(message)
		log.Printf("client[%d] readPump message: %v\n", c.ID, messageStr)

		c.hub.broadcast <- message

		if messageStr == "close" {
			//TODO: only test close when receive "close" message
			log.Printf("client[%d] exit readPump, only test close message\n", c.ID)
			// c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			// c.conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
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
		// c.conn.Close()
		c.close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				log.Printf("client[%d] exit writePump, c.send chan was closed\n", c.ID)
				// c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			messageStr := string(message)
			log.Printf("client[%d] writePump message: %v\n", c.ID, messageStr)
			if messageStr == "close" {
				//TODO: only test close when receive "close" message
				log.Printf("client[%d] exit writePump, only test close message\n", c.ID)
				// c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			err := c.conn.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				log.Printf("client[%d] exit writePump, WriteMessage error = %v\n", c.ID, err)
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

var gClientID uint64

// serveWs handles websocket requests from the peer.
func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	//TODO: handle url identify user, return HTTP error
	//url: /ws?abc=123&def=aaa
	log.Printf("serveWs url = %v", r.URL)
	atomic.AddUint64(&gClientID, 1)
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	client := &Client{ID: gClientID, hub: hub, conn: conn, send: make(chan []byte, 256)}
	client.hub.register <- client

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writePump()
	go client.readPump()
}
