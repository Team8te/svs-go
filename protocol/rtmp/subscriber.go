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
	ch     chan struct{}
	buff   []*ds.Frame
	cancel context.CancelFunc
}

func MakeSubscriber(id string, wr writer) *subscriber {
	sub := &subscriber{
		id:         id,
		firstVideo: true,
		ch:         make(chan struct{}, 1),
		buff:       make([]*ds.Frame, 0, 100),
		wr:         wr,
	}
	return sub
}

func (sub *subscriber) run(ctx context.Context) {
	ctx, sub.cancel = context.WithCancel(ctx)
	go sub.do(ctx)
}

func (sub *subscriber) do(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case _ = <-sub.ch:
			frames := sub.getBuff()
			for _, f := range frames {
				sub.sendFrame(f)
			}
			log.Debugf("Send %v frames", len(frames))
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
	sub.mx.Lock()
	sub.buff = append(sub.buff, f)
	sub.mx.Unlock()
	select {
	case sub.ch <- struct{}{}:
	default:
	}

	return nil
}

func (sub *subscriber) Close() {
	sub.cancel()

	sub.mx.Lock()
	defer sub.mx.Unlock()

	if sub.ch != nil {
		close(sub.ch)
		sub.ch = nil
	}
	if sub.wr != nil {
		sub.wr.Close()
		sub.wr = nil
	}
}

func (sub *subscriber) getBuff() []*ds.Frame {
	sub.mx.Lock()
	defer sub.mx.Unlock()
	res := sub.buff
	sub.buff = make([]*ds.Frame, 0, 100)
	return res
}
