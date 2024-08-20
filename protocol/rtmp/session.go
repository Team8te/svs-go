package rtmp

import (
	"context"
	"fmt"
	"math/rand/v2"
	"net"
	"sync"

	"github.com/Team8te/svs-go/ds"
	log "github.com/sirupsen/logrus"
	"github.com/yapingcat/gomedia/go-rtmp"
)

type MediaSession struct {
	handle    *rtmp.RtmpServerHandle
	conn      net.Conn
	lists     []*ds.Frame
	mtx       sync.Mutex
	id        string
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
			log.Warningf("Failed to process session chunk: %v", err)
			return
		}
	}
}

func (sess *MediaSession) stop() {
	sess.die.Do(func() {
		fmt.Println("stop isReady = false")
		sess.cancel()
		sess.conn.Close()
		close(sess.frameCome)
		close(sess.C)
	})
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

func (sess *MediaSession) GetStreamName() string {
	return sess.handle.GetStreamName()
}
