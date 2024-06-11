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
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/colornames"

	"game-server-test/pkg/types"
	"game-server-test/tcp-udp/pkg/common"
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
	tcpConn net.Conn
	udpConn net.Conn
	myID    types.ClientID

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

	tcpConn, err := net.Dial("tcp", ":4242")
	if err != nil {
		log.Fatalf("Dial TCP: %s", err)
	}

	var myID types.ClientID
	if err = common.DecodeAndRead(tcpConn, &myID); err != nil {
		log.Fatalf("Read my ID: %s", err)
	}
	log.Infof("My ID: %d", myID)

	udpConn, err := net.Dial("udp", ":4242")
	if err != nil {
		log.Fatalf("Dial UDP: %s", err)
	}
	log.Infof("UDP: %s", udpConn.LocalAddr())

	game := &Game{
		tcpConn: tcpConn,
		udpConn: udpConn,
		myID:    myID,

		pos: types.Position{},

		lock:    &sync.RWMutex{},
		players: make(map[types.ClientID]types.Position),
	}

	go game.ReadMessages()

	return game
}

func (g *Game) Shutdown() {
	if err := g.udpConn.Close(); err != nil {
		log.Errorf("Close UDP connection: %s", err)
	}

	if err := g.tcpConn.Close(); err != nil {
		log.Errorf("Close TCP connection: %s", err)
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

	err := common.EncodeAndWrite(g.udpConn, types.Msg{
		FromID: g.myID,
		Type:   types.MsgTypePosition,
		Data:   buf.Bytes(),
	})
	if err != nil {
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

func (g *Game) ReadMessages() {
	for {
		var msg types.Msg
		if err := common.DecodeAndRead(g.udpConn, &msg); err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
				return
			}

			log.Errorf("Read UDP: %s", err)
			continue
		}

		switch msg.Type {
		case types.MsgTypePosition:
			var pos types.Position
			if err := gob.NewDecoder(bytes.NewReader(msg.Data)).Decode(&pos); err != nil {
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
