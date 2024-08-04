package rtmp

import (
	"context"

	"github.com/Team8te/svs-go/ds"
	"github.com/Team8te/svs-go/pkg/av"
)

type streamer interface {
	CreateStreamAndBind(id ds.RoomID, pub av.Publisher) error
	StartStream(id ds.RoomID) error
	RemoveStream(id ds.RoomID) error
	BindSubscribers(id ds.RoomID, subs ...av.Subscriber) error
}

type roomSerice interface {
	GetRoomByName(ctx context.Context, name string) (*ds.Room, error)
	GetRoomByID(ctx context.Context, id string) (*ds.Room, error)
	UpdateRoomByName(_ context.Context, room *ds.Room) error
}
