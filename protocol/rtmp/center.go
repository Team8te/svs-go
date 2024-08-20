package rtmp

import (
	"context"
	"fmt"

	"github.com/Team8te/svs-go/ds"
	cent "github.com/Team8te/svs-go/protocol/center"
	"github.com/yapingcat/gomedia/go-codec"
	"github.com/yapingcat/gomedia/go-rtmp"
)

type center interface {
	Register(name string, p cent.MediaProducer)
	Find(name string) cent.MediaProducer
	AddConsumer(ctx context.Context, producerName string, consumer cent.Consumer) error
	Remove(name string)
}

type MediaCenter struct {
	center center
}

func MakeMediaCenter(c center) *MediaCenter {
	return &MediaCenter{
		center: c,
	}
}

func (c *MediaCenter) Handle(ctx context.Context, conn *MediaSession) error {
	conn.handle.OnPlay(func(app, streamName string, start, duration float64, reset bool) rtmp.StatusCode {
		if source := c.center.Find(streamName); source == nil {
			return rtmp.NETSTREAM_PLAY_NOTFOUND
		}
		return rtmp.NETSTREAM_PLAY_START
	})

	conn.handle.OnPublish(func(app, streamName string) rtmp.StatusCode {
		return rtmp.NETSTREAM_PUBLISH_START

		return rtmp.NETSTREAM_CONNECT_REJECTED
	})

	conn.handle.SetOutput(func(b []byte) error {
		_, err := conn.conn.Write(b)
		return err
	})

	conn.handle.OnStateChange(func(newState rtmp.RtmpState) {
		if newState == rtmp.STATE_RTMP_PLAY_START {
			fmt.Println("play start")
			name := conn.GetStreamName()
			cons := MakeConsumer(name, conn)
			err := c.center.AddConsumer(ctx, name, cons)
			if err == nil {
				fmt.Println("ready to play")
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
			c.center.Register(name, p)
		}
	})

	return nil
}

func (c *MediaCenter) HandlerProducer(ctx context.Context, p *MediaProducer) {
	defer c.center.Remove(p.name)
	p.Dispatch(ctx)
}

func (c *MediaCenter) HandleConsumer(ctx context.Context, cons *Consumer) {
	defer func() {
		cons.Close()
		p := c.center.Find(cons.Name())
		if p != nil {
			p.RemoveConsumer(cons.ID())
		}
	}()
	cons.Run(ctx)
}
