package main

import (
	"net/http"
	"strconv"

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
	gMoleculerService.Actions["push"] = push

	//init listen events handlers
	gMoleculerService.Events["eventA"] = onEventA

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
			log.Fatal("ListenAndServe: ", err)
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

func onEventA(req *protocol.MsEvent) {
	log.Info("run onEventA, req.Data = ", req.Data)
}
