package main

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"image/color"
	"io"
	"net"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/fasthttp/websocket"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/colornames"

	"game-server-test/pkg/types"
)

func main() {
	log.Info("Starting client...")
	game := NewGame()
	if err := ebiten.RunGame(game); err != nil {
		log.Fatalf("Run game: %s", err)
	}
	game.Shutdown()
	log.Info("Bye!")
}

type Game struct {
	ws   *websocket.Conn
	myID types.ClientID

	pos types.Position

	lock    *sync.RWMutex
	players map[types.ClientID]types.Position

	prevUpdatesPerSecond uint
	updatesPerSecond     uint
	updateTime           time.Time
}

func NewGame() *Game {
	ebiten.SetWindowTitle("Client")
	ebiten.SetWindowSize(1080, 720)
	ebiten.SetWindowClosingHandled(true)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ww, wh := ebiten.WindowSize()
	mw, mh := ebiten.Monitor().Size()
	ebiten.SetWindowPosition((mw-ww)/2, (mh-wh)/2)

	ws, _, err := websocket.DefaultDialer.Dial("ws://localhost:4242", nil)
	if err != nil {
		log.Fatalf("Dial WS: %s", err)
	}

	_, data, err := ws.ReadMessage()
	if err != nil {
		log.Fatalf("Read my ID: %s", err)
	}

	var myID types.ClientID
	if err = gob.NewDecoder(bytes.NewReader(data)).Decode(&myID); err != nil {
		log.Fatalf("Read my ID: %s", err)
	}
	log.Infof("My ID: %d", myID)

	game := &Game{
		ws:   ws,
		myID: myID,

		pos: types.Position{},

		lock:    &sync.RWMutex{},
		players: make(map[types.ClientID]types.Position),
	}

	go game.ReadMessages()

	return game
}

func (g *Game) Shutdown() {
	if err := g.ws.Close(); err != nil {
		log.Errorf("Close WS connection: %s", err)
	}
}

func (g *Game) ReadMessages() {
	for {
		_, data, err := g.ws.ReadMessage()
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
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

		switch msg.Type {
		case types.MsgTypePosition:
			var pos types.Position
			if err = gob.NewDecoder(bytes.NewReader(msg.Data)).Decode(&pos); err != nil {
				log.Errorf("Decode: %s", err)
				continue
			}

			g.lock.Lock()
			g.updatesPerSecond++
			g.players[msg.FromID] = pos
			g.lock.Unlock()
		default:
			log.Errorf("Unknown message type: %d", msg.Type)
		}
	}
}

func (g *Game) Update() error {
	if ebiten.IsWindowBeingClosed() || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return ebiten.Termination
	}

	const speed = 10

	if ebiten.IsKeyPressed(ebiten.KeyW) {
		g.pos.Y -= speed
	} else if ebiten.IsKeyPressed(ebiten.KeyS) {
		g.pos.Y += speed
	}
	if ebiten.IsKeyPressed(ebiten.KeyA) {
		g.pos.X -= speed
	} else if ebiten.IsKeyPressed(ebiten.KeyD) {
		g.pos.X += speed
	}

	buf := bytes.NewBuffer(nil)
	if err := gob.NewEncoder(buf).Encode(g.pos); err != nil {
		return fmt.Errorf("encode pos: %w", err)
	}

	msgBuf := bytes.NewBuffer(nil)
	if err := gob.NewEncoder(msgBuf).Encode(types.Msg{
		FromID: g.myID,
		Type:   types.MsgTypePosition,
		Data:   buf.Bytes(),
	}); err != nil {
		return fmt.Errorf("encode: %w", err)
	}

	if err := g.ws.WriteMessage(websocket.BinaryMessage, msgBuf.Bytes()); err != nil {
		return fmt.Errorf("write pos: %w", err)
	}

	if time.Since(g.updateTime) > time.Second {
		g.updateTime = time.Now()
		g.lock.Lock()
		g.prevUpdatesPerSecond = g.updatesPerSecond
		g.updatesPerSecond = 0
		g.lock.Unlock()
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{
		R: 28,
		G: 28,
		B: 28,
		A: 255,
	})

	vector.DrawFilledCircle(screen, float32(g.pos.X), float32(g.pos.Y), 10, color.White, true)

	g.lock.RLock()
	for _, pos := range g.players {
		vector.DrawFilledCircle(screen, float32(pos.X), float32(pos.Y), 10, colornames.Lightgreen, true)
	}
	ebitenutil.DebugPrint(screen, fmt.Sprintf("FPS: %0.2f\nTPS: %0.2f\n%d updates/sec", ebiten.ActualFPS(), ebiten.ActualTPS(), g.prevUpdatesPerSecond))
	g.lock.RUnlock()
}

func (g *Game) Layout(_, _ int) (int, int) { panic("unreachable") }

func (g *Game) LayoutF(outsideWidth, outsideHeight float64) (screenWidth, screenHeight float64) {
	return outsideWidth, outsideHeight
}
