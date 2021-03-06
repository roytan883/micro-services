package main

import (
	"crypto/tls"
	"time"

	"github.com/gorilla/websocket"
	jsoniter "github.com/json-iterator/go"
)

type WsClient struct {
	name string
	url  string
	conn *websocket.Conn
}

func NewWsClient(name string, url string) *WsClient {
	c := &WsClient{
		name: name,
		url:  url,
	}
	log.Info("NewWsClient name = ", name)
	log.Info("NewWsClient url = ", url)
	// wsDialer := websocket.DefaultDialer
	wsDialer := &websocket.Dialer{
		TLSClientConfig:  &tls.Config{InsecureSkipVerify: true},
		HandshakeTimeout: time.Second * 1,
	}
	// wsDialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	conn, _, err := wsDialer.Dial(url, nil)
	// log.Info("NewWsClient r = ", r)
	// log.Info("NewWsClient err = ", err)
	c.conn = conn
	if err != nil {
		log.Errorf("WsClient [%s] dial err: ", name, err)
		return nil
	}
	c.run()
	return c
}

func (c *WsClient) run() {
	go func() {
		defer c.Close()
		c.conn.SetCloseHandler(func(code int, text string) error {
			log.Infof("WsClient [%s] CloseHandler: code[%d] text[%s]", c.name, code, text)
			return nil
		})
		for {
			_, message, err := c.conn.ReadMessage()
			if err != nil {
				log.Println("WsClient read err: ", err)
				return
			}
			// log.Info("WsClient recv: ", string(message))
			jsonObj := &pushMsgDataStruct{}
			err = jsoniter.Unmarshal(message, jsonObj)
			if err == nil {
				if len(jsonObj.Mid) > 0 {
					ack := &ackStruct{
						Aid: jsonObj.Mid,
					}
					c.conn.WriteJSON(ack)
				}
			}
			// log.Printf("WsClient recv: %s", message)
		}
	}()
}

func (c *WsClient) Close() {
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
}

func aaa() {

}
