package cof

import (
	"fmt"
	"testing"
	"time"
)

func BenchmarkPut(b *testing.B) {
	c, _ := Init[string](CleanInterval(0), TTL(5*time.Minute))
	defer c.Stop()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Put(fmt.Sprintf("key-%d", i), "value")
	}
}

func BenchmarkGet_Hit(b *testing.B) {
	c, _ := Init[string](CleanInterval(0), TTL(5*time.Minute))
	defer c.Stop()
	c.Put("key", "value")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Get("key")
	}
}

func BenchmarkGet_Miss(b *testing.B) {
	c, _ := Init[string](CleanInterval(0), TTL(5*time.Minute))
	defer c.Stop()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Get("missing")
	}
}

func BenchmarkPop(b *testing.B) {
	c, _ := Init[string](CleanInterval(0), TTL(5*time.Minute))
	defer c.Stop()

	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i)
		c.Put(key, "value")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Pop(fmt.Sprintf("key-%d", i))
	}
}

func BenchmarkDelete(b *testing.B) {
	c, _ := Init[string](CleanInterval(0), TTL(5*time.Minute))
	defer c.Stop()

	for i := 0; i < b.N; i++ {
		c.Put(fmt.Sprintf("key-%d", i), "value")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Delete(fmt.Sprintf("key-%d", i))
	}
}

func BenchmarkHas(b *testing.B) {
	c, _ := Init[string](CleanInterval(0), TTL(5*time.Minute))
	defer c.Stop()
	c.Put("key", "value")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Has("key")
	}
}

func BenchmarkPutParallel(b *testing.B) {
	c, _ := Init[string](CleanInterval(0), TTL(5*time.Minute))
	defer c.Stop()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			c.Put(fmt.Sprintf("key-%d", i), "value")
			i++
		}
	})
}

func BenchmarkGetParallel(b *testing.B) {
	c, _ := Init[string](CleanInterval(0), TTL(5*time.Minute))
	defer c.Stop()
	c.Put("key", "value")

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c.Get("key")
		}
	})
}
