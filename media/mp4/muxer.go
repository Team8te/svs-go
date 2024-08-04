package mp4

import (
	"os"

	"github.com/Team8te/svs-go/ds"
	log "github.com/sirupsen/logrus"
	"github.com/yapingcat/gomedia/go-codec"
	"github.com/yapingcat/gomedia/go-mp4"
)

type streamType string

const (
	audioType = streamType("audio")
	videoType = streamType("video")
)

const h264DefaultHZ = 90

type MP4Writer struct {
	f       *os.File
	muxer   *mp4.Movmuxer
	streams map[streamType]uint32
}

func NewMP4Muxer(name string) (*MP4Writer, error) {
	mp4file, err := os.OpenFile(name, os.O_CREATE|os.O_RDWR, 0666)
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
		streams: map[streamType]uint32{},
	}, nil
}

func (m *MP4Writer) Write(f *ds.Frame) error {
	if f.IsAudio() {
		id, ok := m.streams[audioType]
		if !ok {
			id = m.muxer.AddAudioTrack(toMP4CodecType(f.Codec))
			m.streams[audioType] = id
		}
		return m.muxer.Write(id, f.Data, uint64(f.PTS), uint64(f.DTS))
	} else if f.IsVideo() {
		id, ok := m.streams[videoType]
		if !ok {
			id = m.muxer.AddVideoTrack(toMP4CodecType(f.Codec))
			m.streams[videoType] = id
		}
		return m.muxer.Write(id, f.Data, uint64(f.PTS), uint64(f.DTS))
	}
	return nil
}

func (m *MP4Writer) Close() {
	err := m.muxer.WriteTrailer()
	if err != nil {
		log.Errorf("failed to close file. Error: %v", err)
	}
	m.f.Close()
}

func (m *MP4Writer) Alive() bool {
	return true
}

func (m *MP4Writer) CalcBaseTimestamp() {

}

func toMP4CodecType(c codec.CodecID) mp4.MP4_CODEC_TYPE {
	switch c {
	case codec.CODECID_VIDEO_H264:
		return mp4.MP4_CODEC_H264
	case codec.CODECID_VIDEO_H265:
		return mp4.MP4_CODEC_H265

	case codec.CODECID_AUDIO_AAC:
		return mp4.MP4_CODEC_AAC
	case codec.CODECID_AUDIO_G711A:
		return mp4.MP4_CODEC_G711A
	case codec.CODECID_AUDIO_G711U:
		return mp4.MP4_CODEC_G711U
	case codec.CODECID_AUDIO_OPUS:
		return mp4.MP4_CODEC_OPUS
	case codec.CODECID_AUDIO_MP3:
		return mp4.MP4_CODEC_MP3
	default:
		return mp4.MP4_CODEC_TYPE(-1)
	}
}
