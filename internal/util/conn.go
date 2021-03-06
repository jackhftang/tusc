package util

import (
	"net"
	"time"
)

// Conn wraps a net.Conn, and sets a deadline for every read
// and write operation.
type Conn struct {
	net.Conn
	ReadTimeout  time.Duration
	WriteTimeout time.Duration

	// closeRecorded will be true if the connection has been closed and the
	// corresponding prometheus counter has been decremented. It will be used to
	// avoid duplicated modifications to this metric.
	closeRecorded bool
}

func (c *Conn) Read(b []byte) (int, error) {
	var err error
	if c.ReadTimeout > 0 {
		err = c.Conn.SetReadDeadline(time.Now().Add(c.ReadTimeout))
	} else {
		err = c.Conn.SetReadDeadline(time.Time{})
	}

	if err != nil {
		return 0, err
	}

	return c.Conn.Read(b)
}

func (c *Conn) Write(b []byte) (int, error) {
	var err error
	if c.WriteTimeout > 0 {
		err = c.Conn.SetWriteDeadline(time.Now().Add(c.WriteTimeout))
	} else {
		err = c.Conn.SetWriteDeadline(time.Time{})
	}

	if err != nil {
		return 0, err
	}

	return c.Conn.Write(b)
}

func (c *Conn) Close() error {
	// Only decremented the prometheus counter if the Close function has not been
	// invoked before to avoid duplicated modifications.
	if !c.closeRecorded {
		c.closeRecorded = true
		//MetricsOpenConnections.Dec()
	}

	return c.Conn.Close()
}
