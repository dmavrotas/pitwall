package ai

import (
	"context"
	"strings"
	"sync"
)

// Cache wraps a Translator with an in-memory cache keyed by normalized question.
// Identical questions in one process won't pay the API call twice.
type Cache struct {
	inner Translator
	mu    sync.RWMutex
	data  map[string]Result
}

// NewCache wraps inner. Pass nil to disable caching (the wrapper still works
// but every call hits inner).
func NewCache(inner Translator) *Cache {
	return &Cache{inner: inner, data: make(map[string]Result)}
}

// Translate looks up the question in the cache; on miss it delegates to the
// wrapped translator and stores a successful result.
func (c *Cache) Translate(ctx context.Context, question string) (Result, error) {
	key := normalize(question)

	c.mu.RLock()
	hit, ok := c.data[key]
	c.mu.RUnlock()
	if ok {
		return hit, nil
	}

	res, err := c.inner.Translate(ctx, question)
	if err != nil {
		return Result{}, err
	}

	c.mu.Lock()
	c.data[key] = res
	c.mu.Unlock()
	return res, nil
}

func normalize(q string) string {
	return strings.Join(strings.Fields(strings.ToLower(q)), " ")
}
