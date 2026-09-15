package initModules

import (
	"sync"
	"sync/atomic"
	"testing"
)

func TestGetInstance_ConcurrentSameKey(t *testing.T) {
	t.Parallel()

	const key = "concurrent-test-key"
	var constructCount atomic.Int32

	var wg sync.WaitGroup
	const goroutines = 100
	results := make([]interface{}, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx] = GetInstance(key, func() interface{} {
				constructCount.Add(1)
				return &struct{ ID int }{ID: 42}
			})
		}(i)
	}

	wg.Wait()

	if constructCount.Load() != 1 {
		t.Fatalf("expected constructor to run once, got %d", constructCount.Load())
	}

	first := results[0]
	for i, r := range results {
		if r != first {
			t.Fatalf("goroutine %d got a different instance pointer", i)
		}
	}
}

func TestGetInstance_ReusesExistingWithoutCallingFactory(t *testing.T) {
	key := "reuse-key"
	var count atomic.Int32

	first := GetInstance(key, func() interface{} {
		count.Add(1)
		return "first"
	})
	second := GetInstance(key, func() interface{} {
		count.Add(1)
		return "second"
	})

	if first != "first" || second != "first" {
		t.Fatalf("expected first instance to be reused, got %v and %v", first, second)
	}
	if count.Load() != 1 {
		t.Fatalf("expected factory to run once, got %d", count.Load())
	}
}
