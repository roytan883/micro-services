package main

import (
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
	jsonObj := &cacheStruct{}
	err = jsoniter.Unmarshal(jsonByte, jsonObj)
	if err != nil {
		log.Warn("run actionSave, parse jsonByte to jsonObj cacheStruct error: ", err)
		return nil, err
	}
	return nil, nil
}

func eventWsConnectorOutOnline(req *protocol.MsEvent) {
	log.Info("run eventWsConnectorOutAck")

	jsonByte, err := jsoniter.Marshal(req.Data)
	if err != nil {
		log.Warn("run eventWsConnectorOutAck, parse req.Data to jsonByte error: ", err)
		return
	}
	jsonObj := &ClientInfo{}
	err = jsoniter.Unmarshal(jsonByte, jsonObj)
	if err != nil {
		log.Warn("run eventWsConnectorOutAck, parse jsonByte to jsonObj ClientInfo error: ", err)
		return
	}
}
