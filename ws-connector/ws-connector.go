package main

import (
	"net/http"
	"strconv"

	moleculer "github.com/roytan883/moleculer-go"
	"github.com/roytan883/moleculer-go/protocol"
)

var gService *moleculer.Service

func createService() moleculer.Service {
	gService = &moleculer.Service{
		ServiceName: AppName,
		Actions:     make(map[string]moleculer.RequestHandler),
		Events:      make(map[string]moleculer.EventHandler),
	}

	//init actions handlers
	gService.Actions["push"] = push

	//init listen events handlers
	gService.Events["eventA"] = onEventA

	createWsService()

	return *gService
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

func createWsService() {
	hub := newHub()
	go hub.run()
	http.HandleFunc("/", serveHome)
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})
	listenHost := ":" + strconv.Itoa(gPort)
	go func() {
		err := http.ListenAndServe(listenHost, nil)
		if err != nil {
			log.Fatal("ListenAndServe: ", err)
		}
	}()
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
