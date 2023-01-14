package authenticator

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
)

type Auth struct {
	context     context.Context
	address     string
	username    string
	password    string
	Client      RedisSwitcher
	initialized bool
}

func (a *Auth) Initialize(ctx context.Context, clientType string, address string, username string, password string, privateKey string) error {
	a.context = ctx
	a.address = address
	a.username = username
	a.password = password
	var switcher RedisSwitcher
	a.Client = switcher
	a.Client.Type = clientType
	a.setupRedisClient()
	a.initialized = true
	return nil
}

func (a *Auth) Authenticate(access_token string) string {
	access_array := strings.Split(access_token, ".")
	data, _ := base64.RawStdEncoding.DecodeString(access_array[1])
	var tokenData odinTokenData
	err := json.Unmarshal(data, &tokenData)
	if err != nil {
		return ""
	}
	clientCode := tokenData.MemberInfo.ClientCode
	presence, err := a.Client.Get(a.context, "at_"+clientCode+":"+access_token).Bool()
	if err != nil {
		return ""
	}
	if presence {
		return clientCode
	} else {
		return ""
	}
}

type odinTokenData struct {
	MemberInfo struct {
		ClientCode string `json:"userId" binding:"required"`
	} `json:"memberInfo" binding:"required"`
}

type RedisSwitcher struct {
	RedisDb        *redis.Client
	RedisClusterDb *redis.ClusterClient
	Type           string
}

func (a *Auth) setupRedisClient() {
	if a.Client.Type == "single" {
		a.Client.RedisDb = redis.NewClient(&redis.Options{
			Addr:     a.address, //"3.110.49.227:6379", //"localhost:6379",
			Username: a.username,
			Password: a.password,
			// Addr:     "3.110.49.227:6379", //"localhost:6379",
			// Username: "",
			// Password: "flowbackend",
		})
		err := a.Client.RedisDb.Ping(a.context).Err()
		if err != nil {
			panic(err)
		}
	} else {
		a.Client.Type = "cluster"
		a.Client.RedisClusterDb = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:    []string{a.address},
			Username: a.username,
			Password: a.password,
			TLSConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		})
		err := a.Client.RedisClusterDb.ForEachShard(a.context, func(ctx context.Context, shard *redis.Client) error {
			return shard.Ping(ctx).Err()
		})
		if err != nil {
			panic(err)
		}
	}
	fmt.Println("[INFO] RedisDB Connected")
}

func (sw *RedisSwitcher) Keys(ctx context.Context, pattern string) *redis.StringSliceCmd {
	if sw.Type == "cluster" {
		return sw.RedisClusterDb.Keys(ctx, pattern)
	} else {
		return sw.RedisDb.Keys(ctx, pattern)
	}
}
func (sw *RedisSwitcher) Get(ctx context.Context, key string) *redis.StringCmd {
	if sw.Type == "cluster" {
		return sw.RedisClusterDb.Get(ctx, key)
	} else {
		return sw.RedisDb.Get(ctx, key)
	}
}
func (sw *RedisSwitcher) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	if sw.Type == "cluster" {
		return sw.RedisClusterDb.Set(ctx, key, value, expiration)
	} else {
		return sw.RedisDb.Set(ctx, key, value, expiration)
	}
}
func (sw *RedisSwitcher) Del(ctx context.Context, key string) *redis.IntCmd {
	if sw.Type == "cluster" {
		return sw.RedisClusterDb.Del(ctx, key)
	} else {
		return sw.RedisDb.Del(ctx, key)
	}
}

func (sw *RedisSwitcher) Pipeline() redis.Pipeliner {
	if sw.Type == "cluster" {
		return sw.RedisClusterDb.Pipeline()
	} else {
		return sw.RedisDb.Pipeline()
	}
}
func (sw *RedisSwitcher) Close() error {
	if sw.Type == "cluster" {
		return sw.RedisClusterDb.Close()
	} else {
		return sw.RedisDb.Close()
	}
}
func (sw *RedisSwitcher) Incr(ctx context.Context, key string) *redis.IntCmd {
	if sw.Type == "cluster" {
		return sw.RedisClusterDb.Incr(ctx, key)
	} else {
		return sw.RedisDb.Incr(ctx, key)
	}
}

func (sw *RedisSwitcher) ExpireAt(ctx context.Context, key string, time time.Time) *redis.BoolCmd {
	if sw.Type == "cluster" {
		return sw.RedisClusterDb.ExpireAt(ctx, key, time)
	} else {
		return sw.RedisDb.ExpireAt(ctx, key, time)
	}
}

// var RedisDB *redis.ClusterClient

// func SetupRedisClient() *redis.ClusterClient {
// 	// db := os.Getenv("REDIS_DB")	// // dbi, _ := strconv.Atoi(db)

// 	err := RedisDB.ForEachShard(ctx, func(ctx context.Context, shard *redis.Client) error {
// 		return shard.Ping(ctx).Err()
// 	})
// 	if err != nil {
// 		panic(err)
// 	}
// 	return RedisDB
// }
