package main

import (
	"bytes"
	"encoding/gob"
	"errors"
	"io"
	"net"

	"github.com/charmbracelet/log"

	"game-server-test/pkg/types"
)

func (s *Server) ReadPackets() {
	for {
		buf := make([]byte, 4096)
		n, addr, err := s.pc.ReadFrom(buf)
		if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, net.ErrClosed) {
			log.Errorf("Read UDP: %s", err)
			continue
		}
		if n == 0 {
			continue
		}

		var msg types.Msg
		if err = gob.NewDecoder(bytes.NewReader(buf[:n])).Decode(&msg); err != nil {
			log.Errorf("Decode: %s", err)
			continue
		}

		if client := s.Client(msg.FromID, addr); client != nil {
			go s.HandlePacket(client, msg)
		} else {
			log.Errorf("Client not found: %s", addr.String())
		}
	}
}

func (s *Server) HandlePacket(myClient *Client, msg types.Msg) {
	myAddr := myClient.PacketAddr.String()

	buf := bytes.NewBuffer(nil)
	if err := gob.NewEncoder(buf).Encode(msg); err != nil {
		log.Errorf("Encode: %s", err)
		return
	}

	s.lock.RLock()
	for _, client := range s.clients {
		if client.PacketAddr == nil || myAddr == client.PacketAddr.String() {
			continue
		}

		n, err := s.pc.WriteTo(buf.Bytes(), client.PacketAddr)
		if err != nil || n != buf.Len() {
			log.Errorf("Write: %s", err)
		}
	}
	s.lock.RUnlock()
}
