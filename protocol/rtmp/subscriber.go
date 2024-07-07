package rtmp

import (
	"github.com/Team8te/svs-go/ds"
	"github.com/yapingcat/gomedia/go-codec"
)

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
