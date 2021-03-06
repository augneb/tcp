package tcp

import (
	"net"
	"errors"
	"context"
	"github.com/augneb/util"
)

type Tcp struct {
	conn       net.Conn
	rch        chan []byte
	inUse      bool
	closed     bool
	packageEof []byte
	packEofLen int
}

var (
	ErrCancelled        = errors.New("user cancelled")
	ErrConnClosed       = errors.New("connection has be closed")
	ErrUnknownException = errors.New("unknown exception")
)

func NewTcp(dsn string, packEof ...interface{}) (*Tcp, error) {
	addr, _ := net.ResolveTCPAddr("tcp", dsn)
	conn, err := net.DialTCP("tcp", nil, addr)

	if err != nil {
		return nil, err
	}

	t := &Tcp{conn: conn, rch: make(chan []byte)}

	if len(packEof) > 0 {
		switch packEof[0].(type) {
		case string:
			t.packageEof = []byte(packEof[0].(string))
		case []byte:
			t.packageEof = packEof[0].([]byte)
		}

		t.packEofLen = len(t.packageEof)
	}

	go t.loopRead()

	return t, nil
}

func (t *Tcp) Write(b []byte) (int, error) {
	if t.closed {
		return 0, ErrConnClosed
	}

	if t.inUse || len(t.rch) > 0 {
		t.Close()
		return 0, ErrUnknownException
	}

	t.inUse = true

	return t.conn.Write(b)
}

func (t *Tcp) Read(ctx context.Context, call util.ReadPackageCall) error {
	if t.closed {
		return ErrConnClosed
	}

	next := false
	for {
		select {
		case pack := <-t.rch:
			if len(pack) == 0 && t.closed {
				return ErrConnClosed
			}

			next = call(pack)
		case <-ctx.Done():
			return ErrCancelled
		}

		if !next {
			break
		}
	}

	t.inUse = false

	return nil
}

func (t *Tcp) IsValid() bool {
	return !t.closed
}

func (t *Tcp) Close() error {
	t.closed = true
	return t.conn.Close()
}

func (t *Tcp) loopRead() {
	eof := []interface{}{}
	if t.packEofLen > 0 {
		eof = []interface{}{t.packageEof}
	}

	err := util.ReadPackage(nil, t.conn, func(pack []byte) bool {
		// 有脏数据
		if !t.inUse {
			t.Close()
			return false
		}

		t.rch <- pack

		if t.closed {
			return false
		}

		return true
	}, eof...)

	if err != nil {
		t.Close()
		t.rch <- []byte{}
	}
}
