package service

import (
	"sync"
	"sync/atomic"

	"github.com/Team8te/svs-go/ds"
	"github.com/Team8te/svs-go/pkg/av"
	"github.com/Team8te/svs-go/protocol/rtmp/cache"
	log "github.com/sirupsen/logrus"
)

type stream struct {
	run atomic.Bool
	pub av.Publisher

	init  bool
	cache *cache.Cache

	mx   sync.RWMutex
	subs []av.Subscriber
}

func (s *stream) do() {
	s.run.Store(true)
	var p av.Packet
	for {
		if !s.run.Load() {
			return
		}
		err := s.pub.Read(&p)
		if err != nil {
			log.Errorf("stream runtime error: %v", err)
			return
		}
		s.cache.Write(p)
		for _, sub := range s.subs {
			if !s.init {
				if err := s.cache.Send(sub); err != nil {
					log.Debugf("[%s] send cache packet error: %v, remove", sub.Info(), err)
				}
				s.init = true
			}
			pp := p
			err := sub.Write(&pp)
			if err != nil {
				log.Errorf("stream runtime error. Failed send to sub: %v", err)
			}
		}
	}
}

func (s *stream) addSubs(subs ...av.Subscriber) {
	s.mx.Lock()
	defer s.mx.Unlock()
	s.subs = append(s.subs, subs...)
}

var globalID int64 = 0

type Streamer struct {
	mx      sync.RWMutex
	streams map[int64]*stream
}

func NewStreamer() *Streamer {
	return &Streamer{
		streams: map[int64]*stream{},
	}
}

func (st *Streamer) CreateStream(pub av.Publisher) (int64, error) {
	s := &stream{
		pub:   pub,
		cache: cache.NewCache(),
	}
	s.run.Store(false)
	st.mx.Lock()
	defer st.mx.Unlock()
	st.streams[globalID] = s

	r := globalID
	globalID++
	return r, nil
}

func (st *Streamer) AddSubscribers(id int64, subs ...av.Subscriber) error {
	st.mx.RLock()
	defer st.mx.RUnlock()
	stream, ok := st.streams[id]
	if !ok {
		return ds.ErrorNotFound
	}

	stream.addSubs(subs...)
	return nil
}

func (st *Streamer) StartStream(id int64) error {
	st.mx.Lock()
	defer st.mx.Unlock()
	s, ok := st.streams[id]
	if !ok {
		return ds.ErrorNotFound
	}
	go s.do()
	return nil
}

func (st *Streamer) StopStream(id int64) error {
	st.mx.Lock()
	defer st.mx.Unlock()
	s, ok := st.streams[id]
	if !ok {
		return ds.ErrorNotFound
	}
	s.run.Store(false)
	return nil
}

func (st *Streamer) RemoveStream(id int64) error {
	st.mx.Lock()
	defer st.mx.Unlock()
	s, ok := st.streams[id]
	if !ok {
		return ds.ErrorNotFound
	}
	s.run.Store(false)
	delete(st.streams, id)
	return nil
}
