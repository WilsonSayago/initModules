package initModules

import (
	"sync"
	"sync/atomic"
	"testing"
)

type onceValueStructConcurrent struct {
	ID int
}

type oncePointerStruct struct {
	Name string
}

type onceValueStructContainer struct {
	ID int
}

type onceValueStructLegacy struct {
	ID int
}

func TestOnceValue_Concurrent(t *testing.T) {
	t.Parallel()

	var count atomic.Int32
	var wg sync.WaitGroup
	const n = 50
	ptrs := make([]*onceValueStructConcurrent, n)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			ptrs[idx] = OnceValue(func() onceValueStructConcurrent {
				count.Add(1)
				return onceValueStructConcurrent{ID: 7}
			})
		}(i)
	}
	wg.Wait()

	if count.Load() != 1 {
		t.Fatalf("constructor calls = %d, want 1", count.Load())
	}
	for i, p := range ptrs {
		if p == nil || p.ID != 7 {
			t.Fatalf("idx %d: unexpected %v", i, p)
		}
		if p != ptrs[0] {
			t.Fatalf("idx %d: different pointer", i)
		}
	}
}

func TestOnce_PointerConstructor_Concurrent(t *testing.T) {
	t.Parallel()

	var count atomic.Int32
	var wg sync.WaitGroup
	const n = 50
	ptrs := make([]*oncePointerStruct, n)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			ptrs[idx] = Once(func() *oncePointerStruct {
				count.Add(1)
				return &oncePointerStruct{Name: "svc"}
			})
		}(i)
	}
	wg.Wait()

	if count.Load() != 1 {
		t.Fatalf("constructor calls = %d, want 1", count.Load())
	}
	for i, p := range ptrs {
		if p == nil || p.Name != "svc" {
			t.Fatalf("idx %d: unexpected %v", i, p)
		}
		if p != ptrs[0] {
			t.Fatalf("idx %d: different pointer", i)
		}
	}
}

func TestContainer_OnceIn_IsolatedFromGlobal(t *testing.T) {
	t.Parallel()

	global := OnceValue(func() onceValueStructContainer { return onceValueStructContainer{ID: 1} })

	c1 := NewContainer()
	c2 := NewContainer()
	fromC1a := OnceValueIn(c1, func() onceValueStructContainer { return onceValueStructContainer{ID: 10} })
	fromC1b := OnceValueIn(c1, func() onceValueStructContainer { return onceValueStructContainer{ID: 99} })
	fromC2 := OnceValueIn(c2, func() onceValueStructContainer { return onceValueStructContainer{ID: 20} })

	if global.ID != 1 || fromC1a.ID != 10 || fromC1b.ID != 10 || fromC2.ID != 20 {
		t.Fatalf("unexpected ids: global=%d c1a=%d c1b=%d c2=%d", global.ID, fromC1a.ID, fromC1b.ID, fromC2.ID)
	}
	if fromC1a != fromC1b {
		t.Fatal("expected same pointer within container")
	}
	if fromC1a == fromC2 {
		t.Fatal("expected different pointers across containers")
	}
}

func TestBaseInstance_BackwardCompatible(t *testing.T) {
	var count atomic.Int32
	a := NewInstance[onceValueStructLegacy]().GetInstance(func() onceValueStructLegacy {
		count.Add(1)
		return onceValueStructLegacy{ID: 3}
	})
	b := NewInstance[onceValueStructLegacy]().GetInstance(func() onceValueStructLegacy {
		count.Add(1)
		return onceValueStructLegacy{ID: 99}
	})
	if count.Load() != 1 || a != b || a.ID != 3 {
		t.Fatalf("got count=%d a=%v b=%v", count.Load(), a, b)
	}
}
