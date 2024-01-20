package main

import (
	"log"
	"net"
	"net/http"

	"github.com/fauxriarty/fuckit/chatroom"
	"github.com/fauxriarty/fuckit/client"
	"github.com/fauxriarty/fuckit/message"
	"github.com/gin-gonic/gin"
)

func Status(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "WOOOOHOOOOOO",
	})
}

func main() {

	// router := gin.Default()

	chatroom := &chatroom.Chatroom{
		Broadcast:  make(chan *message.Message),
		Register:   make(chan *client.Client),
		Unregister: make(chan *client.Client),
		Clients:    make(map[*client.Client]bool),
	}

	go chatroom.Run()

	// router.GET("/", Status)

	http.HandleFunc("/ws", chatroom.HandleConnections)

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Fatal(err)
	}

	var ip string
	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ip = ipnet.IP.String()
				break
			}
		}
	}

	if ip == "" {
		log.Fatal("No IP found")
	}

	log.Println("http server started on " + ip + ":8080")
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
