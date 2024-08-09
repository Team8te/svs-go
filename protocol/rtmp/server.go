package rtmp

import (
	"context"
	"net"
)

const (
	frameBufferCount = 100000
	maxBufferSize    = 4 * 1024 * 1024
)

type Server struct {
	l      net.Listener
	r      roomSerice
	st     streamer
	center *MediaCenter
}

func NewServer(listener net.Listener, r roomSerice, st streamer) *Server {
	return &Server{
		l:      listener,
		r:      r,
		st:     st,
		center: MakeMediaCenter(),
	}
}

func (s *Server) Run(ctx context.Context) {
	for {
		c, err := s.l.Accept()
		if err != nil {
			return
		}
		conn := newMediaSession(c)
		ctx, cancel := context.WithCancel(context.TODO())
		conn.cancel = cancel
		s.center.Handle(ctx, conn)
		go conn.run(ctx)
	}
}
