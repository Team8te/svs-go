package service

import (
	"context"
	"sync"
	"time"

	"github.com/Team8te/svs-go/ds"
	"github.com/Team8te/svs-go/pkg/av"
	"github.com/Team8te/svs-go/pkg/utils/uid"
	log "github.com/sirupsen/logrus"
)

type stream struct {
	cancel context.CancelFunc
	mxp    sync.RWMutex
	pub    av.Publisher

	subs sync.Map
}

func (s *stream) setPub(pub av.Publisher) error {
	s.mxp.Lock()
	defer s.mxp.Unlock()
	if s.pub != nil {
		return ds.ErrorAlreadyUsed
	}
	s.pub = pub
	return nil
}

func (s *stream) do() {
	t := time.Now()
	defer func() {
		v := time.Since(t).Microseconds()
		log.Debugf("Stream processed time: %v", v)
	}()
	frame, err := s.pub.ReadFrame()
	if err != nil {
		log.Errorf("stream runtime error: %v", err)
		return
	}

	if frame == nil {
		return
	}

	s.subs.Range(func(k, v interface{}) bool {
		sub := v.(av.Subscriber)
		err := sub.Write(frame)
		if err != nil {
			sub.Close()
			s.subs.Delete(k)
			log.Errorf("send cache packet error: %v", err)
		}
		return true
	})
}

func (s *stream) run(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	for {
		select {
		case <-ctx.Done():
			return
		default:
			s.do()
		}
	}
}

func (s *stream) addSubs(subs ...av.Subscriber) {
	for _, sub := range subs {
		s.subs.Store(uid.NewId(), sub)
	}
}

func (s *stream) stop() {
	s.cancel()
}

func (s *stream) removeAllSubs() {
	s.subs.Range(func(k, v interface{}) bool {
		sub := v.(av.Subscriber)
		sub.Close()
		s.subs.Delete(k)

		return true
	})
}
