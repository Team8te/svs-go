package service

import (
	"context"

	"github.com/Team8te/svs-go/ds"
)

func (s *Service) CreateNewRoom(ctx context.Context, name string) (*ds.Room, error) {
	room, err := s.r.CreateRoom(ctx, name)

	return room, err
}
