package middleware

import (
	"net/http"
	"runtime"
	"sync"
	"time"

	"personal-blog-backend/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

// RateLimiter 基于 IP 的固定窗口限流器。
// 每个 IP 在每个窗口内最多允许 maxRequests 次请求。
type RateLimiter struct {
	mu          sync.Mutex
	windows     map[string]*windowState
	maxRequests int
	windowSize  time.Duration
	stopCh      chan struct{} // 通知 cleanupLoop 退出
}

type windowState struct {
	count    int
	resetAt  time.Time
}

// NewRateLimiter 创建一个限流器。
// maxRequests: 每个窗口允许的最大请求数。
// windowSize: 时间窗口大小。
func NewRateLimiter(maxRequests int, windowSize time.Duration) *RateLimiter {
	rl := &RateLimiter{
		windows:     make(map[string]*windowState),
		maxRequests: maxRequests,
		windowSize:  windowSize,
		stopCh:      make(chan struct{}),
	}
	// 后台清理过期窗口，避免内存泄漏
	go rl.cleanupLoop()
	// 设置 finalizer，确保 RateLimiter 被 GC 时 cleanupLoop 能退出
	runtime.SetFinalizer(rl, func(r *RateLimiter) {
		select {
		case <-r.stopCh:
			// 已关闭
		default:
			close(r.stopCh)
		}
	})
	return rl
}

// Middleware 返回 Gin 中间件函数
func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()

		rl.mu.Lock()
		now := time.Now()
		ws, exists := rl.windows[ip]

		if !exists || now.After(ws.resetAt) {
			// 新 IP 或窗口已过期：重置计数
			rl.windows[ip] = &windowState{
				count:   1,
				resetAt: now.Add(rl.windowSize),
			}
			rl.mu.Unlock()
			c.Next()
			return
		}

		// 窗口内计数递增
		ws.count++
		count := ws.count
		rl.mu.Unlock()

		if count > rl.maxRequests {
			response.Error(c, http.StatusTooManyRequests, "请求过于频繁，请稍后再试")
			c.Abort()
			return
		}

		c.Next()
	}
}

// cleanupLoop 定期清理过期窗口。
// 通过 stopCh 接收退出信号，避免 goroutine 泄漏。
func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.windowSize)
	defer ticker.Stop()
	for {
		select {
		case <-rl.stopCh:
			return
		case <-ticker.C:
			rl.mu.Lock()
			now := time.Now()
			for ip, ws := range rl.windows {
				if now.After(ws.resetAt) {
					delete(rl.windows, ip)
				}
			}
			rl.mu.Unlock()
		}
	}
}

// Stop 通知后台清理 goroutine 退出。
// 调用后 RateLimiter 不再可用（但已注册的中间件仍可安全执行）。
func (rl *RateLimiter) Stop() {
	close(rl.stopCh)
}
