package hls

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/Team8te/svs-go/pkg/utils/uid"
	"github.com/Team8te/svs-go/protocol/center"
	log "github.com/sirupsen/logrus"
)

type HLSServer struct {
	l        net.Listener
	producer map[string]*m3u8server
}

func NewHLSServer(l net.Listener) *HLSServer {
	return &HLSServer{
		l:        l,
		producer: map[string]*m3u8server{},
	}
}

func (hls *HLSServer) CreateConsumer(name string) (center.Consumer, error) {
	v := hls.producer[name]
	if v != nil {
		return nil, fmt.Errorf("Already exists")
	}
	v = NewM3u8Server(name, uid.NewId(), 3)
	hls.producer[name] = v
	return v, nil
}

func (hls *HLSServer) onM3U8(w http.ResponseWriter, r *http.Request) {
	streamName := strings.TrimLeft(r.URL.Path, "/live/")
	streamName = strings.TrimRight(streamName, ".m3u8")

	s := hls.producer[streamName]
	if s == nil {
		return
	}

	w.Header().Add("Content-Type", "application/vnd.apple.mpegurl")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "*")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	body := s.makeM3u8(streamName)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(body)))
	w.Write(body)
}

func (hls *HLSServer) onTs(w http.ResponseWriter, r *http.Request) {
	log.Debugln("OnTs ", r.URL.Path)
	path := strings.TrimLeft(r.URL.Path, "/live/")
	id := strings.Index(path, "/")
	streamName := path[:id]
	seqID := path[id+1:]

	s := hls.producer[streamName]
	body, err := s.makeTS(seqID)
	if err != nil {
		return
	}

	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(body)))
	w.Header().Set("Content-Type", "video/mp2t")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "*")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Write(body)
}

func (hls *HLSServer) handle(w http.ResponseWriter, r *http.Request) {
	t := time.Now()
	defer func() {
		log.Debugf("handle resp time: %v ms, path: %v", time.Since(t).Milliseconds(), r.URL.Path)
	}()
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
