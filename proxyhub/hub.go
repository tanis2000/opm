package main

import (
	"errors"
)

//A place to store the clients
type Hub struct {
	proxies     map[int64]*Client
	RegisterC   chan *Client
	UnRegisterC chan int64
}

func NewHub() *Hub {
	proxies := make(map[int64]*Client)
	register := make(chan *Client)
	unregister := make(chan int64)
	return &Hub{proxies, register, unregister}
}

func (h *Hub) Add(c *Client) {
	h.RegisterC <- c
}

func (h *Hub) Remove(proxyID int64) {
	h.UnRegisterC <- proxyID
}

func (h *Hub) Search(proxyID int64) (*Client, error) {
	val, ok := h.proxies[proxyID]
	if ok {
		return val, nil
	}
	return nil, errors.New("Proxy not found")
}

func (h *Hub) Listen() {
	for {
		select {

		case client := <-h.RegisterC:
			h.proxies[client.ID] = client

		case client := <-h.UnRegisterC:
			delete(h.proxies, client)

		}
	}
}
