package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/websocket"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

const (
	MongoAddr = "localhost"
)

var (
	exitHub        *Hub
	MongoSess, err = mgo.Dial(MongoAddr)
)

type Request struct {
	Meth string `json:"meth"`
	Host string `json:"host"`
	Cont string `json:"cont"`
	User string `json:"user"`
	Data string `json:"data"`
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func main() {
	if err != nil {
		log.Fatal("Error connecting with mongo!")
	}

	log.Info("Started the hub")

	exitHub = NewHub()
	go exitHub.Listen()

	http.HandleFunc("/websocket", wsHandler)
	http.HandleFunc("/", requestHandler)
	log.Info("Started the http server")
	http.ListenAndServe(":8080", nil)
}

func requestHandler(w http.ResponseWriter, r *http.Request) {
	//These headers are needed to route the request through the network
	proxyID := r.Header.Get("Proxy-Id")
	finalHost := r.Header.Get("Final-host")

	id, err := strconv.Atoi(proxyID)
	if err != nil {
		log.Error(err)
		http.Error(w, `Internal error`, http.StatusBadRequest)
		return
	}

	proxy, err := exitHub.Search(id)
	if err != nil {
		log.Errorf("Invalid proxy_id! %d", id)
		http.Error(w, `Internal Error`, http.StatusBadRequest)
		return
	}

	data := new(bytes.Buffer)
	data.ReadFrom(r.Body)
	dataFinal := base64.StdEncoding.EncodeToString(data.Bytes())
	req := Request{r.Method, finalHost, r.Header.Get("Content-Type"), r.Header.Get("User-Agent"), dataFinal}

	json, _ := json.Marshal(req)
	proxy.Send(json)

	resp := <-proxy.Response
	w.Write(resp.Data)
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error("Failed to upgrade:", err)
	}

	defer conn.Close()

	var newClient = NewClient(conn, exitHub)
	log.Infof("New client %d", newClient.ID)
	exitHub.Add(newClient)

	bsonMap := bson.M{"id": newClient.ID, "use": false, "dead": false}
	MongoSess.DB("OpenPogoMap").C("Proxy").Insert(bsonMap)
	newClient.Listen()
}
