package rtmp

import (
	"context"
	"net"

	log "github.com/sirupsen/logrus"
)

const (
	frameBufferCount = 100000
	maxBufferSize    = 4 * 1024 * 1024
)

type mediaCenter interface {
	Handle(ctx context.Context, conn *MediaSession) error
}

type Server struct {
	l      net.Listener
	r      roomSerice
	st     streamer
	center mediaCenter
}

func NewServer(listener net.Listener, r roomSerice, st streamer, center mediaCenter) *Server {
	return &Server{
		l:      listener,
		r:      r,
		st:     st,
		center: center,
	}
}

func (s *Server) Run(ctx context.Context) {
	log.Info("RMP listen On ", s.l.Addr().String())
	for {
		c, err := s.l.Accept()
		if err != nil {
			return
		}
		conn := newMediaSession(c)
		ctx, cancel := context.WithCancel(context.TODO())
		conn.cancel = cancel
		err = s.center.Handle(ctx, conn)
		if err != nil {
			log.Errorf("Failed to handle new connection: %v", err)
			continue
		}
		go conn.run(ctx)
	}
}
