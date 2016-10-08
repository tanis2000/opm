package main

import (
	"sync/atomic"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/pogointel/opm/opm"
	"github.com/gorilla/websocket"
)

var maxID int64

const (
	// Time allowed to read the next pong message from the peer.
	pongWait = 15 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 7) / 10
)

//A struct to store client data
type Client struct {
	ID       int64
	conn     *websocket.Conn
	Response chan *Message
	Writer   chan []byte
	Hub      *Hub
}

//A struct to store messages
type Message struct {
	Data   []byte
	sender *Client
	time   int64
}

func NewClient(ws *websocket.Conn, h *Hub) *Client {
	atomic.AddInt64(&maxID, 1)
	return &Client{maxID, ws, make(chan *Message), make(chan []byte), h}
}

func (c *Client) write(mt int, payload []byte) error {
	return c.conn.WriteMessage(mt, payload)
}

func (c *Client) Send(data []byte) {
	c.Writer <- data
}

//Listen for messages to read
func (c *Client) readHandler() {
	defer c.handleDisconnect()

	for {
		_, mes, err := c.conn.ReadMessage()
		if err != nil {
			log.Infof("Listener: client disconnected %d", c.ID)
			break
		}
		c.Response <- &Message{mes, c, time.Now().Unix()}
	}
}

//Listen for new messages to write and handle pings
func (c *Client) Listen() {
	ticker := time.NewTicker(pingPeriod)

	go c.readHandler()
	defer c.handleDisconnect()

	for {
		select {
		case toWrite := <-c.Writer:
			err := c.conn.WriteMessage(1, toWrite)
			if err != nil {
				log.Info("Failed to write to ws")
				break
			}

		case <-ticker.C:
			if err := c.write(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

func (c *Client) handleDisconnect() {
	c.conn.Close()
	c.Hub.Remove(c.ID)

	database.UpdateProxy(opm.Proxy{ID: c.ID, Dead: true})

	c.Response <- &Message{[]byte("The client has disconnected"), c, time.Now().Unix()}
}
