package service

import (
	"context"
	"sync"

	"github.com/Team8te/svs-go/ds"
	"github.com/Team8te/svs-go/pkg/av"
	"github.com/Team8te/svs-go/protocol/rtmp/cache"
	log "github.com/sirupsen/logrus"
)

type stream struct {
	cancel context.CancelFunc
	mxp    sync.RWMutex
	pub    av.Publisher

	cache *cache.Cache
	subs  sync.Map
}

type subscriber struct {
	av.Subscriber
	init bool
}

func (s *stream) setPub(pub av.Publisher) error {
	s.mxp.Lock()
	defer s.mxp.Unlock()
	if s.	pub != nil {
		return ds.ErrorAlreadyUsed
	}
	s.pub = pub
	return nil
}

func (s *stream) do(p av.Packet) {
	err := s.pub.Read(&p)
	if err != nil {
		log.Errorf("stream runtime error: %v", err)
		return
	}
	s.cache.Write(p)

	s.subs.Range(func(k, v interface{}) bool {
		sub := v.(*subscriber)
		if !sub.init {
			if err := s.cache.Send(sub); err != nil {
				log.Debugf("[%s] send cache packet error: %v, remove", sub.Info(), err)
			}
			sub.init = true
		} else {
			pp := p
			err := sub.Write(&pp)
			if err != nil {
				log.Errorf("stream runtime error. Failed send to sub: %v", err)
				s.subs.Delete(k)
			}
		}

		return true
	})
}

func (s *stream) run(ctx context.Context) {
	var p av.Packet
	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	for {
		select {
		case <-ctx.Done():
			return
		default:
			s.do(p)
		}
	}
}

func (s *stream) addSubs(subs ...av.Subscriber) {
	for _, sub := range subs {
		s.subs.Store(sub.Info().UID, &subscriber{
			Subscriber: sub,
			init:       false,
		})
	}
}

func (s *stream) stop() {
	s.cancel()
}
