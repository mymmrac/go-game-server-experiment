package main

import (
	"github.com/gofiber/contrib/websocket"

	"game-server-test/pkg/types"
)

type Client struct {
	ID   types.ClientID
	Conn *websocket.Conn
	Data chan []byte
}
