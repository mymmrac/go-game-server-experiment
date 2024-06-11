package main

import (
	"errors"
	"io"
	"math/rand/v2"
	"net"

	"github.com/charmbracelet/log"

	"game-server-test/pkg/types"
	"game-server-test/tcp-udp/pkg/common"
)

func (s *Server) AcceptConnections() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				break
			}

			log.Errorf("Accept conn: %s", err)
			continue
		}

		if client := s.AddConnection(conn); client != nil {
			go s.HandleConn(client)
		}
	}
}

func (s *Server) AddConnection(conn net.Conn) *Client {
	s.lock.Lock()
	client := &Client{
		ID:         types.ClientID(rand.Uint64()),
		Conn:       conn,
		PacketAddr: nil,
	}
	s.clients[client.ID] = client
	s.lock.Unlock()

	log.Infof("New connection from: %s, ID: %d", conn.RemoteAddr().String(), client.ID)
	return client
}

func (s *Server) RemoveConnection(client *Client) {
	addr := client.Conn.RemoteAddr().String()

	s.lock.Lock()
	if _, ok := s.clients[client.ID]; !ok {
		s.lock.Unlock()
		return
	}
	delete(s.clients, client.ID)
	s.lock.Unlock()

	if err := client.Conn.Close(); err != nil {
		log.Errorf("Close connection: %s", err)
	}

	log.Infof("Connection closed: %s, ID: %d", addr, client.ID)
}

func (s *Server) HandleConn(client *Client) {
	defer func() { s.RemoveConnection(client) }()
	conn := client.Conn

	if err := common.EncodeAndWrite(conn, client.ID); err != nil {
		log.Errorf("Write client ID: %s", err)
		return
	}

	for {
		buf := make([]byte, 4096)
		n, err := conn.Read(buf)
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
				return
			}

			log.Errorf("Read TCP: %s", err)
			continue
		}

		log.Infof("Read TCP: %s", buf[:n])
	}
}
