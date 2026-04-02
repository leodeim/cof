// Package cof provides a lightweight, generic, thread-safe in-memory
// key-value cache with TTL-based expiration and automatic cleanup.
package cof

import (
	"errors"
	"maps"
	"slices"
	"sync"
	"time"
)

const (
	// OFF disables time-based behaviour (TTL or cleanup interval).
	OFF = 0

	defaultCleanInterval = 1 * time.Minute
	defaultTTL           = 15 * time.Minute
)

// Errors returned by Init when the supplied options are invalid.
var (
	ErrInvalidTTL           = errors.New("cof: TTL must be non-negative")
	ErrInvalidCleanInterval = errors.New("cof: clean interval must be non-negative")
)

// C is a generic, thread-safe in-memory cache whose values are of type T.
// Create one with [Init] and stop it with [Stop] when it is no longer needed.
type C[T any] struct {
	mu    sync.RWMutex
	cache map[string]item[T]
	options
	stop chan struct{}
}

type item[T any] struct {
	value     T
	expiresOn int64
}

func (i *item[T]) isExpired(now int64) bool {
	return i.expiresOn > OFF && i.expiresOn <= now
}

type options struct {
	cleanInterval time.Duration
	ttl           time.Duration
}

// Option configures the cache created by [Init].
type Option func(o *options)

// CleanInterval sets how often the background goroutine removes expired
// entries. Pass 0 to disable automatic cleanup. Default: 1 minute.
func CleanInterval(ci time.Duration) Option {
	return func(o *options) {
		o.cleanInterval = ci
	}
}

// TTL sets the default time-to-live for every entry written with [C.Put].
// Pass 0 to keep entries forever (no expiration). Default: 15 minutes.
func TTL(ttl time.Duration) Option {
	return func(o *options) {
		o.ttl = ttl
	}
}

// Init creates a new cache and starts a background cleanup goroutine
// (unless the clean interval is set to [OFF]).
// Call [C.Stop] to release resources when the cache is no longer needed.
func Init[T any](opts ...Option) (*C[T], error) {
	o := options{
		cleanInterval: defaultCleanInterval,
		ttl:           defaultTTL,
	}

	for _, opt := range opts {
		opt(&o)
	}

	if o.ttl < 0 {
		return nil, ErrInvalidTTL
	}
	if o.cleanInterval < 0 {
		return nil, ErrInvalidCleanInterval
	}

	c := &C[T]{
		cache:   make(map[string]item[T]),
		stop:    make(chan struct{}),
		options: o,
	}

	go c.cleaner()

	return c, nil
}

// Put inserts or updates a key with the cache's default TTL.
func (c *C[T]) Put(k string, v T) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache[k] = c.newItem(v, c.ttl)
}

// PutWithTTL inserts or updates a key with a custom TTL that overrides the
// cache default. Pass 0 to store the item without expiration.
func (c *C[T]) PutWithTTL(k string, v T, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache[k] = c.newItem(v, ttl)
}

func (c *C[T]) newItem(v T, ttl time.Duration) item[T] {
	var exp int64
	if ttl > 0 {
		exp = time.Now().Add(ttl).UnixMilli()
	}
	return item[T]{value: v, expiresOn: exp}
}

// Pop retrieves the value for key k, removes the entry, and returns (value, true).
// If the key does not exist or is expired it returns the zero value and false.
func (c *C[T]) Pop(k string) (T, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	v, ok := c.cache[k]
	if !ok {
		var zero T
		return zero, false
	}

	delete(c.cache, k)

	if v.isExpired(time.Now().UnixMilli()) {
		var zero T
		return zero, false
	}

	return v.value, true
}

// Get retrieves the value for key k and returns (value, true).
// If the key does not exist or is expired it returns the zero value and false.
func (c *C[T]) Get(k string) (T, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	v, ok := c.cache[k]
	if !ok || v.isExpired(time.Now().UnixMilli()) {
		var zero T
		return zero, false
	}
	return v.value, true
}

// Delete removes the entry for key k. It is a no-op if the key does not exist.
func (c *C[T]) Delete(k string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.cache, k)
}

// Has reports whether the cache contains a live (non-expired) entry for key k.
func (c *C[T]) Has(k string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	v, ok := c.cache[k]
	return ok && !v.isExpired(time.Now().UnixMilli())
}

// Len returns the number of live (non-expired) entries currently in the cache.
// Note: this is O(n) because it must skip expired-but-not-yet-cleaned entries.
func (c *C[T]) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	now := time.Now().UnixMilli()
	n := 0
	for _, v := range c.cache {
		if !v.isExpired(now) {
			n++
		}
	}
	return n
}

// Keys returns the keys of all live (non-expired) entries.
// The order is non-deterministic.
func (c *C[T]) Keys() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	now := time.Now().UnixMilli()
	keys := make([]string, 0, len(c.cache))
	for k, v := range c.cache {
		if !v.isExpired(now) {
			keys = append(keys, k)
		}
	}
	slices.Sort(keys)
	return keys
}

// Clear removes all entries from the cache without stopping the cleanup
// goroutine.
func (c *C[T]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	clear(c.cache)
}

// Stop halts the background cleanup goroutine and removes all entries.
// After Stop is called the cache must not be reused.
func (c *C[T]) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	select {
	case c.stop <- struct{}{}:
	default:
	}

	clear(c.cache)
}

func (c *C[T]) cleaner() {
	if c.cleanInterval <= OFF {
		return
	}

	ticker := time.NewTicker(c.cleanInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.stop:
			return
		case <-ticker.C:
			c.cleanup()
		}
	}
}

func (c *C[T]) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now().UnixMilli()
	maps.DeleteFunc(c.cache, func(_ string, v item[T]) bool {
		return v.isExpired(now)
	})
}
