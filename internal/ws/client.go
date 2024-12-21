package ws

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

// manage every client connection for websocket

var (
	pongWait = 10 * time.Second // how long we wait for pong from client

	pingInterval = (pongWait * 9) / 10 // how often we send ping to client to keep connection alive and the value is less than pongWait
)

// list client connection with map
type ClientList map[*Client]bool

type Client struct {
	connection *websocket.Conn
	manager    *Manager

	chatroom string

	// egress used to send message to client
	// egress will received as event from manager and write to connection
	egress chan Event
}

func NewClient(conn *websocket.Conn, m *Manager) *Client {
	return &Client{
		connection: conn,
		manager:    m,
		egress:     make(chan Event),
	}
}

// pongHandler will be called when client send pong message
func (c *Client) pongHandler(pongMsg string) error {
	return c.connection.SetReadDeadline(time.Now().Add(pongWait))
}

// ! notes
// * - ws connection from gorilla basic just one concurrent connection at a time
// * we can use unbuffered channel/ go routine to handle multiple connection
func (c *Client) ReadMessage() {
	// setup defer function for close connection when function end for every reason
	defer func() {
		// cleanup connection
		c.manager.RemoveClient(c)
	}()

	// set read deadline for connection
	// if client not send pong in pongWait time, the connection will be closed
	if err := c.connection.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		log.Println("error set read deadline: ", err)
		return
	}

	// set read limit for connection to avoid large message
	// if message size more than 512 bytes, the connection will be closed
	c.connection.SetReadLimit(512)

	// set pong handler for connection
	// pong handler will be called on ReadMessage below on for loop, there is just to set the handler
	c.connection.SetPongHandler(c.pongHandler)

	for {
		// read message from connection
		// so will not get the message type because the payload itself is a Event with type and payload, so we need just payload
		// because for now, the message we receive will have structure like this:
		// { "type": string, "payload": any }
		_, payload, err := c.connection.ReadMessage()
		if err != nil {
			// check if error is not normal close error so log it
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Println("error read message: ", err)
			}

			// normal error is like client close connection
			break
		}

		var request Event
		// unmarshal payload to Event struct
		if err := json.Unmarshal(payload, &request); err != nil {
			log.Println("error unmarshal payload: ", err)
			break
		}

		// router event to handler
		if err := c.manager.RouterEvent(request, c); err != nil {
			log.Println("error router event: ", err)
		}

		// for wsClient := range c.manager.clients {
		// 	// send message to all client
		// 	wsClient.egress <- payload
		// }

		// log.Println("message type: ", request.Type)
		// log.Println("message: ", string(payload))
	}
}

func (c *Client) WriteMessage() {
	// setup defer for ensure
	defer func() {
		c.manager.RemoveClient(c)
	}()

	// define the ticker for ping
	// ping used for keep connection alive and avoid connection closed by server
	ticker := time.NewTicker(pingInterval)

	for {
		select {
		// read message from channel
		// message itself is Event struct
		case message, ok := <-c.egress:
			if !ok {
				// if error when write message to connection
				// still trying send the error message to client
				if err := c.connection.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
					// still error just log it
					log.Println("error write close message: ", err)
				}
				// return will break the loop
				return
			}

			// so, we need to marshal the message to json because the message is Event struct
			data, err := json.Marshal(message)
			if err != nil {
				log.Println("error marshal message: ", err)
				return
			}

			// send message to client
			if err := c.connection.WriteMessage(websocket.TextMessage, data); err != nil {
				log.Println("error write message: ", err)
			}
			// no return bcs we want to keep the loop until channel closed

		// ticker above will send ping/signal to ticker channel every pingInterval
		// so case below will be executed/triggered every pingInterval
		case <-ticker.C:
			// send ping to client
			// we can define the message type (for ping) and payload
			// with we send the message PING to client and just to make the connection still alive and the ping message will be not doing anything
			if err := c.connection.WriteMessage(websocket.PingMessage, []byte("")); err != nil {
				log.Println("error write ping message: ", err)
				return // return will break the loop if error happen
			}
		}

	}
}
