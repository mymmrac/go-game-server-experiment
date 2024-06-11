package main

import (
	"net"
	"sync"

	"github.com/charmbracelet/log"

	"game-server-test/pkg/types"
)

type Server struct {
	ln net.Listener
	pc net.PacketConn

	lock    *sync.RWMutex
	clients map[types.ClientID]*Client
}

func NewServer() *Server {
	return &Server{
		ln:      nil,
		pc:      nil,
		lock:    &sync.RWMutex{},
		clients: make(map[types.ClientID]*Client),
	}
}

func (s *Server) Start() {
	var err error

	s.ln, err = net.Listen("tcp", ":4242")
	if err != nil {
		log.Fatalf("Start TCP server: %s", err)
	}

	s.pc, err = net.ListenPacket("udp", ":4242")
	if err != nil {
		log.Fatalf("Start UDP server: %s", err)
	}
}

func (s *Server) Stop() {
	s.lock.Lock() // No unlocking to block new connections

	for _, client := range s.clients {
		if client.Conn == nil {
			continue
		}

		if err := client.Conn.Close(); err != nil {
			log.Errorf("Close client conn: %s", err)
		}
	}

	if s.pc != nil {
		if err := s.pc.Close(); err != nil {
			log.Errorf("Close packet conn: %s", err)
		}
	}

	if s.ln != nil {
		if err := s.ln.Close(); err != nil {
			log.Errorf("Close listener: %s", err)
		}
	}
}

func (s *Server) Client(id types.ClientID, addr net.Addr) *Client {
	s.lock.RLock()
	client, ok := s.clients[id]
	if !ok {
		s.lock.RUnlock()
		return nil
	}
	s.lock.RUnlock()

	// TODO: Client lock?
	if client.PacketAddr == nil {
		client.PacketAddr = addr
	}
	return client
}
