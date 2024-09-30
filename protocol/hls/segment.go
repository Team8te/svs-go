package hls

import (
	"fmt"

	"github.com/Team8te/svs-go/ds"
	"github.com/yapingcat/gomedia/go-codec"
)

type segment struct {
	id       int
	start    uint32
	end      uint32
	duration float32
	uri      string
	name     string
	frames   []*ds.Frame
	count    int
}

func newSegment(id int) *segment {
	return &segment{
		id:     id,
		frames: make([]*ds.Frame, 0, 100),
		count:  0,
	}
}

func (s *segment) setFrame(f *ds.Frame) {
	isIFrame := f.IsVideo() && codec.IsH264IDRFrame(f.Data)
	if s.count == 0 && !isIFrame {
		return
	}
	s.frames = append(s.frames, f)
	if isIFrame {
		s.count++
	}
}

func (s *segment) buildMeta(name string) {
	s.name = name
	s.start = s.frames[0].DTS
	s.end = s.frames[len(s.frames)-1].DTS
	duration := s.end - s.start
	s.duration = float32(duration) / 1000
	s.uri = fmt.Sprintf("sequence-%v-id-%d.ts", s.name, s.id)
}

func (s *segment) getDuration() float32 {
	if len(s.frames) == 0 {
		return 0
	}
	start := s.frames[0].DTS
	end := s.frames[len(s.frames)-1].DTS
	duration := end - start
	return float32(duration) / 1000
}
