package rtmp

import (
	"context"
	"net"

	"github.com/Team8te/svs-go/configure"
	"github.com/Team8te/svs-go/ds"
	"github.com/Team8te/svs-go/media/mp4"
	"github.com/Team8te/svs-go/pkg/utils/uid"
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

	sub worker
	pub worker

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

func (s *rtmpConn) Write(f *ds.Frame) error {
	return s.handle.WriteFrame(f.Codec, f.Data, f.PTS, f.DTS)
}

func (s *rtmpConn) init(ctx context.Context) {
	s.handle.OnPlay(func(app, streamName string, start, duration float64, reset bool) rtmp.StatusCode {
		r, err := s.r.GetRoomByName(ctx, streamName)
		if err != nil {
			return rtmp.NETSTREAM_PLAY_NOTFOUND
		}
		log.Infof("new sub. Stream id: %v . Room id: %v, name: %v", streamName, r.ID, r.Name)
		sub := MakeSubscriber(uid.NewId(), s)
		sub.run(ctx)

		err = s.st.BindSubscribers(r.ID, sub)
		if err != nil {
			return rtmp.NETSTREAM_PLAY_NOTFOUND
		}
		s.sub = sub
		return rtmp.NETSTREAM_PLAY_START
	})

	s.handle.OnPublish(func(app, streamName string) rtmp.StatusCode {
		pub, err := s.makeAndStartPublisher(ctx, streamName)
		if err != nil {
			log.Warnf("Failed to make new publisher for stream: %v. Error: %v", streamName, err)
			return rtmp.NETSTREAM_CONNECT_REJECTED
		}

		s.handle.OnFrame(pub.write)
		s.pub = pub
		return rtmp.NETSTREAM_PUBLISH_START
	})

	s.handle.SetOutput(func(b []byte) error {
		_, err := s.conn.Write(b)
		return err
	})
}

func (s *rtmpConn) makeAndStartPublisher(ctx context.Context, stream string) (*publisher, error) {
	var pub *publisher
	var err error
	defer func() {
		if err != nil {
			if pub != nil {
				pub.Close()
			}
		}
	}()
	pub, err = makePublisher(ctx, stream, s.r, s.st)
	if err != nil {
		return nil, err
	}

	if configure.NeedArchive() {
		w, _ := mp4.NewMP4Muxer(stream + ".mp4")
		sub := MakeSubscriber("archive", w)
		pub.BindSubscribers(ctx, sub)
		sub.run(ctx)
	}

	err = pub.start()
	if err != nil {
		return nil, err
	}

	return pub, nil
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
				s.Close()
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

func (s *rtmpConn) Close() {
	s.cancel()
	if s.pub != nil {
		s.pub.Close()
		s.pub = nil
	}

	if s.sub != nil {
		s.sub = nil
	}
}
