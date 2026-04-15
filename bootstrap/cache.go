package bootstrap

import "github.com/CuratorC/gocanned/cache"

func SetupRedis() (err error) {
	return cache.SetupRedis("redis")
}
