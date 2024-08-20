package rtmp

import (
	"context"
	"fmt"
	"sync"

	cent "github.com/Team8te/svs-go/protocol/center"
)

type MediaProducer struct {
	name      string
	session   *MediaSession
	mtx       sync.Mutex
	consumers []cent.Consumer
	quit      chan struct{}
	die       sync.Once
}

func newMediaProducer(name string, sess *MediaSession) *MediaProducer {
	return &MediaProducer{
		name:      name,
		session:   sess,
		consumers: make([]cent.Consumer, 0, 10),
		quit:      make(chan struct{}),
	}
}

func (producer *MediaProducer) Stop() {
	producer.die.Do(func() {
		close(producer.quit)
	})
}

func (producer *MediaProducer) Dispatch(ctx context.Context) {
	defer func() {
		fmt.Println("quit dispatch")
		producer.Stop()
	}()
	for {
		select {
		case frame := <-producer.session.C:
			if frame == nil {
				continue
			}
			producer.mtx.Lock()
			tmp := make([]cent.Consumer, len(producer.consumers))
			copy(tmp, producer.consumers)
			producer.mtx.Unlock()
			for _, c := range tmp {
				if c.IsAlive() {
					tmp := frame.Clone()
					c.Play(tmp)
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

func (producer *MediaProducer) AddConsumer(consumer cent.Consumer) {
	producer.mtx.Lock()
	defer producer.mtx.Unlock()
	producer.consumers = append(producer.consumers, consumer)
}

func (producer *MediaProducer) RemoveConsumer(id string) {
	producer.mtx.Lock()
	defer producer.mtx.Unlock()
	res := make([]cent.Consumer, 0, len(producer.consumers)-1)
	for _, consume := range producer.consumers {
		if consume.ID() != id {
			res = append(res, consume)
		}
	}

	producer.consumers = res
}

func (producer *MediaProducer) Name() string {
	return producer.name
}
