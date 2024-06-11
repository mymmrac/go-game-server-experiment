package main

import (
	"sync"

	"github.com/charmbracelet/log"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"

	"game-server-test/pkg/types"
)

type Server struct {
	app *fiber.App

	lock    *sync.RWMutex
	clients map[types.ClientID]*Client
}

func NewServer() *Server {
	return &Server{
		app:     nil,
		lock:    &sync.RWMutex{},
		clients: make(map[types.ClientID]*Client),
	}
}

func (s *Server) Start() {
	s.app = fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	s.app.Get("/", websocket.New(s.HandleWS))

	go func() {
		if err := s.app.Listen(":4242"); err != nil {
			log.Fatalf("Listen: %s", err)
		}
	}()
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

	if s.app != nil {
		if err := s.app.Shutdown(); err != nil {
			log.Errorf("Shutdown server: %s", err)
		}
	}
}
