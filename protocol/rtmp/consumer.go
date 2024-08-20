package rtmp

import (
	"context"
	"sync/atomic"

	"github.com/Team8te/svs-go/ds"
	"github.com/yapingcat/gomedia/go-codec"
)

type Consumer struct {
	name    string
	conn    *MediaSession
	isAlive atomic.Bool
}

func MakeConsumer(name string, conn *MediaSession) *Consumer {
	cons := &Consumer{
		name: name,
		conn: conn,
	}
	cons.isAlive.Store(true)
	return cons
}

func (c *Consumer) Run(ctx context.Context) {
	firstVideo := true
	for {
		select {
		case <-c.conn.frameCome:
			c.conn.mtx.Lock()
			frames := c.conn.lists
			c.conn.lists = nil
			c.conn.mtx.Unlock()
			for _, frame := range frames {
				if firstVideo { //wait for I frame
					if frame.Codec == codec.CODECID_VIDEO_H264 && codec.IsH264IDRFrame(frame.Data) {
						firstVideo = false
					} else if frame.Codec == codec.CODECID_VIDEO_H265 && codec.IsH265IDRFrame(frame.Data) {
						firstVideo = false
					} else {
						continue
					}
				}
				err := c.conn.handle.WriteFrame(frame.Codec, frame.Data, frame.PTS, frame.DTS)
				if err != nil {
					c.conn.stop()
					return
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

func (c *Consumer) Name() string {
	return c.name
}

func (c *Consumer) ID() string {
	return c.conn.id
}

func (c *Consumer) Play(frame *ds.Frame) {
	c.conn.play(frame)
}

func (c *Consumer) Close() {
	c.isAlive.Store(false)
}

func (c *Consumer) IsAlive() bool {
	return c.isAlive.Load()
}
