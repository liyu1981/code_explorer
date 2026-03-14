package api

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/rs/zerolog/log"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for now
	},
}

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second
)

// Hub maintains the set of active clients and broadcasts messages to the clients.
type WsHub struct {
	// Registered clients.
	clients map[*WsClient]bool

	// Inbound messages from the sessions or other data.
	broadcast chan *WsMessage

	// Register requests from the clients.
	register chan *WsClient

	// Unregister requests from clients.
	unregister chan *WsClient

	// Subscriptions to topics.
	subscriptions map[string]map[*WsClient]bool

	mu sync.Mutex
}

func NewWsHub() *WsHub {
	return &WsHub{
		broadcast:     make(chan *WsMessage),
		register:      make(chan *WsClient),
		unregister:    make(chan *WsClient),
		clients:       make(map[*WsClient]bool),
		subscriptions: make(map[string]map[*WsClient]bool),
	}
}

// Client is a middleman between the websocket connection and the hub.
type WsClient struct {
	hub *WsHub

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan any

	// Topics this client is subscribed to.
	topics map[string]bool
}

type WsMessage struct {
	Topic   string `json:"topic"`
	Payload any    `json:"payload"`
}

func (h *WsHub) run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Info().Msg("Client registered")
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				for topic := range client.topics {
					if _, ok := h.subscriptions[topic]; ok {
						delete(h.subscriptions[topic], client)
					}
				}
				close(client.send)
			}
			h.mu.Unlock()
			log.Info().Msg("Client unregistered")
		case message := <-h.broadcast:
			h.mu.Lock()
			if clients, ok := h.subscriptions[message.Topic]; ok {
				for client := range clients {
					select {
					case client.send <- message:
					default:
						close(client.send)
						delete(h.clients, client)
					}
				}
			}
			h.mu.Unlock()
		}
	}
}

func (c *WsClient) writePump() {
	defer func() {
		c.conn.Close()
	}()
	for message := range c.send {
		c.conn.SetWriteDeadline(time.Now().Add(writeWait))
		if err := c.conn.WriteJSON(message); err != nil {
			log.Error().Err(err).Msg("Error writing JSON message")
			return
		}
	}
	// The hub closed the channel.
	c.conn.WriteMessage(websocket.CloseMessage, []byte{})
}

// HandleWS handles websocket requests from the peer.
func (h *ApiHandler) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error().Err(err).Msg("WebSocket upgrade error")
		return
	}
	log.Info().Msg("New WebSocket client connected")
	client := &WsClient{
		hub:    h.hub,
		conn:   conn,
		send:   make(chan any, 256),
		topics: make(map[string]bool),
	}
	client.hub.register <- client

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writePump()
	go client.readPump()
}

// handlePingMessage handles ping requests from the client and responds with pong.
func (c *WsClient) handlePingMessage() {
	pongMsg := map[string]string{"type": "pong"}
	c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	if err := c.conn.WriteJSON(pongMsg); err != nil {
		log.Error().Err(err).Msg("Error sending pong")
	}
}

func (c *WsClient) handleSubscribeMessage(msg map[string]any) {
	topic, ok := msg["topic"].(string)
	if !ok {
		log.Debug().Interface("msg", msg).Msg("Subscribe message without topic")
		return
	}
	c.hub.mu.Lock()
	if _, ok := c.hub.subscriptions[topic]; !ok {
		c.hub.subscriptions[topic] = make(map[*WsClient]bool)
	}
	c.hub.subscriptions[topic][c] = true
	c.topics[topic] = true
	c.hub.mu.Unlock()
	log.Info().Str("topic", topic).Msg("Client subscribed to topic")
}

func (c *WsClient) handleUnsubscribeMessage(msg map[string]any) {
	topic, ok := msg["topic"].(string)
	if !ok {
		log.Debug().Interface("msg", msg).Msg("Unsubscribe message without topic")
		return
	}
	c.hub.mu.Lock()
	if _, ok := c.hub.subscriptions[topic]; ok {
		delete(c.hub.subscriptions[topic], c)
	}
	delete(c.topics, topic)
	c.hub.mu.Unlock()
	log.Info().Str("topic", topic).Msg("Client unsubscribed from topic")
}

// handleTextMessage processes incoming text messages from the client.
func (c *WsClient) handleTextMessage(message []byte) {
	// Try to parse as JSON
	var msg map[string]any
	if err := json.Unmarshal(message, &msg); err != nil {
		log.Debug().Str("message", string(message)).Msg("Non-JSON text message")
		return
	}

	// Check if message has a "type" field
	msgType, ok := msg["type"].(string)
	if !ok {
		log.Debug().Interface("msg", msg).Msg("Message without type field")
		return
	}

	// Handle different message types
	switch msgType {
	case "ping":
		c.handlePingMessage()
	case "subscribe":
		c.handleSubscribeMessage(msg)
	case "unsubscribe":
		c.handleUnsubscribeMessage(msg)
	default:
		log.Debug().Str("msgType", msgType).Msg("Unknown message type")
	}
}

func (c *WsClient) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	for {
		messageType, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Debug().Err(err).Msg("Unexpected WebSocket close")
			}
			break
		}

		switch messageType {
		case websocket.TextMessage:
			c.handleTextMessage(message)
		}
	}
}

// API for other Go code

// Publish sends a message to all subscribed clients of a topic.
// It is non-blocking; if the broadcast channel is full, the message will be dropped.
func (h *ApiHandler) Publish(topic string, payload any) {
	msg := &WsMessage{
		Topic:   topic,
		Payload: payload,
	}
	select {
	case h.hub.broadcast <- msg:
	default:
		// Drop message if channel is full to avoid blocking the caller
		log.Warn().Str("topic", topic).Msg("Broadcast channel full, message dropped")
	}
}

// WsBroadcast sends a generic message to all connected WebSocket clients.
// It is non-blocking; if the broadcast channel is full, the message will be dropped.
func (h *ApiHandler) WsBroadcast(v any) {
	h.Publish("broadcast", v)
}
