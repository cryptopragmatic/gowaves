package api

import (
	"net/http"

	"go.uber.org/zap"
)

type RunOptions struct {
	RateLimiterOpts      *RateLimiterOptions
	LogHttpRequestOpts   bool
	CollectMetrics       bool
	UseRealIPMiddleware  bool
	EnableHeartbeatRoute bool
	RouteNotFoundHandler func(w http.ResponseWriter, r *http.Request)
}

type RateLimiterOptions struct {
	MemoryCacheSize      int
	MaxRequestsPerSecond int
	MaxBurst             int
}

func DefaultRunOptions() *RunOptions {
	return &RunOptions{
		RateLimiterOpts:      nil,
		LogHttpRequestOpts:   false,
		EnableHeartbeatRoute: true,
		UseRealIPMiddleware:  true,
		CollectMetrics:       true,
		RouteNotFoundHandler: func(w http.ResponseWriter, r *http.Request) {
			zap.S().Debugf("NodeApi not found %+v, %s", r, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		},
	}
}
