package rtmp

import (
	"context"

	"github.com/Team8te/svs-go/ds"
	"github.com/yapingcat/gomedia/go-codec"
)

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

	return &publisher{
		roomID:  room.ID,
		pubChan: make(chan *ds.Frame, frameBufferCount),
		r:       r,
		s:       s,
	}, nil
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
		Codec: cid,
		Data:  frame,
		PTS:   pts,
		DTS:   dts,
	}
	p.pubChan <- f
}

func (p *publisher) ReadFrame() (*ds.Frame, error) {
	return <-p.pubChan, nil
}

func (p *publisher) Close() {
	close(p.pubChan)
	p.s.RemoveStream(p.roomID)
}
