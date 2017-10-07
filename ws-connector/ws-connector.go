package main

import (
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/json-iterator/go"

	moleculer "github.com/roytan883/moleculer-go"
	"github.com/roytan883/moleculer-go/protocol"
)

var gMoleculerService *moleculer.Service

func createMoleculerService() moleculer.Service {
	gMoleculerService = &moleculer.Service{
		ServiceName: AppName,
		Actions:     make(map[string]moleculer.RequestHandler),
		Events:      make(map[string]moleculer.EventHandler),
	}

	//init actions handlers
	// gMoleculerService.Actions["push"] = push

	//init listen events handlers
	// gMoleculerService.Events["eventA"] = onEventA
	gMoleculerService.Events[AppName+".push"] = onEventPush

	return *gMoleculerService
}

func serveHome(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL)
	if r.URL.Path != "/" {
		http.Error(w, "Not found", 404)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}
	http.ServeFile(w, r, "home.html")
}

var gHub *Hub

func startWsService() {
	gHub = newHub()
	go gHub.run()
	http.HandleFunc("/", serveHome)
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(gHub, w, r)
	})
	listenHost := ":" + strconv.Itoa(gPort)
	go func() {
		err := http.ListenAndServe(listenHost, nil)
		if err != nil {
			log.Fatal("exit process, http ListenAndServe err: ", err)
			os.Exit(1)
		}
	}()
}

func stopWsService() {
	if gHub != nil {
		gHub.close()
	}
}

func push(req *protocol.MsRequest) (interface{}, error) {
	log.Info("run action push, req.Params = ", req.Params)
	data := map[string]interface{}{
		"res1": "AAA",
		"res2": 123,
	}
	return data, nil
	// return nil, errors.New("test return error in push")
}

type pushMsgStruct struct {
	IDs  interface{} `json:"ids"`
	Data interface{} `json:"data"`
}

func onEventPush(req *protocol.MsEvent) {
	log.Info("run onEventPush, req.Data = ", req.Data)
	jsonString, err := jsoniter.Marshal(req.Data)
	if err != nil {
		log.Warn("run onEventPush, parse req.Data to jsonString error: ", err)
		return
	}
	log.Info("jsonString = ", string(jsonString))
	// jsonString = []byte("{\"ids\":[\"uaaa\",\"bbb\"],\"data\":\"abc123\"}")
	// log.Info("jsonString = ", string(jsonString))
	jsonObj := &pushMsgStruct{}
	err = jsoniter.Unmarshal(jsonString, jsonObj)
	if err != nil {
		log.Warn("run onEventPush, parse req.Data to jsonObj error: ", err)
		return
	}
	log.Info("jsonObj = ", jsonObj)
	log.Info("jsonObj IDs = ", jsonObj.IDs)
	log.Info("jsonObj Data = ", jsonObj.Data)
	data, err := jsoniter.Marshal(jsonObj.Data)
	if err != nil {
		log.Warn("run onEventPush, parse jsonObj.Data to data error: ", err)
		return
	}

	// log.Info("type:", reflect.TypeOf(jsonObj.IDs))

	switch jsonObj.IDs.(type) {
	case []string:
		gHub.sendMessage(jsonObj.IDs.([]string), data)
	case []interface{}:
		ids := make([]string, 0)
		_ids := jsonObj.IDs.([]interface{})
		for _, v := range _ids {
			if sv, ok := v.(string); ok {
				ids = append(ids, sv)
			}
		}
		gHub.sendMessage(ids, data)
	case string:
		ids := strings.Split(jsonObj.IDs.(string), ",")
		gHub.sendMessage(ids, data)
	default:
		log.Info("can't parse  jsonObj.IDs")
	}

	// if ids1, ok := jsonObj.IDs.(stringArray); ok {
	// 	gHub.sendMessage(ids1, data)
	// 	return
	// }
	// if ids2, ok := jsonObj.IDs.(string); ok {
	// 	ids3 := strings.Split(ids2, ",")
	// 	gHub.sendMessage(ids3, data)
	// 	return
	// }
	// log.Info("can't parse  jsonObj.IDs")
	// gHub.sendMessage(jsonObj.IDs, []byte(jsonObj.Data))
}
