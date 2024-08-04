package rtmp

import (
	"context"
	"net"

	"github.com/Team8te/svs-go/ds"
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

func (s *rtmpConn) init(ctx context.Context) {
	s.handle.OnPlay(func(app, streamName string, start, duration float64, reset bool) rtmp.StatusCode {
		r, err := s.r.GetRoomByName(ctx, streamName)
		if err != nil {
			return rtmp.NETSTREAM_PLAY_NOTFOUND
		}
		log.Infof("new sub. Stream id: %v . Room id: %v, name: %v", streamName, r.ID, r.Name)
		ns := MakeSubscriber(s)
		ns.run(ctx)

		s.st.AddSubscribers(r.ID, ns)
		s.w = ns
		return rtmp.NETSTREAM_PLAY_START
	})

	s.handle.OnPublish(func(app, streamName string) rtmp.StatusCode {
		pub, err := s.makeAndStartPublisher(ctx, streamName)
		if err != nil {
			log.Warnf("Failed to make new publisher for stream: %v. Error: %v", streamName, err)
			return rtmp.NETSTREAM_CONNECT_REJECTED
		}

		s.handle.OnFrame(pub.write)
		s.w = pub
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
	if s.w != nil {
		s.w.Close()
		s.w = nil
	}
}
