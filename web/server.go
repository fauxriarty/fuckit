package web

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

type Client struct {
	Conn *websocket.Conn
	Send chan []byte
}

// readPump listens for new messages from the client
func (c *Client) ReadPump(server *Server) {
	defer func() {
		server.unregister <- c
		c.Conn.Close()
	}()
	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			log.Println("Error on message read:", err)
			break
		}
		server.broadcast <- Message{Data: message, Sender: c}
	}
}

// writePump sends messages to the client
func (c *Client) WritePump() {
	defer c.Conn.Close()
	for {
		select {
		case message, ok := <-c.Send:
			if !ok {
				if err := c.Conn.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
					log.Printf("Failed to write close message: %v", err)
				}
				return
			}
			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("Failed to write message: %v", err)
			}
		}
	}
}

type Server struct {
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	//broadcast  chan []byte
	broadcast chan Message
}

type Message struct {
	Data   []byte
	Sender *Client
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func NewServer() *Server {
	return &Server{
		broadcast:  make(chan Message),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

func (server *Server) Run() {
	for {
		select {
		case client := <-server.register:
			server.clients[client] = true
		case client := <-server.unregister:
			if _, ok := server.clients[client]; ok {
				delete(server.clients, client)
				close(client.Send)
			}
		case msg := <-server.broadcast:
			for client := range server.clients {
				if client != msg.Sender { // skip sending message to the sender
					select {
					case client.Send <- msg.Data:
					default:
						close(client.Send)
						delete(server.clients, client)
					}
				}
			}
		}
	}
}

func ServeWs(server *Server, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	client := &Client{Conn: conn, Send: make(chan []byte, 256)}
	server.register <- client

	// starts a new goroutine for each client
	go client.ReadPump(server)
	go client.WritePump()
}
