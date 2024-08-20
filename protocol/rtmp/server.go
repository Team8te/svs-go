package rtmp

import (
	"context"
	"net"

	log "github.com/sirupsen/logrus"
)

type mediaCenter interface {
	Handle(ctx context.Context, conn *MediaSession) error
}

type Server struct {
	l      net.Listener
	center mediaCenter
}

func NewServer(listener net.Listener, center mediaCenter) *Server {
	return &Server{
		l:      listener,
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
