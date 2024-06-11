package main

import (
	"net"

	"game-server-test/pkg/types"
)

type Client struct {
	ID         types.ClientID
	Conn       net.Conn
	PacketAddr net.Addr
}
