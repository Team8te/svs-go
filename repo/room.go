package repo

import (
	"context"

	"github.com/Team8te/svs-go/ds"
	"github.com/Team8te/svs-go/pkg/utils/uid"
)

// CreateRoom ...
func (r *Repo) CreateRoom(_ context.Context, name string) (*ds.Room, error) {
	id := r.generateUniqueRoomID()
	room := &ds.Room{
		ID:   id,
		Name: name,
	}
	r.localCache.SetDefault(id.ToString(), room)
	r.localCache.SetDefault(name, room)
	return room, nil
}

// GetRoomByName ...
func (r *Repo) GetRoomByName(_ context.Context, name string) (*ds.Room, error) {
	if key, found := r.localCache.Get(name); found {
		res := *key.(*ds.Room) // make copy
		return &res, nil
	}
	return nil, ds.ErrorNotFound
}

// UpdateRoomByName ...
func (r *Repo) UpdateRoomByName(_ context.Context, room *ds.Room) error {
	if _, found := r.localCache.Get(room.Name); !found {
		return ds.ErrorNotFound

	}
	r.localCache.SetDefault(room.ID.ToString(), room)
	r.localCache.SetDefault(room.Name, room)
	return nil
}

// GetRoomByID ...
func (r *Repo) GetRoomByID(_ context.Context, id string) (*ds.Room, error) {
	if key, found := r.localCache.Get(id); found {
		res := *key.(*ds.Room) // make copy
		return &res, nil
	}
	return nil, ds.ErrorNotFound
}

// DeleteRoomByName ...
func (r *Repo) DeleteRoomByName(ctx context.Context, name string) error {
	room, err := r.GetRoomByName(ctx, name)
	if err != nil {
		return err
	}
	r.localCache.Delete(room.ID.ToString())
	r.localCache.Delete(room.Name)
	return nil
}

func (r *Repo) generateUniqueRoomID() ds.RoomID {
	for {
		key := uid.RandStringRunes(48)
		if _, found := r.localCache.Get(key); !found {
			return ds.RoomID(key)
		}
	}
}
