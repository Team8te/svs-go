package rtmp

import (
	"context"
	"fmt"
	"math/rand/v2"
	"net"
	"sync"
	"sync/atomic"

	"github.com/Team8te/svs-go/ds"
	"github.com/yapingcat/gomedia/go-rtmp"
)

type MediaProducer struct {
	name      string
	session   *MediaSession
	mtx       sync.Mutex
	consumers []*MediaSession
	quit      chan struct{}
	die       sync.Once
}

func newMediaProducer(name string, sess *MediaSession) *MediaProducer {
	return &MediaProducer{
		name:      name,
		session:   sess,
		consumers: make([]*MediaSession, 0, 10),
		quit:      make(chan struct{}),
	}
}

func (producer *MediaProducer) stop() {
	producer.die.Do(func() {
		close(producer.quit)
	})
}

func (producer *MediaProducer) dispatch(ctx context.Context) {
	defer func() {
		fmt.Println("quit dispatch")
		producer.stop()
	}()
	for {
		select {
		case frame := <-producer.session.C:
			if frame == nil {
				continue
			}
			producer.mtx.Lock()
			tmp := make([]*MediaSession, len(producer.consumers))
			copy(tmp, producer.consumers)
			producer.mtx.Unlock()
			for _, c := range tmp {
				if c.ready() {
					tmp := frame.Clone()
					c.play(tmp)
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

func (producer *MediaProducer) addConsumer(consumer *MediaSession) {
	producer.mtx.Lock()
	defer producer.mtx.Unlock()
	producer.consumers = append(producer.consumers, consumer)
}

func (producer *MediaProducer) removeConsumer(id string) {
	producer.mtx.Lock()
	defer producer.mtx.Unlock()
	res := make([]*MediaSession, 0, len(producer.consumers)-1)
	for _, consume := range producer.consumers {
		if consume.id != id {
			res = append(res, consume)
		}
	}

	producer.consumers = res
}

type MediaSession struct {
	handle    *rtmp.RtmpServerHandle
	conn      net.Conn
	lists     []*ds.Frame
	mtx       sync.Mutex
	id        string
	isReady   atomic.Bool
	frameCome chan struct{}
	die       sync.Once
	C         chan *ds.Frame
	cancel    context.CancelFunc
}

func newMediaSession(conn net.Conn) *MediaSession {
	id := fmt.Sprintf("%d", rand.Uint64())
	s := &MediaSession{
		id:        id,
		conn:      conn,
		handle:    rtmp.NewRtmpServerHandle(),
		frameCome: make(chan struct{}, 1),
		C:         make(chan *ds.Frame, 30),
	}

	fmt.Println("newMediaSession isReady = false")
	s.isReady.Store(false)

	return s
}

func (sess *MediaSession) run(_ context.Context) {
	defer sess.stop()
	for {
		buf := make([]byte, 65536)
		n, err := sess.conn.Read(buf)
		if err != nil {
			fmt.Println(err)
			return
		}
		err = sess.handle.Input(buf[:n])
		if err != nil {
			fmt.Println(err)
			return
		}
	}
}

func (sess *MediaSession) stop() {
	sess.die.Do(func() {
		fmt.Println("stop isReady = false")
		sess.isReady.Store(false)
		sess.cancel()
		sess.conn.Close()
		close(sess.frameCome)
		close(sess.C)
	})
}

func (sess *MediaSession) ready() bool {
	return sess.isReady.Load()
}

func (sess *MediaSession) play(frame *ds.Frame) {
	sess.mtx.Lock()
	sess.lists = append(sess.lists, frame)
	sess.mtx.Unlock()
	select {
	case sess.frameCome <- struct{}{}:
	default:
	}
}
