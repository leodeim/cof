package cof_test

import (
	"fmt"
	"time"

	"github.com/leonidasdeim/cof"
)

func ExampleInit() {
	c, err := cof.Init[string](cof.TTL(5*time.Minute), cof.CleanInterval(1*time.Minute))
	if err != nil {
		panic(err)
	}
	defer c.Stop()

	c.Put("greeting", "hello, world!")
	v, ok := c.Get("greeting")
	fmt.Println(v, ok)
	// Output: hello, world! true
}

func ExampleC_Put() {
	c, _ := cof.Init[int]()
	defer c.Stop()

	c.Put("answer", 42)
	v, _ := c.Get("answer")
	fmt.Println(v)
	// Output: 42
}

func ExampleC_PutWithTTL() {
	c, _ := cof.Init[string](cof.TTL(1 * time.Hour))
	defer c.Stop()

	// This specific entry expires in 100ms, overriding the 1h default.
	c.PutWithTTL("flash", "gone soon", 100*time.Millisecond)

	v, ok := c.Get("flash")
	fmt.Println(v, ok)
	// Output: gone soon true
}

func ExampleC_Pop() {
	c, _ := cof.Init[string]()
	defer c.Stop()

	c.Put("token", "abc123")
	v, ok := c.Pop("token")
	fmt.Println(v, ok)

	_, ok = c.Get("token")
	fmt.Println(ok)
	// Output:
	// abc123 true
	// false
}

func ExampleC_Delete() {
	c, _ := cof.Init[string]()
	defer c.Stop()

	c.Put("k", "v")
	c.Delete("k")

	_, ok := c.Get("k")
	fmt.Println(ok)
	// Output: false
}

func ExampleC_Has() {
	c, _ := cof.Init[string]()
	defer c.Stop()

	fmt.Println(c.Has("k"))
	c.Put("k", "v")
	fmt.Println(c.Has("k"))
	// Output:
	// false
	// true
}

func ExampleC_Len() {
	c, _ := cof.Init[string]()
	defer c.Stop()

	c.Put("a", "1")
	c.Put("b", "2")
	fmt.Println(c.Len())
	// Output: 2
}

func ExampleC_Keys() {
	c, _ := cof.Init[string]()
	defer c.Stop()

	c.Put("banana", "b")
	c.Put("apple", "a")
	fmt.Println(c.Keys())
	// Output: [apple banana]
}

func ExampleC_Clear() {
	c, _ := cof.Init[string]()
	defer c.Stop()

	c.Put("a", "1")
	c.Put("b", "2")
	c.Clear()
	fmt.Println(c.Len())
	// Output: 0
}
