package rtmp

import (
	"context"

	"github.com/Team8te/svs-go/pkg/av"
	"github.com/Team8te/svs-go/protocol/rtmp/core"
	log "github.com/sirupsen/logrus"
)

type streamer interface {
	CreateStream(pub av.Publisher) (int64, error)
	StartStream(id int64) error
	RemoveStream(id int64) error
	AddSubscribers(id int64, subs ...av.Subscriber) error
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
func (s *Service) CreateRtmpPublisher(ctx context.Context, conn *core.ConnServer) (int64, error) {
	r, err := s.r.GetRoomByID(ctx, conn.GetName())
	if err != nil {
		return 0, err
	}

	p := newPublisher(conn)
	log.Debugf("new publisher: %+v", p.Info())

	r.StreamID, err = s.st.CreateStream(p)
	if err != nil {
		return 0, err
	}
	s.r.UpdateRoomByName(ctx, r)
	go s.st.StartStream(r.StreamID)
	return r.StreamID, nil
}

func (s *Service) CreateSubscriber(ctx context.Context, conn *core.ConnServer) error {
	r, err := s.r.GetRoomByName(ctx, conn.GetName())
	if err != nil {
		return err
	}
	sub := NewVirWriter(conn)
	log.Infof("new sub: %+v . Room id: %v, name: %v, stream_id: %v", sub.Info(), r.ID, r.Name, r.StreamID)

	return s.st.AddSubscribers(r.StreamID, sub)
}
