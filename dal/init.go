package dal

import (
	"stock/dal/db"
	"stock/dal/redis"
)

func Init() {
	db.Init()
	redis.Init()
}
