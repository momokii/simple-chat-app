package ws

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var (
	webSocketUpgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
)

type Manager struct {
	clients ClientList
	sync.RWMutex

	handlers map[string]EventHandler
}

func NewManager() *Manager {
	m := &Manager{
		clients:  make(ClientList),
		handlers: make(map[string]EventHandler),
	}

	m.setupEventHandler()
	return m
}

func (m *Manager) setupEventHandler() {
	// for minimalizing switch case in router event, we can use map to store event type and handler

	// every event type will have its own handler
	m.handlers[EventSendMessage] = SendMessage
	m.handlers[EventChatRoom] = ChatRoomHandler
}

func (m *Manager) RouterEvent(event Event, c *Client) error {
	// check if event type is registered
	if handler, ok := m.handlers[event.Type]; ok {
		if err := handler(event, c); err != nil {
			return err
		}

		return nil

	} else {
		return errors.New("event type not found")
	}
}

func (m *Manager) ServeWS(w http.ResponseWriter, r *http.Request) {
	log.Println("new connection")

	conn, err := webSocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("error serve websocket: ", err)
		return
	}

	// every new connection will create new client and manager will manage it
	client := NewClient(conn, m)

	m.AddClient(client)

	// start client processes
	go client.ReadMessage()
	go client.WriteMessage()

}

func (m *Manager) AddClient(client *Client) {
	m.Lock()
	defer m.Unlock()

	m.clients[client] = true
}

func (m *Manager) RemoveClient(client *Client) {
	m.Lock()
	defer m.Unlock()

	if _, ok := m.clients[client]; ok {
		client.connection.Close()
		delete(m.clients, client)
	}

}

func SendMessage(event Event, c *Client) error {
	var chatevent SendMessageEvent
	var broadMessage NewMessageEvent

	if err := json.Unmarshal(event.Payload, &chatevent); err != nil {
		return fmt.Errorf("error unmarshal payload: %v", err)
	}

	broadMessage.Message = chatevent.Message
	broadMessage.From = chatevent.From
	broadMessage.Sent = time.Now()

	data, err := json.Marshal(broadMessage)
	if err != nil {
		return fmt.Errorf("error marshal payload: %v", err)
	}

	outgoingEvent := Event{
		Payload: data,
		Type:    EventNewMessage,
	}

	for client := range c.manager.clients {
		// only send message to client in same chatroom
		if client.chatroom == c.chatroom {
			client.egress <- outgoingEvent
		}
	}

	return nil
}

func ChatRoomHandler(event Event, c *Client) error {
	var chatevent ChangeRoomEvent

	if err := json.Unmarshal(event.Payload, &chatevent); err != nil {
		return fmt.Errorf("error unmarshal payload: %v", err)
	}

	c.chatroom = chatevent.Name

	return nil
}

func checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")

	switch origin {
	case "http://localhost:3000":
		return true
	default:
		return false
	}
}
