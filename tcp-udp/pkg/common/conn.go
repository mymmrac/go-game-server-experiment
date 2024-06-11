package common

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
)

func EncodeAndWrite(writer io.Writer, value any) error {
	buf := bytes.NewBuffer(nil)
	if err := gob.NewEncoder(buf).Encode(value); err != nil {
		return fmt.Errorf("encode: %w", err)
	}

	n, err := writer.Write(buf.Bytes())
	if err != nil {
		return fmt.Errorf("write: %w", err)
	}

	if n != len(buf.Bytes()) {
		return fmt.Errorf("write: %w", io.ErrShortWrite)
	}

	return nil
}

func DecodeAndRead(reader io.Reader, value any) error {
	buf := make([]byte, 4096)
	n, err := reader.Read(buf)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}

	if err = gob.NewDecoder(bytes.NewReader(buf[:n])).Decode(value); err != nil {
		return fmt.Errorf("decode: %w", err)
	}

	return nil
}
