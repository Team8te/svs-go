package rtmp

import (
	"context"
	"net"
)

const (
	frameBufferCount = 100000
	maxBufferSize    = 4 * 1024
)

type Server struct {
	l  net.Listener
	r  roomSerice
	st streamer
}

func NewServer(listener net.Listener, r roomSerice, st streamer) *Server {
	return &Server{
		l:  listener,
		r:  r,
		st: st,
	}
}

func (s *Server) Run(ctx context.Context) {
	for {
		c, err := s.l.Accept()
		if err != nil {
			return
		}
		conn := s.newConn(c)
		ctx, conn.cancel = context.WithCancel(context.TODO())
		conn.init(ctx)
		go conn.run(ctx)
	}
}
