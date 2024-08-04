package rtmp

import (
	"context"
	"fmt"
	"sync"

	"github.com/Team8te/svs-go/ds"
	log "github.com/sirupsen/logrus"
	"github.com/yapingcat/gomedia/go-codec"
)

type subscriber struct {
	firstVideo bool
	conn       *rtmpConn

	mx     sync.RWMutex
	buff   chan *ds.Frame
	cancel context.CancelFunc
}

func MakeSubscriber(conn *rtmpConn) *subscriber {
	sub := &subscriber{
		firstVideo: true,
		buff:       make(chan *ds.Frame, frameBufferCount),
		conn:       conn,
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
	err := sub.conn.write(f)
	if err != nil {
		log.Error("failed to send frame to subscriber", err)
	}
}

func (sub *subscriber) Write(f *ds.Frame) error {
	sub.mx.RLock()
	defer sub.mx.RUnlock()
	if sub.buff == nil {
		return fmt.Errorf("sub closed")
	}
	sub.buff <- f
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
}
