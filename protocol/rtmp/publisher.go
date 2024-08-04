package rtmp

import (
	"context"

	"github.com/Team8te/svs-go/ds"
	"github.com/Team8te/svs-go/pkg/av"
	"github.com/yapingcat/gomedia/go-codec"
)

var id = int64(0)

type publisher struct {
	roomID  ds.RoomID
	pubChan chan *ds.Frame
	r       roomSerice
	s       streamer
}

func makePublisher(ctx context.Context, streamID string, r roomSerice, s streamer) (*publisher, error) {
	room, err := r.GetRoomByID(ctx, streamID)
	if err != nil {
		return nil, err
	}

	pub := &publisher{
		roomID:  room.ID,
		pubChan: make(chan *ds.Frame, frameBufferCount),
		r:       r,
		s:       s,
	}

	return pub, nil
}

func (p *publisher) start() error {
	err := p.s.CreateStreamAndBind(p.roomID, p)
	if err != nil {
		return err
	}
	return p.s.StartStream(p.roomID)
}

func (p *publisher) write(cid codec.CodecID, pts, dts uint32, frame []byte) {
	f := &ds.Frame{
		ID:    id,
		Codec: cid,
		Data:  frame,
		PTS:   pts,
		DTS:   dts,
	}
	id++
	p.pubChan <- f
}

func (p *publisher) ReadFrame() (*ds.Frame, error) {
	return <-p.pubChan, nil
}

func (p *publisher) Close() {
	close(p.pubChan)
	p.s.RemoveStream(p.roomID)
}

func (p *publisher) BindSubscribers(_ context.Context, subs ...av.Subscriber) error {
	return p.s.BindSubscribers(p.roomID, subs...)
}
