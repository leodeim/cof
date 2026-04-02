package cof

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

type testItem struct {
	id  int
	str string
}

func newTestCache(t *testing.T, opts ...Option) *C[testItem] {
	t.Helper()
	c, err := Init[testItem](opts...)
	require.NoError(t, err)
	t.Cleanup(func() { c.Stop() })
	return c
}

// ---------------------------------------------------------------------------
// Init / option validation
// ---------------------------------------------------------------------------

func TestInit_Defaults(t *testing.T) {
	c := newTestCache(t)
	assert.Equal(t, defaultCleanInterval, c.cleanInterval)
	assert.Equal(t, defaultTTL, c.ttl)
}

func TestInit_CustomOptions(t *testing.T) {
	c := newTestCache(t, TTL(5*time.Second), CleanInterval(2*time.Second))
	assert.Equal(t, 2*time.Second, c.cleanInterval)
	assert.Equal(t, 5*time.Second, c.ttl)
}

func TestInit_DisabledCleanup(t *testing.T) {
	c := newTestCache(t, CleanInterval(0))
	assert.Equal(t, time.Duration(0), c.cleanInterval)
}

func TestInit_DisabledTTL(t *testing.T) {
	c := newTestCache(t, TTL(0))
	assert.Equal(t, time.Duration(0), c.ttl)
}

func TestInit_NegativeTTL(t *testing.T) {
	_, err := Init[testItem](TTL(-1 * time.Second))
	assert.ErrorIs(t, err, ErrInvalidTTL)
}

func TestInit_NegativeCleanInterval(t *testing.T) {
	_, err := Init[testItem](CleanInterval(-1 * time.Second))
	assert.ErrorIs(t, err, ErrInvalidCleanInterval)
}

// ---------------------------------------------------------------------------
// Put / Get
// ---------------------------------------------------------------------------

func TestPutGet(t *testing.T) {
	c := newTestCache(t)

	// miss
	v, ok := c.Get("missing")
	assert.False(t, ok)
	assert.Empty(t, v)

	item1 := testItem{id: 1, str: "one"}
	c.Put("1", item1)

	v, ok = c.Get("1")
	assert.True(t, ok)
	assert.Equal(t, item1, v)
}

func TestPut_Overwrite(t *testing.T) {
	c := newTestCache(t)

	c.Put("k", testItem{id: 1, str: "first"})
	c.Put("k", testItem{id: 2, str: "second"})

	v, ok := c.Get("k")
	assert.True(t, ok)
	assert.Equal(t, 2, v.id)
}

// ---------------------------------------------------------------------------
// PutWithTTL
// ---------------------------------------------------------------------------

func TestPutWithTTL(t *testing.T) {
	c := newTestCache(t, CleanInterval(10*time.Millisecond))

	c.PutWithTTL("short", testItem{id: 1, str: "short"}, 200*time.Millisecond)
	c.PutWithTTL("long", testItem{id: 2, str: "long"}, 2*time.Second)

	// both present initially
	_, ok := c.Get("short")
	assert.True(t, ok)
	_, ok = c.Get("long")
	assert.True(t, ok)

	time.Sleep(350 * time.Millisecond)

	// short expired, long still alive
	_, ok = c.Get("short")
	assert.False(t, ok)
	_, ok = c.Get("long")
	assert.True(t, ok)
}

func TestPutWithTTL_ZeroMeansNoExpiry(t *testing.T) {
	c := newTestCache(t, TTL(100*time.Millisecond), CleanInterval(10*time.Millisecond))

	c.PutWithTTL("forever", testItem{id: 1, str: "forever"}, 0)

	time.Sleep(250 * time.Millisecond)

	v, ok := c.Get("forever")
	assert.True(t, ok)
	assert.Equal(t, "forever", v.str)
}

// ---------------------------------------------------------------------------
// Pop
// ---------------------------------------------------------------------------

func TestPop(t *testing.T) {
	c := newTestCache(t)

	c.Put("k", testItem{id: 1, str: "val"})

	v, ok := c.Pop("k")
	assert.True(t, ok)
	assert.Equal(t, 1, v.id)

	// second pop should miss
	v, ok = c.Pop("k")
	assert.False(t, ok)
	assert.Empty(t, v)
}

func TestPop_Expired(t *testing.T) {
	c := newTestCache(t, TTL(100*time.Millisecond), CleanInterval(0))

	c.Put("k", testItem{id: 1, str: "val"})
	time.Sleep(200 * time.Millisecond)

	v, ok := c.Pop("k")
	assert.False(t, ok)
	assert.Empty(t, v)
}

func TestPop_Missing(t *testing.T) {
	c := newTestCache(t)
	v, ok := c.Pop("nope")
	assert.False(t, ok)
	assert.Empty(t, v)
}

// ---------------------------------------------------------------------------
// Get returns false for expired items (bug fix verification)
// ---------------------------------------------------------------------------

func TestGet_Expired(t *testing.T) {
	// Disable automatic cleanup so the item stays in the map but is expired.
	c := newTestCache(t, TTL(100*time.Millisecond), CleanInterval(0))

	c.Put("k", testItem{id: 1, str: "val"})
	time.Sleep(200 * time.Millisecond)

	v, ok := c.Get("k")
	assert.False(t, ok, "Get must not return expired items")
	assert.Empty(t, v)
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestDelete(t *testing.T) {
	c := newTestCache(t)

	c.Put("k", testItem{id: 1, str: "val"})
	c.Delete("k")

	_, ok := c.Get("k")
	assert.False(t, ok)
}

func TestDelete_Missing(t *testing.T) {
	c := newTestCache(t)
	// should not panic
	c.Delete("nope")
}

// ---------------------------------------------------------------------------
// Has
// ---------------------------------------------------------------------------

func TestHas(t *testing.T) {
	c := newTestCache(t)

	assert.False(t, c.Has("k"))

	c.Put("k", testItem{id: 1, str: "val"})
	assert.True(t, c.Has("k"))
}

func TestHas_Expired(t *testing.T) {
	c := newTestCache(t, TTL(100*time.Millisecond), CleanInterval(0))

	c.Put("k", testItem{id: 1, str: "val"})
	time.Sleep(200 * time.Millisecond)

	assert.False(t, c.Has("k"), "Has must return false for expired items")
}

// ---------------------------------------------------------------------------
// Len
// ---------------------------------------------------------------------------

func TestLen(t *testing.T) {
	c := newTestCache(t)

	assert.Equal(t, 0, c.Len())

	c.Put("a", testItem{id: 1, str: "a"})
	c.Put("b", testItem{id: 2, str: "b"})
	assert.Equal(t, 2, c.Len())
}

func TestLen_ExcludesExpired(t *testing.T) {
	c := newTestCache(t, TTL(100*time.Millisecond), CleanInterval(0))

	c.Put("a", testItem{id: 1, str: "a"})
	c.PutWithTTL("b", testItem{id: 2, str: "b"}, 2*time.Second)

	time.Sleep(200 * time.Millisecond)

	assert.Equal(t, 1, c.Len(), "Len must exclude expired entries")
}

// ---------------------------------------------------------------------------
// Keys
// ---------------------------------------------------------------------------

func TestKeys(t *testing.T) {
	c := newTestCache(t)

	assert.Empty(t, c.Keys())

	c.Put("b", testItem{id: 2, str: "b"})
	c.Put("a", testItem{id: 1, str: "a"})

	keys := c.Keys()
	assert.Equal(t, []string{"a", "b"}, keys, "Keys must be sorted")
}

func TestKeys_ExcludesExpired(t *testing.T) {
	c := newTestCache(t, TTL(100*time.Millisecond), CleanInterval(0))

	c.Put("expire", testItem{id: 1, str: "e"})
	c.PutWithTTL("keep", testItem{id: 2, str: "k"}, 2*time.Second)

	time.Sleep(200 * time.Millisecond)

	assert.Equal(t, []string{"keep"}, c.Keys())
}

// ---------------------------------------------------------------------------
// Clear
// ---------------------------------------------------------------------------

func TestClear(t *testing.T) {
	c := newTestCache(t)

	c.Put("a", testItem{id: 1, str: "a"})
	c.Put("b", testItem{id: 2, str: "b"})

	c.Clear()
	assert.Equal(t, 0, c.Len())
	_, ok := c.Get("a")
	assert.False(t, ok)
}

// ---------------------------------------------------------------------------
// TTL expiration via cleaner
// ---------------------------------------------------------------------------

func TestTTL_CleanerRemovesExpired(t *testing.T) {
	c := newTestCache(t, TTL(200*time.Millisecond), CleanInterval(50*time.Millisecond))

	c.Put("1", testItem{id: 1, str: "one"})

	// present before expiration
	v, ok := c.Get("1")
	assert.True(t, ok)
	assert.Equal(t, 1, v.id)

	time.Sleep(400 * time.Millisecond)

	// gone after expiration + cleanup
	v, ok = c.Get("1")
	assert.False(t, ok)
	assert.Empty(t, v)
}

func TestTTL_ZeroMeansNoExpiry(t *testing.T) {
	c := newTestCache(t, TTL(0), CleanInterval(50*time.Millisecond))

	c.Put("k", testItem{id: 1, str: "val"})

	time.Sleep(200 * time.Millisecond)

	v, ok := c.Get("k")
	assert.True(t, ok)
	assert.Equal(t, "val", v.str)
}

// ---------------------------------------------------------------------------
// Stop
// ---------------------------------------------------------------------------

func TestStop_ClearsCache(t *testing.T) {
	c, err := Init[testItem]()
	require.NoError(t, err)

	c.Put("k", testItem{id: 1, str: "val"})
	c.Stop()

	// internal map should be empty after stop
	assert.Equal(t, 0, len(c.cache))
}

func TestStop_Idempotent(t *testing.T) {
	c, err := Init[testItem]()
	require.NoError(t, err)

	c.Stop()
	c.Stop() // must not panic or block
}

// ---------------------------------------------------------------------------
// Concurrency
// ---------------------------------------------------------------------------

func TestConcurrentAccess(t *testing.T) {
	c := newTestCache(t, TTL(1*time.Second), CleanInterval(50*time.Millisecond))

	const goroutines = 50
	const opsPerG = 200

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < opsPerG; i++ {
				key := "key"
				val := testItem{id: id*1000 + i, str: "val"}

				switch i % 7 {
				case 0:
					c.Put(key, val)
				case 1:
					c.PutWithTTL(key, val, 50*time.Millisecond)
				case 2:
					c.Get(key)
				case 3:
					c.Pop(key)
				case 4:
					c.Delete(key)
				case 5:
					c.Has(key)
				case 6:
					c.Len()
				}
			}
		}(g)
	}

	wg.Wait()
}
