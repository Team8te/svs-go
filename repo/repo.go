package repo

import (
	"github.com/patrickmn/go-cache"
)

type Repo struct {
	localCache *cache.Cache
}

func NewRepo() *Repo {
	return &Repo{
		localCache: cache.New(cache.NoExpiration, 0),
	}
}
