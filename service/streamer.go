package service

import (
	"context"
	"sync"

	"github.com/Team8te/svs-go/ds"
	"github.com/Team8te/svs-go/pkg/av"
	"github.com/Team8te/svs-go/protocol/rtmp/cache"
)

type Streamer struct {
	mx      sync.RWMutex
	streams map[ds.RoomID]*stream
}

func NewStreamer() *Streamer {
	return &Streamer{
		streams: map[ds.RoomID]*stream{},
	}
}

func (st *Streamer) CreateStreamAndBind(id ds.RoomID, pub av.Publisher) error {
	s := st.FindStream(id)
	if s == nil {
		s = st.createEmptyStream()
		st.EmplaceStream(id, s)
	}

	s.pub = pub
	return nil
}

func (st *Streamer) createEmptyStream() *stream {
	s := &stream{
		cache: cache.NewCache(),
	}
	return s
}

func (st *Streamer) EmplaceStream(id ds.RoomID, s *stream) {
	st.mx.Lock()
	defer st.mx.Unlock()
	st.streams[id] = s
}

func (st *Streamer) AddSubscribers(id ds.RoomID, subs ...av.Subscriber) error {
	stream := st.FindStream(id)
	if stream == nil {
		stream = st.createEmptyStream()
		st.EmplaceStream(id, stream)
	}

	stream.addSubs(subs...)
	return nil
}

func (st *Streamer) FindStream(id ds.RoomID) *stream {
	st.mx.RLock()
	defer st.mx.RUnlock()
	stream, ok := st.streams[id]
	if !ok {
		return nil
	}

	return stream
}

func (st *Streamer) StartStream(id ds.RoomID) error {
	st.mx.Lock()
	defer st.mx.Unlock()
	s, ok := st.streams[id]
	if !ok {
		return ds.ErrorNotFound
	}
	go s.run(context.Background())
	return nil
}

func (st *Streamer) StopStream(id ds.RoomID) error {
	st.mx.Lock()
	defer st.mx.Unlock()
	s, ok := st.streams[id]
	if !ok {
		return ds.ErrorNotFound
	}
	s.stop()
	return nil
}

func (st *Streamer) RemoveStream(id ds.RoomID) error {
	st.mx.Lock()
	defer st.mx.Unlock()
	s, ok := st.streams[id]
	if !ok {
		return ds.ErrorNotFound
	}
	s.stop()
	delete(st.streams, id)
	return nil
}
