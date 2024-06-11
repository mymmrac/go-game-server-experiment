package main

import (
	"bytes"
	"context"
	"encoding/gob"
	"math/rand/v2"

	"github.com/charmbracelet/log"
	"github.com/gofiber/contrib/websocket"

	"game-server-test/pkg/types"
)

func (s *Server) HandleWS(conn *websocket.Conn) {
	client := s.AddConnection(conn)
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		s.RemoveConnection(client)
	}()

	buf := bytes.NewBuffer(nil)
	if err := gob.NewEncoder(buf).Encode(client.ID); err != nil {
		log.Errorf("Encode ID: %s", err)
		return
	}

	if err := conn.WriteMessage(websocket.BinaryMessage, buf.Bytes()); err != nil {
		log.Errorf("Write ID: %s", err)
		return
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case data := <-client.Data:
				if err := conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
					log.Errorf("Write: %s", err)
					return
				}
			}
		}
	}()

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err) {
				return
			}

			log.Errorf("Read WS: %s", err)
			continue
		}

		var msg types.Msg
		if err = gob.NewDecoder(bytes.NewReader(data)).Decode(&msg); err != nil {
			log.Errorf("Decode: %s", err)
			continue
		}

		go s.HandleMessage(client, msg)
	}
}

func (s *Server) AddConnection(conn *websocket.Conn) *Client {
	s.lock.Lock()
	client := &Client{
		ID:   types.ClientID(rand.Uint64()),
		Conn: conn,
		Data: make(chan []byte),
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

func (s *Server) HandleMessage(myClient *Client, msg types.Msg) {
	buf := bytes.NewBuffer(nil)
	if err := gob.NewEncoder(buf).Encode(msg); err != nil {
		log.Errorf("Encode: %s", err)
		return
	}

	s.lock.RLock()
	for id, client := range s.clients {
		if id == myClient.ID {
			continue
		}

		client.Data <- buf.Bytes()
	}
	s.lock.RUnlock()
}
