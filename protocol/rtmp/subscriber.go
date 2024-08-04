package rtmp

import (
	"context"
	"sync"

	"github.com/Team8te/svs-go/ds"
	log "github.com/sirupsen/logrus"
	"github.com/yapingcat/gomedia/go-codec"
)

type writer interface {
	Write(f *ds.Frame) error
	Close()
}

type subscriber struct {
	id         string
	firstVideo bool
	wr         writer

	mx     sync.RWMutex
	buff   chan *ds.Frame
	cancel context.CancelFunc
}

func MakeSubscriber(id string, wr writer) *subscriber {
	sub := &subscriber{
		id:         id,
		firstVideo: true,
		buff:       make(chan *ds.Frame, frameBufferCount),
		wr:         wr,
	}
	return sub
}

func (sub *subscriber) run(ctx context.Context) {
	ctx, sub.cancel = context.WithCancel(ctx)
	//go sub.do(ctx)
}

func (sub *subscriber) do(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case f := <-sub.buff:
			sub.sendFrame(f)
		}
	}
}

func (sub *subscriber) sendFrame(f *ds.Frame) {
	if sub.firstVideo { //wait for I frame
		if f.Codec == codec.CODECID_VIDEO_H264 && codec.IsH264IDRFrame(f.Data) {
			sub.firstVideo = false
		} else if f.Codec == codec.CODECID_VIDEO_H265 && codec.IsH265IDRFrame(f.Data) {
			sub.firstVideo = false
		} else {
			return
		}
	}
	log.Debugf("Sub: %v Frame id: %v", sub.id, f.ID)
	err := sub.wr.Write(f)
	if err != nil {
		log.Error("failed to send frame to subscriber", err)
	}
}

func (sub *subscriber) Write(f *ds.Frame) error {
	sub.sendFrame(f)
	//sub.buff <- f
	return nil
}

func (sub *subscriber) Close() {
	sub.cancel()

	sub.mx.Lock()
	defer sub.mx.Unlock()

	if sub.buff != nil {
		close(sub.buff)
		sub.buff = nil
	}
	if sub.wr != nil {
		sub.wr.Close()
		sub.wr = nil
	}
}
