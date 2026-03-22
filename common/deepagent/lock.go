package deepagent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	commonredis "GopherAI/common/redis"
)

type localLock struct {
	locked bool
}

type lockManager struct {
	mu    sync.Mutex
	locks map[string]*localLock
}

func newLockManager() *lockManager {
	return &lockManager{
		locks: make(map[string]*localLock),
	}
}

func (m *lockManager) Acquire(ctx context.Context, key string, ttl time.Duration) (func(), error) {
	if commonredis.Rdb != nil {
		token := uuid.NewString()
		ok, err := commonredis.Rdb.SetNX(ctx, key, token, ttl).Result()
		if err == nil {
			if !ok {
				return nil, fmt.Errorf("lock busy")
			}
			return func() {
				const script = `
if redis.call("get", KEYS[1]) == ARGV[1] then
	return redis.call("del", KEYS[1])
end
return 0
`
				_, _ = commonredis.Rdb.Eval(context.Background(), script, []string{key}, token).Result()
			}, nil
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	lock, ok := m.locks[key]
	if !ok {
		lock = &localLock{}
		m.locks[key] = lock
	}
	if lock.locked {
		return nil, fmt.Errorf("lock busy")
	}

	lock.locked = true
	return func() {
		m.mu.Lock()
		defer m.mu.Unlock()
		if current, exists := m.locks[key]; exists {
			current.locked = false
		}
	}, nil
}
