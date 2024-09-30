package hls

import (
	"bytes"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/yapingcat/gomedia/go-mpeg2"
)

func (m3u8 *m3u8server) makeTS(seqID string) ([]byte, error) {
	t := time.Now()
	defer func() {
		log.Debugf("makeTS process time: %v ms", time.Since(t).Milliseconds())
	}()
	buf := bytes.NewBuffer(make([]byte, 0, 1024*1024))

	muxer := mpeg2.NewTSMuxer()
	muxer.OnPacket = func(pkg []byte) {
		buf.Write(pkg)
	}
	seg := m3u8.segments[seqID]
	if seg == nil {
		return nil, fmt.Errorf("invalid segment")
	}
	vid := muxer.AddStream(mpeg2.TS_STREAM_H264)
	aid := muxer.AddStream(mpeg2.TS_STREAM_AAC)

	for _, f := range seg.frames {
		if f.DTS > seg.end {
			break
		}

		if f.IsVideo() {
			muxer.Write(vid, f.Data, uint64(f.PTS), uint64(f.DTS))
		} else if f.IsAudio() {
			muxer.Write(aid, f.Data, uint64(f.PTS), uint64(f.DTS))
		}
	}

	log.Debugln("ts segment length: ", buf.Len())
	return buf.Bytes(), nil
}
