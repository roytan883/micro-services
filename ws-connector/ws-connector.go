package main

import (
	"crypto/sha1"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync/atomic"
	// _ "net/http/pprof" //https://localhost:12220/debug/pprof
	"os"
	"runtime/pprof"
	"strconv"
	"strings"
	"time"

	"github.com/json-iterator/go"
	"golang.org/x/net/netutil"

	moleculer "github.com/roytan883/moleculer-go"
	"github.com/roytan883/moleculer-go/protocol"
)

var gMoleculerService *moleculer.Service

func createMoleculerService() moleculer.Service {
	gMoleculerService = &moleculer.Service{
		ServiceName: ServiceName,
		Actions:     make(map[string]moleculer.RequestHandler),
		Events:      make(map[string]moleculer.EventHandler),
	}

	//init actions handlers
	gMoleculerService.Actions["push"] = actionPush
	gMoleculerService.Actions["count"] = actionCount
	gMoleculerService.Actions["metrics"] = actionMetrics
	gMoleculerService.Actions["userInfo"] = actionUserInfo

	//init listen events handlers
	gMoleculerService.Events[cWsConnectorInKickClient] = eventInKickClient
	gMoleculerService.Events[cWsConnectorInKickUser] = eventInKickUser
	gMoleculerService.Events[cWsConnectorInPush] = eventInPush
	gMoleculerService.Events[cWsConnectorInSyncUsersInfo] = eventInSyncUsersInfo
	gMoleculerService.Events[cWsConnectorInSyncMetrics] = eventInSyncMetrics

	return *gMoleculerService
}

func actionCount(req *protocol.MsRequest) (interface{}, error) {
	log.Info("run actionCount")
	count := gHub.count()
	log.Info("run actionCount, count: ", count)
	return count, nil
}

//mol $ call ws-connector.metrics
func actionMetrics(req *protocol.MsRequest) (interface{}, error) {
	log.Info("run actionMetrics")
	metrics := gHub.metrics()
	log.Info("run actionMetrics, metrics: ", metrics)
	return metrics, nil
}

func eventInSyncMetrics(req *protocol.MsEvent) {
	log.Info("run eventInSyncMetrics")
	metrics := gHub.metrics()
	log.Info("run eventInSyncMetrics, metrics: ", metrics)
	pBroker.Broadcast(cWsConnectorOutSyncMetrics, metrics)
}

func actionUserInfo(req *protocol.MsRequest) (interface{}, error) {
	log.Info("run actionUserInfo, req.Params = ", req.Params)
	jsonString, err := jsoniter.Marshal(req.Params)
	if err != nil {
		log.Warn("run eventInKick, parse req.Data to jsonString error: ", err)
		return nil, errors.New("parse error")
	}
	jsonObj := &userIDStruct{}
	err = jsoniter.Unmarshal(jsonString, jsonObj)
	if err != nil {
		log.Warn("run eventInKick, parse req.Data to jsonObj userIDStruct error: ", err)
		return nil, errors.New("parse error")
	}
	if len(jsonObj.UserID) > 0 {
		return gHub.getUserOnlineInfo(jsonObj.UserID)
	}
	return nil, errors.New("userID error: " + jsonObj.UserID)
}

func eventInSyncUsersInfo(req *protocol.MsEvent) {
	log.Info("run eventInSyncUsersInfo")
	gHub.syncUsersInfo()
}

func eventInKickClient(req *protocol.MsEvent) {
	log.Info("run eventInKickClient, req.Data = ", req.Data)
	jsonString, err := jsoniter.Marshal(req.Data)
	if err != nil {
		log.Warn("run eventInKickClient, parse req.Data to jsonString error: ", err)
		return
	}
	jsonObj := &cidStruct{}
	err = jsoniter.Unmarshal(jsonString, jsonObj)
	if err != nil {
		log.Warn("run eventInKickClient, parse req.Data to jsonObj cidStruct error: ", err)
		return
	}
	if len(jsonObj.Cid) > 0 {
		gHub.kickClient(jsonObj.Cid)
	}
}

func eventInKickUser(req *protocol.MsEvent) {
	log.Info("run eventInKickUser, req.Data = ", req.Data)
	jsonString, err := jsoniter.Marshal(req.Data)
	if err != nil {
		log.Warn("run eventInKickUser, parse req.Data to jsonString error: ", err)
		return
	}
	jsonObj := &userIDStruct{}
	err = jsoniter.Unmarshal(jsonString, jsonObj)
	if err != nil {
		log.Warn("run eventInKickUser, parse req.Data to jsonObj userIDStruct error: ", err)
		return
	}
	if len(jsonObj.UserID) > 0 {
		gHub.kickUser(jsonObj.UserID)
	}
}

func actionPush(req *protocol.MsRequest) (interface{}, error) {

	log.Info("run actionPush, req.Params = ", req.Params)
	jsonString, err := jsoniter.Marshal(req.Params)
	if err != nil {
		log.Warn("run actionPush, parse req.Data to jsonString error: ", err)
		return nil, err
	}
	jsonObj := &pushMsgStruct{}
	err = jsoniter.Unmarshal(jsonString, jsonObj)
	if err != nil {
		log.Warn("run eventInPush, parse req.Data to jsonObj error: ", err)
		return nil, err
	}
	log.Info("run eventInPush, jsonObj = ", jsonObj)

	// log.Info("type:", reflect.TypeOf(jsonObj.IDs))

	ids := make([]string, 0)
	switch jsonObj.IDs.(type) {
	case []string:
		ids = jsonObj.IDs.([]string)
	case []interface{}:
		_ids := jsonObj.IDs.([]interface{})
		for _, v := range _ids {
			if sv, ok := v.(string); ok {
				ids = append(ids, sv)
			}
		}
	case string:
		ids = strings.Split(jsonObj.IDs.(string), ",")
	default:
		log.Info("can't parse jsonObj.IDs")
		return nil, errors.New("can't parse ids")
	}
	gHub.sendMessage(ids, jsonObj.Data)

	return nil, nil
}

//if only userID, then push to all platfrom of userID
//mol repl:
//emit ws-connector.in.push --ids utest-0 --Data.mid aaaabbbb --Data.msg.a hello --Data.msg.b 123 --Data.msg.c true
func eventInPush(req *protocol.MsEvent) {
	log.Info("run eventInPush, req.Data = ", req.Data)
	jsonString, err := jsoniter.Marshal(req.Data)
	if err != nil {
		log.Warn("run eventInPush, parse req.Data to jsonString error: ", err)
		return
	}
	jsonObj := &pushMsgStruct{}
	err = jsoniter.Unmarshal(jsonString, jsonObj)
	if err != nil {
		log.Warn("run eventInPush, parse req.Data to jsonObj error: ", err)
		return
	}
	log.Info("run eventInPush, jsonObj = ", jsonObj)

	// log.Info("type:", reflect.TypeOf(jsonObj.IDs))
	ids := make([]string, 0)
	switch jsonObj.IDs.(type) {
	case []string:
		ids = jsonObj.IDs.([]string)
	case []interface{}:
		_ids := jsonObj.IDs.([]interface{})
		for _, v := range _ids {
			if sv, ok := v.(string); ok {
				ids = append(ids, sv)
			}
		}
	case string:
		ids = strings.Split(jsonObj.IDs.(string), ",")
	default:
		log.Info("can't parse jsonObj.IDs")
		return
	}
	gHub.sendMessage(ids, jsonObj.Data)
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

//TODO: change this salt in PRODUCTION
const sha256salt string = "d9cd5e3663eefbe5868e903cc68f895bd849b3bb374b2aa0cff80bb16cb4e63d"

func genSha1String(s string) string {
	h := sha1.New()
	h.Write([]byte(s))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func gettoken(w http.ResponseWriter, r *http.Request) {
	queryValues := r.URL.Query()

	log.Printf("gettoken queryValues = %v", queryValues)
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
	version := queryValues.Get("version")
	if len(version) < 1 {
		w.WriteHeader(401)
		w.Write([]byte("Need Query Values [version] in URL"))
		return
	}
	timestamp := strconv.Itoa(int(time.Now().UnixNano() / 1e6))

	encodeString := userID + platform + version + timestamp + sha256salt
	genToken := genSha1String(encodeString)
	w.WriteHeader(200)
	data, err := jsoniter.Marshal(map[string]interface{}{
		"timestamp": timestamp,
		"token":     genToken,
	})
	if err != nil {
		log.Error("gettoken: Marshal data err: ", err)
		w.WriteHeader(500)
		return
	}
	w.Write(data)
}

// var gClientID uint64

// serveWs handles websocket requests from the peer.
func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	defer atomic.AddInt64(&gCurrentAccepting, -1)
	atomic.AddInt64(&gCurrentAccepting, 1)

	// if atomic.LoadInt64(&gCurrentClients) >= gMaxClients {
	// 	log.Warnf("gCurrentClients[%d] >= gMaxClients[%d], reject new client", atomic.LoadInt64(&gCurrentClients), gMaxClients)
	// 	w.WriteHeader(503)
	// 	w.Write([]byte("Service Unavailable: MaxClient"))
	// 	return
	// }

	//max concurrenty accept new webscoket 500
	if atomic.LoadInt64(&gCurrentAccepting) > maxConcurrentAccept {
		log.Warn("Too Busy: gCurrentAccepting = ", gCurrentAccepting)
		w.WriteHeader(503)
		w.Write([]byte("Service Unavailable: Busy"))
		return
	}

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

	version := queryValues.Get("version")
	if len(version) < 1 {
		w.WriteHeader(401)
		w.Write([]byte("Need Query Values [version] in URL"))
		return
	}

	timestamp := queryValues.Get("timestamp")
	if len(timestamp) < 1 {
		w.WriteHeader(401)
		w.Write([]byte("Need Query Values [timestamp] in URL"))
		return
	}

	token := queryValues.Get("token")
	if len(token) < 1 {
		w.WriteHeader(401)
		w.Write([]byte("Need Query Values [token] in URL"))
		return
	}

	//only if ws-token.verify return invalid == true to reject client
	//request timeout or error , default let it pass

	verifyToken := &verifyTokenStruct{
		URL:       r.URL.String(),
		UserID:    userID,
		Platform:  platform,
		Version:   version,
		Timestamp: timestamp,
		Token:     token,
	}
	// res, err := pBroker.Call("pushConnector.verify", verifyToken, nil)
	res, err := pBroker.Call(cWsTokenActionVerify, verifyToken, nil)
	if err == nil {
		jsonByte, err := jsoniter.Marshal(res)
		if err == nil {
			jsonObj := &verifyTokenResultStruct{}
			err := jsoniter.Unmarshal(jsonByte, jsonObj)
			if err == nil {
				if jsonObj.Invalid {
					log.Warn("RPC ws-token.verify Invalid: ", verifyToken.String())
					w.WriteHeader(403)
					w.Write([]byte("token is not valid"))
					return
				}
			}
		}
	} else {
		// log.Warn("RPC ws-token.verify err: ", err)
		// log.Warn("RPC ws-token.verify err, default allow client connection: ", userID+"_"+platform)
	}

	// encodeString := userID + platform + version + timestamp + sha256salt
	// genToken := genSha1String(encodeString)
	// if token != genToken {
	// 	w.WriteHeader(403)
	// 	w.Write([]byte("token is not valid"))
	// 	return
	// }

	//TODO: just parse id and platform from token, validate token, return fail

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Warn("websocket Upgrade connection err: ", err)
		w.WriteHeader(503)
		// log.Println(err)
		return
	}

	// atomic.AddUint64(&gClientID, 1)
	// log.Warn("websocket connection times: ", atomic.LoadUint64(&gClientID))

	client := &Client{
		// ID:           strconv.Itoa(int(gClientID)),
		Cid:            fmt.Sprintf("%s_%s_%s", userID, platform, gNodeID),
		UserID:         userID,
		Platform:       platform,
		Version:        version,
		Timestamp:      timestamp,
		Token:          token,
		ConnectTime:    strconv.Itoa(int(time.Now().UnixNano() / 1e6)),
		DisconnectTime: "0",
		hub:            hub,
		conn:           conn,
		sendChan:       make(chan []byte, 10),
		sendPongChan:   make(chan int, 10),
	}
	gHub.register(client)

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writePump()
	go client.readPump()
}

func startWsService() {
	gHub = newHub()
	gHub.run()
	// http.HandleFunc("/", serveHome)
	// http.HandleFunc("/gettoken", func(w http.ResponseWriter, r *http.Request) {
	// 	gettoken(w, r)
	// })
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(gHub, w, r)
	})
	if gIsDebug > 0 {
		http.HandleFunc("/dumpmem", func(w http.ResponseWriter, r *http.Request) {
			fm, err := os.OpenFile("./mem.out"+time.Now().Format("-15-04-05"), os.O_RDWR|os.O_CREATE, 0644)
			if err != nil {
				log.Fatal(err)
			}
			pprof.WriteHeapProfile(fm)
			fm.Close()
			w.WriteHeader(200)
			w.Write([]byte("OK"))
		})
	}

	listenHost := ":" + strconv.Itoa(gPort)
	go func() {
		acceptSocket, err := net.Listen("tcp", listenHost)
		if err != nil {
			log.Fatal("TCP Listen: ", err)
			os.Exit(1)
		}
		acceptSocket = netutil.LimitListener(acceptSocket, gMaxClients)
		err = http.ServeTLS(acceptSocket, nil, "cert_go.pem", "key_go.pem")
		// err := http.ListenAndServeTLS(listenHost, "cert_go.pem", "key_go.pem", nil)
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
