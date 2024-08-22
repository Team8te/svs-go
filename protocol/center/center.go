package center

import (
	"context"
	"fmt"
	"sync"

	"github.com/Team8te/svs-go/ds"
)

type Consumer interface {
	Name() string
	ID() string
	Run(ctx context.Context)
	Close()
	IsAlive() bool
	Play(name string, frame *ds.Frame)
}

type MediaProducer interface {
	Name() string
	RemoveConsumer(id string)
	AddConsumer(consumer Consumer)
	Dispatch(ctx context.Context)
}

type MediaCenter struct {
	center map[string]MediaProducer
	mtx    sync.Mutex
}

func MakeMediaCenter() *MediaCenter {
	return &MediaCenter{
		center: make(map[string]MediaProducer),
	}
}

func (c *MediaCenter) Register(name string, p MediaProducer) error {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	current := c.center[name]
	if current != nil {
		return fmt.Errorf("Already producer exists: %v", name)
	}
	c.center[name] = p
	return nil
}

func (c *MediaCenter) Remove(name string) error {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	p := c.center[name]
	if p == nil {
		return fmt.Errorf("Not found: %v", name)
	}
	delete(c.center, name)
	return nil
}

func (c *MediaCenter) Find(name string) MediaProducer {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	if p, found := c.center[name]; found {
		return p
	} else {
		return nil
	}
}

func (c *MediaCenter) AddConsumer(_ context.Context, producerName string, consumer Consumer) error {
	p := c.Find(producerName)
	if p == nil {
		return fmt.Errorf("producer not found: %v", producerName)
	}

	p.AddConsumer(consumer)
	return nil
}

func (c *MediaCenter) RemoveConsumer(_ context.Context, producerName string, id string) error {
	p := c.Find(producerName)
	if p == nil {
		return fmt.Errorf("producer not found: %v", producerName)
	}
	p.RemoveConsumer(id)
	return nil
}
