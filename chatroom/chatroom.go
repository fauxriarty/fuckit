package chatroom

import (
	"log"
	"net/http"

	"github.com/fauxriarty/fuckit/client"
	"github.com/fauxriarty/fuckit/message"
	"github.com/gorilla/websocket"
)

type Chatroom struct {
	Clients    map[*client.Client]bool
	Broadcast  chan *message.Message
	Register   chan *client.Client
	Unregister chan *client.Client
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins
	},
}

func (c *Chatroom) Run() {
	for {
		select {
		case client := <-c.Register:
			c.Clients[client] = true
			log.Printf("Registered new client: %v\n", client)
		case client := <-c.Unregister:
			if _, ok := c.Clients[client]; ok {
				delete(c.Clients, client)
				close(client.Send)
				log.Printf("Unregistered client: %v\n", client)
			}
		case message := <-c.Broadcast:
			for client := range c.Clients {
				select {
				case client.Send <- message:
					log.Printf("Broadcasted message: %v to client: %v\n", message, client)
				default:
					log.Printf("Skipped client: %v. Buffer might be full.\n", client)
				}
			}
		}
	}
}

func (c *Chatroom) HandleConnections(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v\n", err)
		return
	}

	client := &client.Client{Conn: conn, Send: make(chan *message.Message, 256)} // Added buffer to the channel
	c.Register <- client

	log.Printf("Handling connections for client: %v\n", client)
	go c.HandleMessages(client)
}

func (c *Chatroom) HandleMessages(client *client.Client) {
	defer func() {
		c.Unregister <- client
		client.Conn.Close()
		log.Printf("Closed connection for client: %v\n", client)
	}()

	for {
		message := &message.Message{}
		err := client.Conn.ReadJSON(message)
		if err != nil {
			log.Printf("Error reading JSON from client: %v, error: %v\n", client, err)
			break
		}

		log.Printf("Received message: %v from client: %v\n", message, client)
		c.Broadcast <- message
	}
}
