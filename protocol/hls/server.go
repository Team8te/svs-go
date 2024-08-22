package hls

import (
	"bytes"
	"container/list"
	"context"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/Team8te/svs-go/ds"
	"github.com/Team8te/svs-go/protocol/center"
	log "github.com/sirupsen/logrus"
	"github.com/yapingcat/gomedia/go-mp4"
	"github.com/yapingcat/gomedia/go-mpeg2"
)

type hlsSession struct {
}

type mp4Segment struct {
	start       uint64
	end         uint64
	duration    float32
	uri         string
	description string
}

type hlsmuxer struct {
	mode       string
	segments   []*mp4Segment
	streamName string
	duration   int
}

func (muxer *hlsmuxer) makeM3u8() string {
	buf := make([]byte, 0, 4096)
	m3u := bytes.NewBuffer(buf)
	maxDuration := 0
	for _, seg := range muxer.segments {
		duration := seg.end - seg.start
		seg.duration = float32(duration) / 1000
		if maxDuration < int(math.Ceil(float64(seg.duration))) {
			maxDuration = int(math.Ceil(float64(seg.duration)))
		}
	}

	m3u.WriteString("#EXTM3U\n")
	m3u.WriteString(fmt.Sprintf("#EXT-X-TARGETDURATION:%d\n", maxDuration))
	m3u.WriteString("#EXT-X-VERSION:3\n")
	m3u.WriteString("#EXT-X-MEDIA-SEQUENCE:0\n")

	for _, seg := range muxer.segments {
		fmt.Println(seg.start, seg.end)
		m3u.WriteString(fmt.Sprintf("#EXTINF:%.3f,%s\n", seg.duration, seg.description))
		m3u.WriteString(muxer.streamName + "/" + seg.uri + "\n")
	}
	m3u.WriteString("#EXT-X-ENDLIST\n")
	return m3u.String()
}

func (muxer *hlsmuxer) makeHlsSegment(table []mp4.SyncSample, endTimestamp uint64) {
	if len(table) == 0 {
		return
	}

	idx := 0
	start := table[idx].Dts
	for start < table[len(table)-1].Dts {
		if idx < len(table)-1 && table[idx].Dts-start < uint64(muxer.duration)*1000 {
			idx++
			continue
		}
		seg := &mp4Segment{
			start:       start,
			end:         table[idx].Dts,
			description: fmt.Sprintf("mp4 sync sample %d", idx),
			uri:         fmt.Sprintf("sequence-%d.ts?start=%d&end=%d", len(muxer.segments), start, table[idx].Dts),
		}
		muxer.segments = append(muxer.segments, seg)
		start = table[idx].Dts
		idx++
	}
	if start < endTimestamp {
		seg := &mp4Segment{
			start:       start,
			end:         endTimestamp,
			description: fmt.Sprintf("last mp4 sync sample"),
			uri:         fmt.Sprintf("sequence-%d.ts?start=%d&end=%d", len(muxer.segments), start, endTimestamp),
		}
		muxer.segments = append(muxer.segments, seg)
	}
}

type MediaCenter interface {
	AddConsumer(ctx context.Context, producerName string, consumer center.Consumer) error
	RemoveConsumer(ctx context.Context, producerName string, id string) error
	Find(name string) center.MediaProducer
}

type HLSServer struct {
	l         net.Listener
	c         MediaCenter
	m3u8Cache map[string]*list.List
}

func NewHLSServer(l net.Listener, c MediaCenter) *HLSServer {
	return &HLSServer{
		l: l,
		c: c,
	}
}

func (hls *HLSServer) makeM3U8PlayList(stream string) []byte {
	return nil
}

func (hls *HLSServer) Play(name string, frame *ds.Frame) {
	
}

func (hls *HLSServer) onM3U8(w http.ResponseWriter, r *http.Request) {
	streamName := strings.TrimLeft(r.URL.Path, "/live/")
	streamName = strings.TrimRight(streamName, ".m3u8")
	fileName := streamName + ".mp4"
	//hls.c.AddConsumer(context.TODO(), streamName)
	f, err := os.Open(fileName)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer f.Close()

	demuxer := mp4.CreateMp4Demuxer(f)
	headInfo, err := demuxer.ReadHead()
	if err != nil && err != io.EOF {
		fmt.Println(err)
	} else {
		fmt.Printf("%+v\n", headInfo)
	}
	vid := 0
	var endTs uint64 = 0
	for _, info := range headInfo {
		if info.Cid == mp4.MP4_CODEC_H264 || info.Cid == mp4.MP4_CODEC_H265 {
			vid = info.TrackId
			endTs = info.EndDts
		}
	}
	table, err := demuxer.GetSyncTable(uint32(vid))
	if err != nil {
		fmt.Println(err)
	}

	muxer := hlsmuxer{duration: 10, streamName: streamName}
	muxer.makeHlsSegment(table, endTs)
	w.Header().Add("Content-Type", "application/vnd.apple.mpegurl")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "*")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	m := muxer.makeM3u8()
	fmt.Println(m)
	body := []byte(m)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(body)))
	w.Write(body)
}

func (hls *HLSServer) onTs(w http.ResponseWriter, r *http.Request) {
	start, _ := strconv.ParseInt(r.URL.Query().Get("start"), 10, 64)
	end, _ := strconv.ParseInt(r.URL.Query().Get("end"), 10, 64)
	fmt.Println("start:", start, "end:", end)
	streamName := strings.TrimLeft(r.URL.Path, "/live/")
	idx := strings.Index(streamName, "/")
	streamName = streamName[:idx]
	fileName := streamName + ".mp4"
	f, err := os.Open(fileName)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer f.Close()
	demuxer := mp4.CreateMp4Demuxer(f)
	demuxer.ReadHead()
	demuxer.SeekTime(uint64(start))

	buf := bytes.NewBuffer(make([]byte, 0, 1024*1024))

	muxer := mpeg2.NewTSMuxer()
	muxer.OnPacket = func(pkg []byte) {
		buf.Write(pkg)
	}

	vid := muxer.AddStream(mpeg2.TS_STREAM_H264)
	aid := muxer.AddStream(mpeg2.TS_STREAM_AAC)
	first := true
	for {
		pkg, err := demuxer.ReadPacket()
		if err != nil {
			w.Write([]byte(err.Error()))
			return
		}
		if first && pkg.Cid == mp4.MP4_CODEC_H264 {
			fmt.Println("Go pkg:", pkg.Pts, " ", pkg.Dts)
			first = false
		}
		//
		if pkg.Dts >= uint64(end) {
			break
		}
		if pkg.Cid == mp4.MP4_CODEC_H264 {
			muxer.Write(vid, pkg.Data, pkg.Pts, pkg.Dts)
		} else if pkg.Cid == mp4.MP4_CODEC_AAC {
			muxer.Write(aid, pkg.Data, pkg.Pts, pkg.Dts)
		}
	}
	fmt.Println("ts segment length:", buf.Len())
	w.Header().Set("Content-Length", fmt.Sprintf("%d", buf.Len()))
	w.Header().Set("Content-Type", "video/mp2t")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "*")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Write(buf.Bytes())
}

func (hls *HLSServer) handle(w http.ResponseWriter, r *http.Request) {
	if path.Base(r.URL.Path) == "crossdomain.xml" {
		w.Header().Set("Content-Type", "application/xml")
		w.Write(crossdomainxml)
		return
	}
	switch path.Ext(r.URL.Path) {
	case ".m3u8":
		hls.onM3U8(w, r)
	case ".ts":
		hls.onTs(w, r)
	default:
		http.Error(w, BadRequest.Error(), http.StatusBadRequest)
	}

}

func (hls *HLSServer) Run(ctx context.Context) {
	select {
	case <-ctx.Done():
		break
	default:
		hls.start()
	}
}

func (hls *HLSServer) start() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		hls.handle(w, r)
	})

	log.Info("HTTP-HLS listen On ", hls.l.Addr().String())
	http.Serve(hls.l, mux)
}
