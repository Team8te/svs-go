package mp4

import (
	"os"

	"github.com/Team8te/svs-go/pkg/av"
	"github.com/yapingcat/gomedia/go-mp4"
)

type streamType string

const (
	audioType = streamType("audio")
	videoType = streamType("video")
)

const h264DefaultHZ = 90

type stream struct {
	id uint32
	t  streamType
}

type MP4Writer struct {
	f       *os.File
	muxer   *mp4.Movmuxer
	streams map[int64]*stream
	info    av.Info
}

func NewMP4Writer(info av.Info) (*MP4Writer, error) {
	mp4filename := "test2_fmp4.mp4"
	mp4file, err := os.OpenFile(mp4filename, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return nil, err
	}
	muxer, err := mp4.CreateMp4Muxer(mp4file)
	if err != nil {
		return nil, err
	}
	return &MP4Writer{
		f:       mp4file,
		muxer:   muxer,
		streams: map[int64]*stream{},
	}, nil
}

func (m *MP4Writer) Write(p *av.Packet) error {
	st, ok := m.streams[int64(p.StreamID)]
	if !ok {
		st = &stream{}
		if p.IsAudio {
			st.id = m.muxer.AddAudioTrack(mp4.MP4_CODEC_AAC)
			st.t = audioType
		} else if p.IsVideo {
			st.id = m.muxer.AddVideoTrack(mp4.MP4_CODEC_H264)
			st.t = videoType
		} else {
			return nil
		}
		m.streams[int64(p.StreamID)] = st
	}

	dts := uint64(p.TimeStamp) * uint64(h264DefaultHZ)
	pts := dts
	var videoH av.VideoPacketHeader
	if p.IsVideo {
		videoH, _ = p.Header.(av.VideoPacketHeader)
		pts = dts + uint64(videoH.CompositionTime())*uint64(h264DefaultHZ)
	}

	return m.muxer.Write(st.id, p.Data, pts, dts)
}

func (m *MP4Writer) Close(error) {
	defer m.f.Close()
	m.muxer.WriteTrailer()
}

func (m *MP4Writer) Alive() bool {
	return true
}

func (m *MP4Writer) CalcBaseTimestamp() {

}

func (m *MP4Writer) Info() av.Info {
	return m.info
}
