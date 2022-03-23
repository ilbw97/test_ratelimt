package main

import (
	"context"
	"log"
	"os"
	"sort"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// type APIConnection struct {
// 	rateLimiter *rate.Limiter
// }

// func Open() *APIConnection {
// 	return &APIConnection{
// 		// 1초당 1개의 이벤트라는 속도 제한을 설정
// 		rateLimiter: rate.NewLimiter(rate.Limit(1), 1),
// 	}
// }

// func (a *APIConnection) ReadFile(ctx context.Context) error {
// 	// 속도 제한에 의해 요청을 완료하기에 충분한 토큰을 가질 때까지 대기
// 	if err := a.rateLimiter.Wait(ctx); err != nil {
// 		return err
// 	}
// 	// Pretend we do work here
// 	return nil
// }

// func (a *APIConnection) ResolveAddress(ctx context.Context) error {
// 	if err := a.rateLimiter.Wait(ctx); err != nil {
// 		return err
// 	}

// 	// Pretend we do work here
// 	return nil
// }

// RateLimiter 인터페이스를 정의
type RateLimiter interface {
	Wait(context.Context) error
	Limit() rate.Limit
}

func MultiLimiter(limiters ...RateLimiter) *multiLimiter {
	byLimit := func(i, j int) bool {
		return limiters[i].Limit() < limiters[j].Limit()
	}
	// 최적화를 위하여 Limit() 값으로 정렬 (오름차순)
	sort.Slice(limiters, byLimit)
	return &multiLimiter{limiters: limiters}
}

type multiLimiter struct {
	limiters []RateLimiter
}

func (l *multiLimiter) Wait(ctx context.Context) error {
	// 모든 Limiter로부터 토큰을 획득해야 실행 가능
	for _, l := range l.limiters {
		if err := l.Wait(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (l *multiLimiter) Limit() rate.Limit {
	// 가장 한정적인 Limit() 값을 반환 (분당 속도 제한기를 반환)
	// 이 예제에서 실제 호출되지 않음
	return l.limiters[0].Limit()
}

// 단위 시간당 요청수를 좀 더 직관적으로 표현하기 위해서 도우미 함수를 정의
func Per(eventCount int, duration time.Duration) rate.Limit {
	return rate.Every(duration / time.Duration(eventCount))
}

func Open() *APIConnection {
	// 초당 최대 2번의 요청을 허용하는 속도 제한 (Limit() = 2)
	secondLimit := rate.NewLimiter(Per(2, time.Second), 1)
	// 6초에 1번 재충전되고, 최대 10번의 요청을 동시에 허용하는 속도 제한 (Limit() = 0.16666666666666666)
	minuteLimit := rate.NewLimiter(Per(10, time.Minute), 10)
	return &APIConnection{
		rateLimiter: MultiLimiter(secondLimit, minuteLimit),
	}
}

type APIConnection struct {
	rateLimiter RateLimiter
}

func (a *APIConnection) ReadFile(ctx context.Context) error {
	if err := a.rateLimiter.Wait(ctx); err != nil {
		return err
	}
	// Pretend we do work here
	return nil
}

func (a *APIConnection) ResolveAddress(ctx context.Context) error {
	if err := a.rateLimiter.Wait(ctx); err != nil {
		return err
	}
	// Pretend we do work here
	return nil
}

func main() {
	defer log.Printf("Done.")
	log.SetOutput(os.Stdout)
	log.SetFlags(log.Ltime)

	apiConnection := Open()
	var wg sync.WaitGroup
	wg.Add(20)

	for i := 0; i < 10; i++ {
		go func() {
			defer wg.Done()
			err := apiConnection.ReadFile(context.Background())
			if err != nil {
				log.Printf("cannot ReadFile: %v", err)
			}
			log.Printf("ReadFile")
		}()
	}

	for i := 0; i < 10; i++ {
		go func() {
			defer wg.Done()
			err := apiConnection.ResolveAddress(context.Background())
			if err != nil {
				log.Printf("cannot ResolveAddress: %v", err)
			}
			log.Printf("ResolveAddress")
		}()
	}

	wg.Wait()
}
