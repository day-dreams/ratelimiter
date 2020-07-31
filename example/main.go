package main

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/go-redis/redis"

	"github.com/day-dreams/ratelimiter"
)

// 100个goroutine去获取ticket，每秒打印一次获取情况

func main() {
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
	limiter, err := ratelimiter.New(client, 50, 50, 10)
	if err != nil {
		panic(err)
	}

	count := int64(0)
	for i := 0; i != 100; i++ {
		go func() {

			ticker := time.NewTicker(time.Second)

			for range ticker.C {
				cnt := 1

				ok, err := limiter.Get(context.TODO(), "YourUserID", cnt)
				if err != nil {
					fmt.Printf("limiter.Get failed. %v\n", err)
					continue
				}

				if ok {
					atomic.AddInt64(&count, int64(cnt))
				}
			}
		}()
	}

	time.Sleep(time.Second)
	fmt.Printf("inspect in redis: hgetall YourUserID\n")
	begin := time.Now()
	ticker := time.NewTicker(time.Second)
	last := int64(0)
	for now := range ticker.C {
		total := atomic.LoadInt64(&count)
		cost := now.Sub(begin).Seconds()
		fmt.Printf("got ticket:%v(total),%v(last second),  average:%f/s.\n", total, total-last, float64(total)/cost)
		last = total
	}

}
