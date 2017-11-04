package tcp

import (
	"net"
	"time"
	"io"
	"errors"
	"context"
	"github.com/augneb/util"
)

type Tcp struct {
	conn        net.Conn
	closed      bool
	packageEof  []byte
	packEofLen  int
}

func NewTcp(dsn string, timeout ...time.Duration) (*Tcp, error) {
	var conn net.Conn
	var err error

	if len(timeout) > 0 && timeout[0] > 0 {
		conn, err = net.DialTimeout("tcp", dsn, timeout[0])
	} else {
		conn, err = net.Dial("tcp", dsn)
	}

	if err != nil {
		return nil, err
	}

	return &Tcp{conn: conn}, nil
}

func (t *Tcp) SetPackageEof(eof []byte) *Tcp {
	t.packageEof = eof
	t.packEofLen = len(eof)

	return t
}

func (t *Tcp) Write(b []byte) (int, error) {
	if t.closed {
		return 0, errors.New("connection has be closed")
	}

	return t.conn.Write(b)
}

func (t *Tcp) Read(ctx context.Context, call util.ReadPackageCall) error {
	if t.closed {
		return errors.New("connection has be closed")
	}

	var e error
	if t.packEofLen == 0 {
		e = util.ReadPackage(ctx, t.conn, call)
	} else {
		e = util.ReadPackage(ctx, t.conn, call, t.packageEof)
	}

	if e == io.EOF {
		t.Close()
	}

	return e
}

func (t *Tcp) IsValid() bool {
	if t.closed {
		return false
	}

	t.conn.SetReadDeadline(time.Now())
	if n, err := t.conn.Read([]byte{}); err == io.EOF || n > 0 {
		t.Close()
		return false
	}

	var zero time.Time
	t.conn.SetReadDeadline(zero)

	return true
}

func (t *Tcp) Close() error {
	t.closed = true
	return t.conn.Close()
}

