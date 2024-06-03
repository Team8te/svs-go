package service

import (
	"context"

	"github.com/Team8te/svs-go/ds"
)

type repository interface {
	CreateRoom(_ context.Context, name string) (*ds.Room, error)
	GetRoomByName(_ context.Context, name string) (*ds.Room, error)
	DeleteRoomByName(ctx context.Context, name string) error
}

// Service ...
type Service struct {
	r repository
}

// NewService ...
func NewService() *Service {
	return &Service{}
}
