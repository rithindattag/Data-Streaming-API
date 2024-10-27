package websocket

import (
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rithindattag/realtime-streaming-api/pkg/logger"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type Hub struct {
	clients    map[*Client]bool
	Broadcast  chan Message
	Register   chan *Client
	Unregister chan *Client
	streams    map[string][]*Client
	mu         sync.Mutex
	logger     *logger.Logger
}

type Client struct {
	Hub      *Hub
	StreamID string
	Conn     *websocket.Conn
	Send     chan []byte
}

type Message struct {
	StreamID string
	Data     []byte
}

func NewHub(logger *logger.Logger) *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		Broadcast:  make(chan Message),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		streams:    make(map[string][]*Client),
		logger:     logger,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			h.clients[client] = true
			h.streams[client.StreamID] = append(h.streams[client.StreamID], client)
			h.mu.Unlock()
			h.logger.Info("Client registered", "streamID", client.StreamID)
		case client := <-h.Unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
				h.removeClientFromStream(client)
				h.logger.Info("Client unregistered", "streamID", client.StreamID)
			}
			h.mu.Unlock()
		case message := <-h.Broadcast:
			// Assuming message is now a struct with StreamID and Data fields
			h.mu.Lock()
			for _, client := range h.streams[message.StreamID] {
				select {
				case client.Send <- message.Data:
				default:
					close(client.Send)
					delete(h.clients, client)
					h.removeClientFromStream(client)
					h.logger.Info("Client removed due to blocked channel", "streamID", client.StreamID)
				}
			}
			h.mu.Unlock()
			h.logger.Info("Broadcasting message", "streamID", message.StreamID)
		}
	}
}

func (h *Hub) removeClientFromStream(client *Client) {
	clients := h.streams[client.StreamID]
	for i, c := range clients {
		if c == client {
			h.streams[client.StreamID] = append(clients[:i], clients[i+1:]...)
			break
		}
	}
	if len(h.streams[client.StreamID]) == 0 {
		delete(h.streams, client.StreamID)
	}
}

func (h *Hub) CreateStream(streamID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.streams[streamID]; !ok {
		h.streams[streamID] = make([]*Client, 0)
	}
}

func (c *Client) ReadPump() {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()
	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error { c.Conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, _, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				// log.Printf("error: %v", err)
			}
			break
		}
		// Process the message if needed
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func Upgrade(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	return upgrader.Upgrade(w, r, nil)
}
