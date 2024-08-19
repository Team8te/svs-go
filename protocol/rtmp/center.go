package rtmp

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/Team8te/svs-go/ds"
	"github.com/yapingcat/gomedia/go-codec"
	"github.com/yapingcat/gomedia/go-rtmp"
)

type MediaCenter struct {
	center map[string]*MediaProducer
	mtx    sync.Mutex
}

func MakeMediaCenter() *MediaCenter {
	return &MediaCenter{
		center: make(map[string]*MediaProducer),
	}
}

func (c *MediaCenter) Register(name string, p *MediaProducer) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	c.center[name] = p
}

func (c *MediaCenter) Remove(name string) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	delete(c.center, name)
}

func (c *MediaCenter) Find(name string) *MediaProducer {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	if p, found := c.center[name]; found {
		return p
	} else {
		return nil
	}
}

func (c *MediaCenter) Handle(ctx context.Context, conn *MediaSession) error {
	conn.handle.OnPlay(func(app, streamName string, start, duration float64, reset bool) rtmp.StatusCode {
		if source := c.Find(streamName); source == nil {
			return rtmp.NETSTREAM_PLAY_NOTFOUND
		}
		return rtmp.NETSTREAM_PLAY_START
	})

	conn.handle.OnPublish(func(app, streamName string) rtmp.StatusCode {
		return rtmp.NETSTREAM_PUBLISH_START
	})

	conn.handle.SetOutput(func(b []byte) error {
		_, err := conn.conn.Write(b)
		return err
	})

	conn.handle.OnStateChange(func(newState rtmp.RtmpState) {
		if newState == rtmp.STATE_RTMP_PLAY_START {
			fmt.Println("play start")
			name := conn.GetStreamName()
			source := c.Find(name)
			if source != nil {
				fmt.Println("ready to play")
				cons := &Consumer{
					Name: name,
					ID:   conn.id,
					conn: conn,
				}
				cons.isAlive.Store(true)
				source.addConsumer(cons)
				go c.HandleConsumer(ctx, cons)
			}
		} else if newState == rtmp.STATE_RTMP_PUBLISH_START {
			fmt.Println("publish start")
			conn.handle.OnFrame(func(cid codec.CodecID, pts, dts uint32, frame []byte) {
				f := &ds.Frame{
					Codec: cid,
					Data:  frame,
					PTS:   pts,
					DTS:   dts,
				}
				conn.C <- f
			})
			name := conn.GetStreamName()
			p := newMediaProducer(name, conn)
			go c.HandlerProducer(ctx, p)
			c.Register(name, p)
		}
	})

	return nil
}

func (c *MediaCenter) HandlerProducer(ctx context.Context, p *MediaProducer) {
	defer c.Remove(p.name)
	p.dispatch(ctx)
}

func (c *MediaCenter) HandleConsumer(ctx context.Context, cons *Consumer) {
	defer func() {
		cons.Close()
		p := c.Find(cons.Name)
		if p != nil {
			p.removeConsumer(cons.ID)
		}
	}()
	cons.run(ctx)
}

type Consumer struct {
	ID      string
	Name    string
	conn    *MediaSession
	isAlive atomic.Bool
}

func (c *Consumer) run(ctx context.Context) {
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

func (c *Consumer) Play(frame *ds.Frame) {
	c.conn.play(frame)
}

func (c *Consumer) Close() {
	c.isAlive.Store(false)
}

func (c *Consumer) IsAlive() bool {
	return c.isAlive.Load()
}
