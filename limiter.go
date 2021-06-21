package limiter

import (
	"context"
	"github.com/go-redis/redis/v8"
	"github.com/go-redis/redis_rate/v9"
	"net/http"
	"encoding/json"
)

const (
	REDIS_SERVER = "limiter_cache:6379"
)

type LimiterError struct {
	Message string `json:"error"`
}

func LimiterWriteError(w http.ResponseWriter, m *LimiterError) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusUnauthorized)

	if err := json.NewEncoder(w).Encode(m); err != nil {
		panic(err)
	}
}

var LimiterMiddleware = func(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.Background()
		rdb := redis.NewClient(&redis.Options{
			Addr: REDIS_SERVER,
		})
		
		limiter := redis_rate.NewLimiter(rdb)
		// NOTE: For testing purposes only
		res, err := limiter.Allow(ctx, "healthcheck:testing_key", redis_rate.PerSecond(10))
		
		if err != nil {
			panic(err)
		}

		if res.Allowed == 0 {
			LimiterWriteError(w, &LimiterError{Message: "API maximum rate limit reached. Request Rate Limit: 10 requests per second."})
			return
		}

		next.ServeHTTP(w, r)
		return
	})
}