package hls

import (
	"bytes"
	"container/list"
	"context"
	"fmt"
	"math"

	"github.com/Team8te/svs-go/ds"
)

type m3u8server struct {
	name        string
	id          string
	counter     int
	limit       int
	maxDuration int
	ll          *list.List
	segments    map[string]*segment
	current     *segment
	end         bool
}

func NewM3u8Server(name string, id string, maxDuration int) *m3u8server {
	return &m3u8server{
		name:        name,
		id:          id,
		counter:     0,
		limit:       100000,
		maxDuration: maxDuration,
		ll:          list.New(),
		current:     newSegment(0),
		segments:    make(map[string]*segment),
	}
}

func (m3u8 *m3u8server) SetFrame(frame *ds.Frame) {
	m3u8.current.setFrame(frame)

	if m3u8.current.getDuration() < float32(m3u8.maxDuration) {
		return
	}

	m3u8.current.buildMeta(m3u8.name)

	m3u8.ll.PushBack(m3u8.current)
	m3u8.segments[m3u8.current.uri] = m3u8.current
	m3u8.counter++
	m3u8.current = newSegment(m3u8.counter)
	if m3u8.ll.Len() >= m3u8.limit {
		m3u8.ll.Remove(m3u8.ll.Front())
	}
}

func (m3u8 *m3u8server) makeM3u8(name string) []byte {
	buff := bytes.NewBuffer(make([]byte, 0, 4096))
	maxDuration := 0
	for e := m3u8.ll.Front(); e != nil; e = e.Next() {
		seg := e.Value.(*segment)
		if maxDuration < int(math.Ceil(float64(seg.duration))) {
			maxDuration = int(math.Ceil(float64(seg.duration)))
		}
	}
	seq := 0
	body := bytes.NewBuffer(nil)
	for e := m3u8.ll.Front(); e != nil; e = e.Next() {
		seg := e.Value.(*segment)
		body.WriteString(fmt.Sprintf("#EXTINF:%.3f,%s\n", seg.duration, "live"))
		body.WriteString(name + "/" + seg.uri + "\n")
		seq++
	}
	buff.WriteString("#EXTM3U\n")
	buff.WriteString("#EXT-X-VERSION:3\n")
	buff.WriteString("#EXT-X-ALLOW-CACHE:NO\n")
	//buff.WriteString("#EXT-X-PLAYLIST-TYPE:EVENT\n")
	buff.WriteString(fmt.Sprintf("#EXT-X-TARGETDURATION:%d\n", m3u8.maxDuration))
	buff.WriteString(fmt.Sprintf("#EXT-X-MEDIA-SEQUENCE:%v\n", 1000))
	buff.Write(body.Bytes())
	if m3u8.end {
		buff.WriteString("#EXT-X-ENDLIST\n")
	}
	fmt.Println(buff.String())

	return buff.Bytes()
}

func (m3u8 *m3u8server) Play(name string, f *ds.Frame) {
	m3u8.SetFrame(f)
}

func (m3u8 *m3u8server) Close() {
	m3u8.current = nil
	m3u8.ll = nil
	m3u8.name = ""
	m3u8.id = ""
}

func (m3u8 *m3u8server) ID() string {
	return m3u8.id
}

func (m3u8 *m3u8server) IsAlive() bool {
	return true
}

func (m3u8 *m3u8server) Name() string {
	return m3u8.name
}

func (m3u8 *m3u8server) Run(ctx context.Context) {

}
