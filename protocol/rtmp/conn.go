package rtmp

import (
	"context"
	"net"

	"github.com/Team8te/svs-go/configure"
	"github.com/Team8te/svs-go/ds"
	"github.com/Team8te/svs-go/media/mp4"
	log "github.com/sirupsen/logrus"
	"github.com/yapingcat/gomedia/go-rtmp"
)

type worker interface {
	Close()
}

type rtmpConn struct {
	cancel context.CancelFunc
	conn   net.Conn
	handle *rtmp.RtmpServerHandle

	w worker

	r  roomSerice
	st streamer
}

func (s *Server) newConn(c net.Conn) *rtmpConn {
	return &rtmpConn{
		conn:   c,
		handle: rtmp.NewRtmpServerHandle(),
		r:      s.r,
		st:     s.st,
	}
}

func (s *rtmpConn) write(f *ds.Frame) error {
	return s.handle.WriteFrame(f.Codec, f.Data, f.PTS, f.DTS)
}

func (s *rtmpConn) init() {
	s.handle.OnPlay(func(app, streamName string, start, duration float64, reset bool) rtmp.StatusCode {
		ctx := context.Background()
		r, err := s.r.GetRoomByName(ctx, streamName)
		if err != nil {
			return rtmp.NETSTREAM_PLAY_NOTFOUND
		}
		log.Infof("new sub. Stream id: %v . Room id: %v, name: %v", streamName, r.ID, r.Name)
		ns := &subscriber{
			conn:       s,
			firstVideo: true,
		}

		s.st.AddSubscribers(r.ID, ns)
		return rtmp.NETSTREAM_PLAY_START
	})

	s.handle.OnPublish(func(app, streamName string) rtmp.StatusCode {
		ctx := context.Background()
		return s.runPublusher(ctx, streamName)
	})

	s.handle.SetOutput(func(b []byte) error {
		_, err := s.conn.Write(b)
		return err
	})
	s.handle.OnStateChange(func(newState rtmp.RtmpState) {
		switch newState {
		case rtmp.STATE_RTMP_PLAY_START:
			return
		case rtmp.STATE_RTMP_PUBLISH_START:
			if !configure.NeedArchive() {
				return
			}
			stream := s.handle.GetStreamName()
			ctx := context.Background()
			r, err := s.r.GetRoomByID(ctx, stream)
			if err != nil {
				return
			}

			w, _ := mp4.NewMP4Writer(stream + ".mp4")
			s.st.AddSubscribers(r.ID, w)
		}
	})
}

func (s *rtmpConn) runPublusher(ctx context.Context, stream string) rtmp.StatusCode {
	pub, err := makePublisher(ctx, stream, s.r, s.st)
	s.handle.OnFrame(pub.write)
	err = pub.start()
	if err != nil {
		pub.Close()
		return rtmp.NETSTREAM_CONNECT_REJECTED
	}
	s.w = pub
	return rtmp.NETSTREAM_PUBLISH_START
}

func (s *rtmpConn) run(ctx context.Context) {
	defer s.conn.Close()
	buf := make([]byte, maxBufferSize)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			err := s.do(buf)
			if err != nil {
				s.close()
				return
			}
		}
	}
}

func (s *rtmpConn) do(buf []byte) error {
	n, err := s.conn.Read(buf)
	if err != nil {
		log.Error("failed to read chunk", "error: ", err)
		return err
	}
	err = s.handle.Input(buf[:n])
	if err != nil {
		log.Error("failed to process chunk", "error: ", err)
		return err
	}

	return nil
}

func (s *rtmpConn) close() {
	s.cancel()
	s.w.Close()
}
