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
		server.broadcast <- message
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
	broadcast  chan []byte
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// NewServer creates a new Server instance
func NewServer() *Server {
	return &Server{
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

// Run starts the server
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
		case message := <-server.broadcast:
			for client := range server.clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(server.clients, client)
				}
			}
		}
	}
}

// ServeWs handles new WebSocket requests
func ServeWs(server *Server, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	client := &Client{Conn: conn, Send: make(chan []byte, 256)}
	server.register <- client

	// Start a new goroutine for each client
	go client.ReadPump(server)
	go client.WritePump()
}
