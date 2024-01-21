package main

import (
	"log"
	"net"
	"net/http"

	"github.com/fauxriarty/fuckit/web"
)

func main() {
	server := web.NewServer()
	go server.Run()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		web.ServeWs(server, w, r)
	})

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

	log.Fatal(http.ListenAndServe(":8080", nil))
}
