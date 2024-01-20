package client

import (
	"github.com/fauxriarty/fuckit/message"
	"github.com/gorilla/websocket"
)

type Client struct {
	Send chan *message.Message
	Conn *websocket.Conn
}
