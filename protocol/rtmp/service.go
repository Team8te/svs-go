package rtmp

import (
	"context"

	"github.com/Team8te/svs-go/ds"
	"github.com/Team8te/svs-go/pkg/av"
	"github.com/Team8te/svs-go/protocol/rtmp/core"
	log "github.com/sirupsen/logrus"
)

type streamer interface {
	CreateStreamAndBind(pub av.Publisher, id ds.RoomID) error
	StartStream(id ds.RoomID) error
	RemoveStream(id ds.RoomID) error
	AddSubscribers(id ds.RoomID, subs ...av.Subscriber) error
}

// Service ...
type Service struct {
	r  roomSerice
	st streamer
}

// NewService ...
func NewService(r roomSerice, st streamer) *Service {
	return &Service{
		r:  r,
		st: st,
	}
}

// CreateRtmpDestination ...
func (s *Service) CreateRtmpPublisher(ctx context.Context, conn *core.ConnServer) error {
	r, err := s.r.GetRoomByID(ctx, conn.GetName())
	if err != nil {
		return err
	}

	p := newPublisher(conn)
	log.Debugf("new publisher: %+v", p.Info())

	err = s.st.CreateStreamAndBind(p, r.ID)
	if err != nil {
		return err
	}
	s.r.UpdateRoomByName(ctx, r)
	go s.st.StartStream(r.ID)
	return nil
}

func (s *Service) CreateSubscriber(ctx context.Context, conn *core.ConnServer) error {
	r, err := s.r.GetRoomByName(ctx, conn.GetName())
	if err != nil {
		return err
	}
	sub := NewVirWriter(conn)
	log.Infof("new sub: %+v . Room id: %v, name: %v", sub.Info(), r.ID, r.Name)

	return s.st.AddSubscribers(r.ID, sub)
}
