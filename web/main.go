// package main

// import (
// 	"log"
// 	"net"
// 	"net/http"

// 	"github.com/fauxriarty/fuckit/chatroom"
// 	"github.com/fauxriarty/fuckit/client"
// 	"github.com/fauxriarty/fuckit/message"
// 	"github.com/gin-gonic/gin"
// )

// func Status(c *gin.Context) {
// 	c.JSON(http.StatusOK, gin.H{
// 		"message": "WOOOOHOOOOOO",
// 	})
// }

// func main() {

// 	// router := gin.Default()

// 	chatroom := &chatroom.Chatroom{
// 		Broadcast:  make(chan *message.Message),
// 		Register:   make(chan *client.Client),
// 		Unregister: make(chan *client.Client),
// 		Clients:    make(map[*client.Client]bool),
// 	}

// 	go chatroom.Run()

// 	// router.GET("/", Status)

// 	http.HandleFunc("/ws", chatroom.HandleConnections)

// 	addrs, err := net.InterfaceAddrs()
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	var ip string
// 	for _, address := range addrs {
// 		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
// 			if ipnet.IP.To4() != nil {
// 				ip = ipnet.IP.String()
// 				break
// 			}
// 		}
// 	}

// 	if ip == "" {
// 		log.Fatal("No IP found")
// 	}

// 	log.Println("http server started on " + ip + ":8080")
// 	err = http.ListenAndServe(":8080", nil)
// 	if err != nil {
// 		log.Fatal("ListenAndServe: ", err)
// 	}
// }

package main

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

// Client represents a connected client (user).
type Client struct {
	Conn *websocket.Conn
	Send chan []byte
}

// Chatroom represents a single chat room.
type Chatroom struct {
	Clients   map[*Client]bool
	Join      chan *Client
	Leave     chan *Client
	Broadcast chan []byte
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow connections from any origin
	},
}

// Run starts the chatroom's main loop.
func (c *Chatroom) Run() {
	for {
		select {
		case client := <-c.Join:
			c.Clients[client] = true
		case client := <-c.Leave:
			if _, ok := c.Clients[client]; ok {
				delete(c.Clients, client)
				close(client.Send)
			}
		case message := <-c.Broadcast:
			for client := range c.Clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(c.Clients, client)
				}
			}
		}
	}
}

// handleConnections handles incoming WebSocket connections.
func (c *Chatroom) HandleConnections(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	client := &Client{Conn: conn, Send: make(chan []byte, 256)}
	c.Join <- client

	go client.HandleMessages(c)
}

// HandleMessages reads messages from the WebSocket connection.
func (c *Client) HandleMessages(chatroom *Chatroom) {
	defer func() {
		chatroom.Leave <- c
		c.Conn.Close()
	}()

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			log.Println("Error on message reading:", err)
			break
		}
		chatroom.Broadcast <- message
	}
}

func main() {
	chatroom := &Chatroom{
		Broadcast: make(chan []byte),
		Join:      make(chan *Client),
		Leave:     make(chan *Client),
		Clients:   make(map[*Client]bool),
	}

	go chatroom.Run()

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		if err := chatroom.HandleConnections(w, r); err != nil {
			log.Printf("Error handling connection: %v\n", err)
		}
	})

	log.Println("Chat server started on :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
