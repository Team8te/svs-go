package ds

import "github.com/yapingcat/gomedia/go-codec"

type Frame struct {
	Codec codec.CodecID
	Data  []byte
	PTS   uint32
	DTS   uint32
}
