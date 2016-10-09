package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/websocket"
	"github.com/pogointel/opm/db"
	"github.com/pogointel/opm/opm"
)

const (
	MongoAddr = "localhost"
)

var (
	exitHub  *Hub
	database *db.OpenMapDb
)

type Request struct {
	Meth string `json:"meth"`
	Host string `json:"host"`
	Cont string `json:"cont"`
	User string `json:"user"`
	Data string `json:"data"`
}

type Settings struct {
	DbUser     string
	DbPassword string
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func main() {
	opmSettings, err := opm.LoadSettings("")
	if err != nil {
		log.Printf("Error loading settings (%s). Using default settings.\n", err)
	}
	// Login DB
	database, err = db.NewOpenMapDb("OpenPogoMap", MongoAddr, opmSettings.DbUser, opmSettings.DbPassword)
	if err != nil {
		log.Fatal(err)
	}
	// Set max id
	maxID, err = database.MaxProxyId()
	if err != nil {
		log.Println(err)
	}
	log.Printf("Max id: %d", maxID)
	// Delete old stuff
	database.DropProxies()

	log.Info("Started the hub")

	exitHub = NewHub()
	go exitHub.Listen()

	// proxy client (ws) server
	wsMux := http.NewServeMux()
	wsMux.HandleFunc("/websocket", wsHandler)
	wsServer := http.Server{Addr: fmt.Sprintf(":%d", opmSettings.ProxyWSListenPort), Handler: wsMux}
	// proxy request server
	prMux := http.NewServeMux()
	prMux.HandleFunc("/", requestHandler)
	prServer := http.Server{
		Addr:         fmt.Sprintf(":%d", opmSettings.ProxyListenPort),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 20 * time.Second,
		Handler:      prMux,
	}
	// start servers
	log.Println("Starting WS server")
	go runServer(wsServer)
	log.Println("Starting requests server")
	log.Fatal(prServer.ListenAndServe())

}

func runServer(s http.Server) {
	err := s.ListenAndServe()
	log.Fatal(err)
}

func requestHandler(w http.ResponseWriter, r *http.Request) {
	//These headers are needed to route the request through the network
	proxyID := r.Header.Get("Proxy-Id")
	finalHost := r.Header.Get("Final-host")

	id, err := strconv.ParseInt(proxyID, 10, 64)
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

	j, _ := json.Marshal(req)
	proxy.Send(j)

	resp := <-proxy.Response
	w.Write(resp.Data)
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error("Failed to upgrade:", err)
		return
	}
	defer conn.Close()

	var newClient = NewClient(conn, exitHub)
	log.Infof("New client %d", newClient.ID)
	exitHub.Add(newClient)

	p := opm.Proxy{ID: newClient.ID, Dead: false, Use: false}
	database.AddProxy(p)
	newClient.Listen()
}
