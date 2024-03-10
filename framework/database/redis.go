package database

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"gitlab.zixel.cn/go/framework/config"
	"gitlab.zixel.cn/go/framework/xutil"
)

const (
	// 先get获取，如果有就刷新ttl，没有再set。 这种是可重入锁，防止在同一线程中多次获取锁而导致死锁发生。
	lockCommand = `if redis.call("GET", KEYS[1]) == ARGV[1] then
	     redis.call("SET", KEYS[1], ARGV[1], "PX", ARGV[2])
	     return "OK"
     else
	     return redis.call("SET", KEYS[1], ARGV[1], "NX", "PX", ARGV[2])
     end`

	// 删除。必须先匹配id值，防止A超时后，B马上获取到锁，A的解锁把B的锁删了
	delCommand = `if redis.call("GET", KEYS[1]) == ARGV[1] then
	     return redis.call("DEL", KEYS[1])
     else
	     return 0
     end`
	letters   = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	randomLen = 10
)

type RedisConnection struct {
	cli              *redis.UniversalClient
	config           *RedisConnConfig
	errContextCancel error
}
type RedisConnConfig struct {
	Addr          []string
	Password      string
	MasterName    string
	Db            int
	PoolSize      int
	MinIdelSize   int
	PoolTimeout   time.Duration
	Timeout       time.Duration
	RetryInterval time.Duration
}

type RedisLock struct {
	ctx       context.Context
	timeoutMs int
	key       string
	id        string
}

var connect RedisConnection

func getFullKey(key string) string {
	serviceName := config.GetString("server.name", "NoServiceName")
	return serviceName + ":" + key
}

func ConnectRedis(config RedisConnConfig) (err error) {
	uniConfig := &redis.UniversalOptions{
		Addrs:           config.Addr,     // redis地址
		Password:        config.Password, // redis密码，没有则留空
		DB:              config.Db,       // 默认数据库，默认是0
		DialTimeout:     config.Timeout,
		MaxRetryBackoff: config.RetryInterval,
		PoolSize:        config.PoolSize,
		MinIdleConns:    config.MinIdelSize,
		PoolTimeout:     config.PoolTimeout,
	}
	if config.MasterName != "" {
		uniConfig.MasterName = config.MasterName
	}
	ctx, cancel := context.WithTimeout(context.Background(), conn.timeout)
	defer cancel()
	cli := redis.NewUniversalClient(uniConfig)
	connect.cli = &cli
	_, err = (*connect.cli).Ping(ctx).Result()
	if err != nil {
		return err
	}
	connect.config = &config
	connect.errContextCancel = errors.New("context cancel")
	return
}

func RedisGetCmdable() redis.Cmdable {
	return *connect.cli
}

func RedisSet(key string, val any, timeout *time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), conn.timeout)
	defer cancel()
	err := RedisSetCtx(ctx, key, val, timeout)
	if err != nil {
		return err
	}
	return nil
}

func RedisSetCtx(ctx context.Context, key string, val any, timeout *time.Duration) error {
	var timeoutTime time.Duration
	if timeout != nil {
		timeoutTime = *timeout
	}
	err := (*connect.cli).Set(ctx, getFullKey(key), val, timeoutTime).Err()
	if err != nil {
		return err
	}
	return nil
}

func RedisGetSet(key string, val any) error {
	ctx, cancel := context.WithTimeout(context.Background(), conn.timeout)
	defer cancel()
	err := RedisGetSetCtx(ctx, key, val)
	if err != nil {
		return err
	}
	return nil
}

func RedisGetSetCtx(ctx context.Context, key string, val any) error {
	err := (*connect.cli).GetSet(ctx, getFullKey(key), val).Err()
	if err != nil {
		return err
	}
	return nil
}
func RedisGetInt64(key string) (val *int64, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), conn.timeout)
	defer cancel()
	val, err = RedisGetInt64Ctx(ctx, key)
	if err != nil {
		return
	}
	return
}

func RedisGetInt64Ctx(ctx context.Context, key string) (val *int64, err error) {
	ans, err := (*connect.cli).Get(ctx, getFullKey(key)).Int64()
	if err == redis.Nil {
		val = nil
		return
	} else if err != nil {
		return
	}
	val = &ans
	return
}
func RedisGetInt(key string) (val *int, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), conn.timeout)
	defer cancel()
	val, err = RedisGetIntCtx(ctx, key)
	if err != nil {
		return
	}
	return
}

func RedisGetIntCtx(ctx context.Context, key string) (val *int, err error) {
	ans, err := (*connect.cli).Get(ctx, getFullKey(key)).Int()
	if err == redis.Nil {
		val = nil
		return
	} else if err != nil {
		return
	}
	val = &ans
	return
}
func RedisGetUInt64(key string) (val *uint64, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), conn.timeout)
	defer cancel()
	val, err = RedisGetUInt64Ctx(ctx, key)
	if err != nil {
		return
	}
	return
}

func RedisGetUInt64Ctx(ctx context.Context, key string) (val *uint64, err error) {
	ans, err := (*connect.cli).Get(ctx, getFullKey(key)).Uint64()
	if err == redis.Nil {
		val = nil
		return
	} else if err != nil {
		return
	}
	val = &ans
	return
}
func RedisGetBytes(key string) (val *[]byte, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), conn.timeout)
	defer cancel()
	val, err = RedisGetBytesCtx(ctx, key)
	if err != nil {
		return
	}
	return
}

func RedisGetBytesCtx(ctx context.Context, key string) (val *[]byte, err error) {
	ans, err := (*connect.cli).Get(ctx, getFullKey(key)).Bytes()
	if err == redis.Nil {
		val = nil
		return
	} else if err != nil {
		return
	}
	val = &ans
	return
}
func RedisGetFloate64(key string) (val *float64, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), conn.timeout)
	defer cancel()
	val, err = RedisGetFloate64Ctx(ctx, key)
	if err != nil {
		return
	}
	return
}

func RedisGetFloate64Ctx(ctx context.Context, key string) (val *float64, err error) {
	ans, err := (*connect.cli).Get(ctx, getFullKey(key)).Float64()
	if err == redis.Nil {
		val = nil
		return
	} else if err != nil {
		return
	}
	val = &ans
	return
}
func RedisGetFloate32(key string) (val *float32, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), conn.timeout)
	defer cancel()
	val, err = RedisGetFloate32Ctx(ctx, key)
	if err != nil {
		return
	}
	return
}

func RedisGetFloate32Ctx(ctx context.Context, key string) (val *float32, err error) {
	ans, err := (*connect.cli).Get(ctx, getFullKey(key)).Float32()
	if err == redis.Nil {
		val = nil
		return
	} else if err != nil {
		return
	}
	val = &ans
	return
}

func RedisGetString(key string) (val *string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), conn.timeout)
	defer cancel()
	val, err = RedisGetStringCtx(ctx, key)
	if err != nil {
		return
	}
	return
}

func RedisGetStringCtx(ctx context.Context, key string) (val *string, err error) {
	ans, err := (*connect.cli).Get(ctx, getFullKey(key)).Result()
	if err == redis.Nil {
		val = nil
		return
	} else if err != nil {
		return
	}
	val = &ans
	return
}
func RedisSetMap(key string, val map[string]any, timeout *time.Duration) (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), conn.timeout)
	defer cancel()
	err = RedisSetMapCtx(ctx, key, val, timeout)
	if err != nil {
		return
	}
	return
}

func RedisSetMapCtx(ctx context.Context, key string, val map[string]any, timeout *time.Duration) (err error) {
	err = (*connect.cli).HMSet(ctx, getFullKey(key), val).Err()
	if err != nil {
		return
	}
	if timeout != nil {
		(*connect.cli).PExpire(ctx, getFullKey(key), *timeout).Err()
	}
	return
}
func RedisGetMap(key string) (val map[string]string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), conn.timeout)
	defer cancel()
	val, err = RedisGetMapCtx(ctx, key)
	if err != nil {
		return
	}
	return
}

func RedisGetMapCtx(ctx context.Context, key string) (val map[string]string, err error) {
	val, err = (*connect.cli).HGetAll(ctx, getFullKey(key)).Result()
	if err == redis.Nil {
		val = nil
		return
	} else if err != nil {
		return
	}
	return
}

func RedisGetStruct(key string, result *any) (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), conn.timeout)
	defer cancel()
	err = RedisGetStructCtx(ctx, key, result)
	if err != nil {
		return
	}
	return
}

func RedisGetStructCtx(ctx context.Context, key string, result *any) (err error) {
	val := (*connect.cli).HGetAll(ctx, getFullKey(key))
	if err = val.Scan(result); err != nil {
		return
	}
	return
}
func RedisMultiGet(key []string) (ans []any, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), conn.timeout)
	defer cancel()
	ans, err = RedisMultiGetCtx(ctx, key)
	if err != nil {
		return
	}
	return
}

func RedisMultiGetCtx(ctx context.Context, key []string) (ans []any, err error) {
	keys := make([]string, 0, len(key))
	for _, v := range key {
		key = append(key, getFullKey(v))
	}
	ans, err = (*connect.cli).MGet(ctx, keys...).Result()
	return
}
func RedisMultiSet(val map[string]any, timeout *time.Duration) (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), conn.timeout)
	defer cancel()
	err = RedisMultiSetCtx(ctx, val, timeout)
	if err != nil {
		return
	}
	return
}

func RedisMultiSetCtx(ctx context.Context, val map[string]any, timeout *time.Duration) (err error) {
	keys := make(map[string]any, len(val))
	for k, v := range val {
		keys[getFullKey(k)] = v
	}
	err = (*connect.cli).MSet(ctx, keys).Err()
	return
}
func RedisMultiDelCtx(ctx context.Context, keys []string) (err error) {
	keyset := make([]string, 0, len(keys))
	for _, v := range keys {
		keyset = append(keyset, getFullKey(v))
	}
	err = (*connect.cli).Del(ctx, keyset...).Err()
	return
}
func RedisMultiDel(keys []string) (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), conn.timeout)
	defer cancel()
	err = RedisMultiDelCtx(ctx, keys)
	if err != nil {
		return
	}
	return
}

// 缓存相关
func NewRedisCacheCtx(ctx context.Context, name string, key string, val any, timeout time.Duration) (err error) {
	cacheKey := name + "::" + key
	seriVal, err := json.Marshal(&val)
	if err != nil {
		return
	}
	return RedisSetCtx(ctx, cacheKey, seriVal, &timeout)
}
func NewRedisCache(name string, key string, val any, timeout time.Duration) (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), conn.timeout)
	defer cancel()
	err = NewRedisCacheCtx(ctx, name, key, val, timeout)
	return
}
func NewRedisCachesCtx(ctx context.Context, name string, val map[string]any, timeout time.Duration) error {
	cacheKeys := make(map[string]any, len(val))
	for k, v := range val {
		cacheKeys[name+"::"+k] = v
	}
	return RedisMultiSetCtx(ctx, cacheKeys, &timeout)
}
func NewRedisCaches(name string, val map[string]any, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), conn.timeout)
	defer cancel()
	return NewRedisCachesCtx(ctx, name, val, timeout)
}
func GetRedisCache[T any](name string, key string) (ans *T, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), conn.timeout)
	defer cancel()
	return GetRedisCacheCtx[T](ctx, name, key)
}
func GetRedisCacheCtx[T any](ctx context.Context, name string, key string) (ans *T, err error) {
	cacheKey := name + "::" + key
	res, err := RedisGetBytesCtx(ctx, cacheKey)
	if err == nil && res != nil {
		tmp := new(T)
		err = json.Unmarshal(*res, tmp)

		if err != nil {
			return nil, err
		}
		ans = tmp
	}
	return
}
func GetRedisCaches[T any](name string, keys []string) (ans *[]T, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), conn.timeout)
	defer cancel()
	return GetRedisCachesCtx[T](ctx, name, keys)
}
func GetRedisCachesCtx[T any](ctx context.Context, name string, keys []string) (ans *[]T, err error) {
	cacheKeys := make([]string, 0, len(keys))
	for _, v := range keys {
		cacheKeys = append(cacheKeys, name+"::"+v)
	}
	tmpAns := make([]T, 0)
	res, err := RedisMultiGetCtx(ctx, cacheKeys)
	if err == nil {
		for _, v := range res {
			b, ok := v.([]byte)
			if !ok {
				err = errors.New("cahche parse failed")
				return
			}
			tmp := new(T)
			err = json.Unmarshal(b, tmp)
			if err != nil {
				return nil, err
			}
			tmpAns = append(tmpAns, *tmp)
		}
	}
	ans = &tmpAns
	return
}
func RemoveRedisCaches(name string, keys []string) (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), conn.timeout)
	defer cancel()
	return RemoveRedisCachesCtx(ctx, name, keys)
}
func RemoveRedisCachesCtx(ctx context.Context, name string, keys []string) (err error) {
	cacheKeys := make([]string, 0, len(keys))
	for _, v := range keys {
		cacheKeys = append(cacheKeys, name+"::"+v)
	}
	err = RedisMultiDelCtx(ctx, cacheKeys)
	return
}

// lock相关
func NewRedisLock(ctx context.Context, key string) *RedisLock {
	timeout := connect.config.Timeout
	if deadline, ok := ctx.Deadline(); ok {
		timeout = time.Until(deadline)
	}
	id, err := xutil.GetSnowflakeString()
	if err != nil {
		id = randomStr(randomLen)
	}
	rl := &RedisLock{
		ctx:       ctx,
		timeoutMs: int(timeout.Milliseconds()),
		key:       getFullKey(key),
		id:        id,
	}

	return rl
}

func (rl *RedisLock) TryLock(sec time.Duration) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), conn.timeout)
	defer cancel()
	t := strconv.Itoa(int(sec.Abs().Milliseconds()))
	resp, err := (*connect.cli).Eval(ctx, lockCommand, []string{rl.key}, []string{rl.id, t}).Result()
	if err != nil || resp == nil {
		return false, nil
	}

	reply, ok := resp.(string)
	return ok && reply == "OK", nil
}

func (rl *RedisLock) LockRetry(sec time.Duration) error {
	for {
		select {
		case <-rl.ctx.Done():
			return connect.errContextCancel
		default:
			b, err := rl.TryLock(sec)
			if err != nil {
				return err
			}
			if b {
				return nil
			}
			time.Sleep(connect.config.RetryInterval)
		}
	}
}
func (rl *RedisLock) Lock(sec time.Duration) error {
	select {
	case <-rl.ctx.Done():
		return connect.errContextCancel
	default:
		b, err := rl.TryLock(sec)
		if err != nil {
			return err
		}
		if !b {
			return errors.New("")
		}
	}
	return nil
}
func (rl *RedisLock) Unlock() {
	ctx, cancel := context.WithTimeout(context.Background(), conn.timeout)
	defer cancel()
	(*connect.cli).Eval(ctx, delCommand, []string{rl.key}, []string{rl.id}).Result()
}

func randomStr(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
