package types

type ClientID uint64

type Msg struct {
	FromID ClientID
	Type   MsgType
	Data   []byte
}

type MsgType uint

const (
	_ MsgType = iota
	MsgTypePosition
)

type Position struct {
	X int
	Y int
}
