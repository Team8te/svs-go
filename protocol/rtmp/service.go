package rtmp

import (
	"context"
	"sync"

	"github.com/Team8te/svs-go/protocol/rtmp/core"
	log "github.com/sirupsen/logrus"
)

// Service ...
type Service struct {
	r       roomSerice
	streams sync.Map
}

// NewService ...
func NewService(r roomSerice) *Service {
	return &Service{
		r:       r,
		streams: sync.Map{},
	}
}

// CreateRtmpDestination ...
func (s *Service) CreateRtmpPublisher(ctx context.Context, conn *core.ConnServer) (int64, error) {
	r, err := s.r.GetRoomByID(ctx, conn.GetName())
	if err != nil {
		return 0, err
	}

	p := newPublisher(conn)
	log.Debugf("new publisher: %+v", p.Info())

	stream := NewStreamWithReader(p)
	stream.info = p.Info()

	go s.runStream(r.ID.ToString(), stream)

	return 0, nil
}

func (s *Service) runStream(id string, st *Stream) {
	s.streams.Store(id, st)
	defer s.removeStream(id)
	st.TransStart()
}

func (s *Service) removeStream(id string) error {
	s.streams.Delete(id)
	return nil
}

func (s *Service) CreateSubscriber(ctx context.Context, conn *core.ConnServer) error {
	r, err := s.r.GetRoomByName(ctx, conn.GetName())
	if err != nil {
		return err
	}
	sub := NewSub(conn)
	log.Infof("new sub: %+v . Room id: %v, name: %v", sub.Info(), r.ID, r.Name)

	go s.runSub(sub, r.Name)

	return nil
}

func (s *Service) runSub(sub *VirWriter, roomName string) {
	s.streams.Store(roomName, sub)
	defer s.removeStream(roomName)
	go sub.Check()
	sub.SendPacket()
}
