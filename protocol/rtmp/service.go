package rtmp

import (
	"github.com/Team8te/svs-go/ds"
	"github.com/Team8te/svs-go/pkg/av"
)

type streamer interface {
	CreateStreamAndBind(id ds.RoomID, pub av.Publisher) error
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
