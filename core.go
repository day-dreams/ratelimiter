package ratelimiter

import (
	"context"

	"github.com/go-redis/redis"
)

const (
	script string = `
local key = KEYS[1]
local rate=tonumber(ARGV[1])
local burst=tonumber(ARGV[2])
local count=tonumber(ARGV[3])
local expires=tonumber(ARGV[4])

redis.replicate_commands()
local t=redis.call('TIME')
local now = t[1] + t[2]/1000000
local exists=redis.call('EXISTS',key)
if exists == 0 then
    redis.call('hset',key,"rate",rate)
    redis.call('hset',key,"burst",burst)
    redis.call('hset',key,"lastBucket",0)
    -- 默认‘初次发放’是在10秒前
    redis.call('hset',key,"lastTime",now-10)
	redis.call('hset',key,'bucket',0)
end

redis.call('expire',key,expires)

local lastTime=redis.call('hget',key,"lastTime")
local lastBucket=redis.call('hget',key,"lastBucket")
local elasped=now - lastTime
local delta= elasped * rate 
local bucket= redis.call('hget',key,"bucket")

-- 桶里现在有几个ticket？
if ( delta + bucket  < burst) 
then
    bucket=delta+bucket
else
	bucket=burst
end

if ( count <= bucket )then
--	local msg="[yes]elasped:" .. elasped .. ",delta:" .. delta .. ",bucket:" .. bucket
--	redis.log(redis.LOG_WARNING, msg)
    lastBucket=lastBucket+count
    lastTime=now
	redis.call('hset',key,"lastBucket",lastBucket)
	redis.call('hset',key,"lastTime",lastTime)
	redis.call('hset',key,"bucket",bucket-count) -- 桶里还剩几个bucket
    return lastBucket
else
    -- 不能发
--	local msg="[no]elasped:" .. elasped .. ",delta:" .. delta .. ",bucket:" .. bucket
--	redis.log(redis.LOG_WARNING, msg)
    return -1 
end
`
)

// Limiter 令牌桶
type Limiter interface {
	// Get，获取n个ticket
	Get(ctx context.Context, key string, n int) (ok bool, err error)
}

func New(client *redis.Client, rate, burst, expire int) (Limiter, error) {

	script := redis.NewScript(script)

	l := &limiter{
		client: client,
		script: script,
		expire: expire,
		rate:   rate,
		burst:  burst,
	}

	return l, nil
}

type limiter struct {
	client *redis.Client
	script *redis.Script

	expire int

	rate  int
	burst int
}

func (l *limiter) Get(ctx context.Context, key string, n int) (ok bool, err error) {

	got, err := l.script.Run(l.client, []string{key}, l.rate, l.burst, n, l.expire).Int64()

	if err != nil {
		return false, err
	}
	return got != -1, nil
}
