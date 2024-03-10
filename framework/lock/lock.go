package lock

import (
	"context"
	"math/rand"
	"time"

	redisV8 "github.com/go-redis/redis/v8"
)

const (
	letters    = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	delCommand = `if redis.call("GET", KEYS[1]) == ARGV[1] then
    return redis.call("DEL", KEYS[1])
else
    return 0
end`
)

type RedisLock struct {
	// redis客户端
	store *redisV8.Client
}

/*
*
初始化RedisLock
*/
func NewRedisLock(store *redisV8.Client) *RedisLock {
	return &RedisLock{
		store: store,
	}
}

/*
*
获取随机字符串
*/
func (redislock *RedisLock) RandomStr(n int) string {
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

/*
*
加锁
vaule:每次请求的唯一值，避免一个客户端释放了其它客户端持有的锁
*/
func (redislock *RedisLock) Lock(ctx context.Context, lockKey, vaule string, sec time.Duration) (bool, error) {
	return redislock.store.SetNX(ctx, lockKey, vaule, sec).Result()
}

/*
*
解锁
vaule:和加锁时vaule一致
*/
func (redislock *RedisLock) Unlock(ctx context.Context, lockKey, vaule string) (bool, error) {
	rsp, err := redislock.store.Eval(ctx, delCommand, []string{lockKey}, vaule).Result()
	if err != nil {
		return false, err
	}
	num := rsp.(int64)
	if num <= 0 {
		return false, err
	}
	return true, nil
}

func (redislock *RedisLock) TryLock(ctx context.Context, lockKey, vaule string, sec time.Duration, f func() error) (bool, error) {
	var (
		flag bool
		err  error
	)
	defer func() {
		redislock.store.Eval(ctx, delCommand, []string{lockKey}, vaule)
	}()

	flag, err = redislock.store.SetNX(ctx, lockKey, vaule, sec).Result()
	if !flag || err != nil {
		return false, err
	}
	if err = f(); err != nil {
		return true, err
	}
	return true, nil
}
