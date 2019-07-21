package util

import (
  "errors"
  "net"
  "os"
  "time"
)

// Listener wraps a net.Listener, and gives a place to store the timeout
// parameters. On Accept, it will wrap the net.Conn with our own Conn for us.
// Original implementation taken from https://gist.github.com/jbardin/9663312
// Thanks! <3
type Listener struct {
  net.Listener
  ReadTimeout  time.Duration
  WriteTimeout time.Duration
}

func NewListener(addr string, readTimeout, writeTimeout time.Duration) (net.Listener, error) {
  l, err := net.Listen("tcp", addr)
  if err != nil {
    return nil, err
  }

  tl := &Listener{
    Listener:     l,
    ReadTimeout:  readTimeout,
    WriteTimeout: writeTimeout,
  }
  return tl, nil
}

// Binds to a UNIX socket. If the file already exists, try to remove it before
// binding again. This logic is borrowed from Gunicorn
// (see https://github.com/benoitc/gunicorn/blob/a8963ef1a5a76f3df75ce477b55fe0297e3b617d/gunicorn/sock.py#L106)
func NewUnixListener(path string, readTimeout, writeTimeout time.Duration) (net.Listener, error) {
  stat, err := os.Stat(path)

  if err != nil {
    if !os.IsNotExist(err) {
      return nil, err
    }
  } else {
    if stat.Mode()&os.ModeSocket != 0 {
      err = os.Remove(path)

      if err != nil {
        return nil, err
      }
    } else {
      return nil, errors.New("specified path is not a socket")
    }
  }

  l, err := net.Listen("unix", path)

  if err != nil {
    return nil, err
  }

  tl := &Listener{
    Listener:     l,
    ReadTimeout:  readTimeout,
    WriteTimeout: writeTimeout,
  }

  return tl, nil
}

func (l *Listener) Accept() (net.Conn, error) {
  c, err := l.Listener.Accept()
  if err != nil {
    return nil, err
  }

  //go MetricsOpenConnections.Inc()

  tc := &Conn{
    Conn:         c,
    ReadTimeout:  l.ReadTimeout,
    WriteTimeout: l.WriteTimeout,
  }
  return tc, nil
}

