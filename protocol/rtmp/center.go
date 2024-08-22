package rtmp

import (
	"context"
	"fmt"

	"github.com/Team8te/svs-go/ds"
	cent "github.com/Team8te/svs-go/protocol/center"
	log "github.com/sirupsen/logrus"
	"github.com/yapingcat/gomedia/go-codec"
	"github.com/yapingcat/gomedia/go-rtmp"
)

type center interface {
	Register(name string, p cent.MediaProducer) error
	Find(name string) cent.MediaProducer
	AddConsumer(ctx context.Context, producerName string, consumer cent.Consumer) error
	Remove(name string) error
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
		cons := MakeConsumer(streamName, conn)
		err := c.center.AddConsumer(ctx, streamName, cons)
		if err != nil {
			log.Warnf("Failed to register new consumer: %v", err)
			return rtmp.NETSTREAM_PLAY_NOTFOUND
		}
		log.Debugf("Consumer ready to play: %v", streamName)
		go c.HandleConsumer(ctx, cons)
		return rtmp.NETSTREAM_PLAY_START
	})

	conn.handle.OnPublish(func(app, streamName string) rtmp.StatusCode {
		conn.handle.OnFrame(func(cid codec.CodecID, pts, dts uint32, frame []byte) {
			f := &ds.Frame{
				Codec: cid,
				Data:  frame,
				PTS:   pts,
				DTS:   dts,
			}
			conn.C <- f
		})
		p := newMediaProducer(streamName, conn)
		err := c.center.Register(streamName, p)
		if err != nil {
			log.Warnf("Failed to register new producer: %v", err)
			return rtmp.NETSTREAM_CONNECT_REJECTED
		}
		go c.HandlerProducer(ctx, p)
		log.Debugf("Producer ready to publish: %v", streamName)
		return rtmp.NETSTREAM_PUBLISH_START
	})

	conn.handle.SetOutput(func(b []byte) error {
		_, err := conn.conn.Write(b)
		return err
	})

	conn.handle.OnStateChange(func(newState rtmp.RtmpState) {
		if newState == rtmp.STATE_RTMP_PLAY_START {
			fmt.Println("play start")
		} else if newState == rtmp.STATE_RTMP_PUBLISH_START {
			fmt.Println("publish start")
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
