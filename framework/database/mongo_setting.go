package database

import (
	"sync"
	"sync/atomic"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

var id_next int64 = 0
var id_last int64 = 0
var id_allocate_count int64 = 10
var id_lock sync.Mutex

type Mongo_Setting_t struct {
	Key string `bson:"key"`
	Val any    `bson:"value"`
}

var Mongo_Setting = NewCollection[Mongo_Setting_t]("setting")

func generateIds(key string, count int64) int64 {
	res, err := Mongo_Setting.FindOneAndUpdate(
		bson.M{"key": key},
		bson.M{"$inc": bson.M{
			"value": count,
		}},
		"value",
	)

	if err != nil {
		return -1
	}

	kv := res.(Mongo_Setting_t)
	switch val := (kv.Val).(type) {
	case int:
		return int64(val) + count
	case int32:
		return int64(val) + count
	case int64:
		return val + count
	default:
		return -1
	}
}

func Mongo_NewIdAllocator(generatorName string) (g func() int64, err error) {
	_, err = Mongo_Setting.FindOne(bson.M{"key": generatorName})
	if err == mongo.ErrNoDocuments {
		err = Mongo_Setting.Insert(bson.M{"key": generatorName, "value": 10000000})
		if err != nil {
			return
		}
	}

	if err != nil {
		return
	}

	g = func() int64 {
		id_lock.Lock()
		defer id_lock.Unlock()

		if id_next >= id_last {
			id_last = generateIds(generatorName, id_allocate_count)

			if id_next+id_allocate_count != id_last {
				id_next = id_last - id_allocate_count
			}
		}

		return atomic.AddInt64(&id_next, 1)
	}

	return
}
