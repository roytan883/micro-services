package main

import (
	moleculer "github.com/roytan883/moleculer-go"
	"github.com/roytan883/moleculer-go/protocol"
)

func createService() moleculer.Service {
	service := moleculer.Service{
		ServiceName: AppName,
		Actions:     make(map[string]moleculer.RequestHandler),
		Events:      make(map[string]moleculer.EventHandler),
	}

	//init actions handlers
	actionA := func(req *protocol.MsRequest) (interface{}, error) {
		log.Info("run actionA, req.Params = ", req.Params)
		data := map[string]interface{}{
			"res1": "AAA",
			"res2": 123,
		}
		return data, nil
		// return nil, errors.New("test return error in actionA")
	}
	actionB := func(req *protocol.MsRequest) (interface{}, error) {
		log.Info("run actionB, req.Params = ", req.Params)
		data := map[string]interface{}{
			"res1": "BBB",
			"res2": 456,
		}
		return data, nil
		// return nil, errors.New("test return error in actionB")
	}
	bench := func(req *protocol.MsRequest) (interface{}, error) {
		// log.Info("run actionB, req.Params = ", req.Params)
		data := map[string]interface{}{
			"res1": "CCC",
			"res2": 789,
		}
		return data, nil
		// return nil, errors.New("test return error in actionB")
	}
	service.Actions["actionA"] = actionA
	service.Actions["actionB"] = actionB
	service.Actions["bench"] = bench

	//init listen events handlers
	onEventUserCreate := func(req *protocol.MsEvent) {
		log.Info("run onEventUserCreate, req.Data = ", req.Data)
	}
	onEventUserDelete := func(req *protocol.MsEvent) {
		log.Info("run onEventUserDelete, req.Data = ", req.Data)
	}
	service.Events["user.create"] = onEventUserCreate
	service.Events["user.delete"] = onEventUserDelete

	return service
}
