package ds

import (
	"github.com/yapingcat/gomedia/go-codec"
)

type Frame struct {
	ID    int64
	Codec codec.CodecID
	Data  []byte
	PTS   uint32
	DTS   uint32
}

func (f *Frame) IsVideo() bool {
	switch f.Codec {
	case codec.CODECID_VIDEO_H264,
		codec.CODECID_VIDEO_H265,
		codec.CODECID_VIDEO_VP8:
		return true
	default:
		return false
	}
}

func (f *Frame) IsAudio() bool {
	switch f.Codec {
	case codec.CODECID_AUDIO_AAC,
		codec.CODECID_AUDIO_G711A,
		codec.CODECID_AUDIO_G711U,
		codec.CODECID_AUDIO_OPUS,
		codec.CODECID_AUDIO_MP3:
		return true
	default:
		return false
	}
}

func (f *Frame) Clone() *Frame {
	tmp := &Frame{
		Codec: f.Codec,
		PTS:   f.PTS,
		DTS:   f.DTS,
	}
	tmp.Data = make([]byte, len(f.Data))
	copy(tmp.Data, f.Data)
	return tmp
}
