package rtmp

import (
	"context"
	"net"

	"github.com/Team8te/svs-go/ds"
	log "github.com/sirupsen/logrus"
	"github.com/yapingcat/gomedia/go-codec"
	"github.com/yapingcat/gomedia/go-rtmp"
)

const (
	frameBufferCount = 100000
)

type Server struct {
	l net.Listener
	Service
}

func NewServer(listener net.Listener, r roomSerice, st streamer) *Server {
	return &Server{
		l: listener,
		Service: Service{
			r:  r,
			st: st,
		},
	}
}

func (s *Server) Run() {
	for {
		c, err := s.l.Accept()
		if err != nil {
			return
		}
		conn := s.newConn(c)
		conn.init()
		ctx := context.Background()
		ctx, conn.cancel = context.WithCancel(ctx)
		go conn.run(ctx)
	}
}

type rtmpConn struct {
	cancel context.CancelFunc
	conn   net.Conn
	handle *rtmp.RtmpServerHandle

	Service
}

func (s *Server) newConn(c net.Conn) *rtmpConn {
	return &rtmpConn{
		conn:   c,
		handle: rtmp.NewRtmpServerHandle(),
		Service: Service{
			r:  s.r,
			st: s.st,
		},
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
		r, err := s.r.GetRoomByID(ctx, streamName)
		if err != nil {
			return rtmp.NETSTREAM_CONNECT_REJECTED
		}

		pub := &publisher{
			pubChan: make(chan *ds.Frame, frameBufferCount),
			conn:    s,
		}
		s.handle.OnFrame(pub.write)
		err = s.st.CreateStreamAndBind(r.ID, pub)
		if err != nil {
			return rtmp.NETSTREAM_CONNECT_REJECTED
		}
		err = s.st.StartStream(r.ID)
		if err != nil {
			s.st.RemoveStream(r.ID)
			return rtmp.NETSTREAM_CONNECT_REJECTED
		}
		return rtmp.NETSTREAM_PUBLISH_START
	})

	s.handle.SetOutput(func(b []byte) error {
		_, err := s.conn.Write(b)
		return err
	})

	s.handle.OnStateChange(func(newState rtmp.RtmpState) {
		if newState == rtmp.STATE_RTMP_PLAY_START {
			log.Debug("new client, connect remote: ", s.conn.RemoteAddr().String(),
				"local:", s.conn.LocalAddr().String())

		} else if newState == rtmp.STATE_RTMP_PUBLISH_START {
			//s.handle.OnFrame(s.pub.write)
		}
	})
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
}

type publisher struct {
	pubChan chan *ds.Frame
	conn    *rtmpConn
}

func (p *publisher) write(cid codec.CodecID, pts, dts uint32, frame []byte) {
	f := &ds.Frame{
		Codec: cid,
		Data:  frame, //make([]byte, len(frame)),
		PTS:   pts,
		DTS:   dts,
	}
	p.pubChan <- f
}

func (p *publisher) ReadFrame() (*ds.Frame, error) {
	return <-p.pubChan, nil
}

func (p *publisher) Close() {
	close(p.pubChan)
}

type subscriber struct {
	firstVideo bool
	conn       *rtmpConn
}

func (sub *subscriber) Write(f *ds.Frame) error {
	if sub.firstVideo { //wait for I frame
		if f.Codec == codec.CODECID_VIDEO_H264 && codec.IsH264IDRFrame(f.Data) {
			sub.firstVideo = false
		} else if f.Codec == codec.CODECID_VIDEO_H265 && codec.IsH265IDRFrame(f.Data) {
			sub.firstVideo = false
		} else {
			return nil
		}
	}
	return sub.conn.write(f)
}

func (sub *subscriber) Close() {
	sub.conn.cancel()
}
