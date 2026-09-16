package initModules

import (
	"strings"
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

func resetGlobalSingletonsForTest(t *testing.T) {
	t.Helper()
	globalSingletons.Clear()
	t.Cleanup(globalSingletons.Clear)
}

func TestOnceValue_Concurrent(t *testing.T) {
	resetGlobalSingletonsForTest(t)

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
	resetGlobalSingletonsForTest(t)

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
	resetGlobalSingletonsForTest(t)

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

func TestContainer_OnceIn_NilContainer(t *testing.T) {
	resetGlobalSingletonsForTest(t)

	var constructed atomic.Int32
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for nil Container")
		}
		msg, ok := r.(string)
		if !ok || !strings.Contains(msg, "nil Container") {
			t.Fatalf("panic = %v", r)
		}
		if constructed.Load() != 0 {
			t.Fatal("constructor must not run")
		}
		global := OnceValue(func() onceValueStructContainer {
			constructed.Add(1)
			return onceValueStructContainer{ID: 1}
		})
		if constructed.Load() != 1 || global.ID != 1 {
			t.Fatalf("global registry was modified: count=%d id=%d", constructed.Load(), global.ID)
		}
	}()
	_ = OnceIn[onceValueStructContainer](nil, func() *onceValueStructContainer {
		constructed.Add(1)
		return &onceValueStructContainer{ID: 99}
	})
}

func TestContainer_OnceValueIn_NilContainer(t *testing.T) {
	resetGlobalSingletonsForTest(t)

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for nil Container")
		}
		msg, ok := r.(string)
		if !ok || !strings.Contains(msg, "nil Container") {
			t.Fatalf("panic = %v", r)
		}
		global := OnceValue(func() onceValueStructContainer { return onceValueStructContainer{ID: 1} })
		if global.ID != 1 {
			t.Fatalf("global registry was modified: %+v", global)
		}
	}()
	_ = OnceValueIn[onceValueStructContainer](nil, func() onceValueStructContainer {
		return onceValueStructContainer{ID: 99}
	})
}

func TestOnceIn_NilConstructor(t *testing.T) {
	c := NewContainer()
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for nil constructor")
		}
		msg, ok := r.(string)
		if !ok || !strings.Contains(msg, "nil constructor") {
			t.Fatalf("panic = %v", r)
		}
	}()
	_ = OnceIn[oncePointerStruct](c, nil)
}

func TestOnce_NilConstructor(t *testing.T) {
	resetGlobalSingletonsForTest(t)
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for nil constructor")
		}
		msg, ok := r.(string)
		if !ok || !strings.Contains(msg, "nil constructor") {
			t.Fatalf("panic = %v", r)
		}
	}()
	_ = Once[oncePointerStruct](nil)
}

func TestOnceIn_ConstructorReturnsNil(t *testing.T) {
	c := NewContainer()
	ctor := func() *oncePointerStruct { return nil }
	assertPanicString(t, func() { _ = OnceIn(c, ctor) }, "constructor returned nil")
	assertPanicString(t, func() { _ = OnceIn(c, ctor) }, "constructor returned nil")
}

func TestOnce_ConstructorReturnsNil_SecondCallKeepsMessage(t *testing.T) {
	resetGlobalSingletonsForTest(t)

	var calls atomic.Int32
	ctor := func() *oncePointerStruct {
		calls.Add(1)
		return nil
	}
	assertPanicString(t, func() { _ = Once(ctor) }, "constructor returned nil")
	assertPanicString(t, func() { _ = Once(ctor) }, "constructor returned nil")
	if calls.Load() != 1 {
		t.Fatalf("constructor calls = %d, want 1", calls.Load())
	}
}

func TestOnceIn_ConstructorPanic_SecondCallKeepsMessage(t *testing.T) {
	c := NewContainer()
	var calls atomic.Int32
	ctor := func() *oncePointerStruct {
		calls.Add(1)
		panic("boom from ctor")
	}
	assertPanicString(t, func() { _ = OnceIn(c, ctor) }, "boom from ctor")
	assertPanicString(t, func() { _ = OnceIn(c, ctor) }, "boom from ctor")
	if calls.Load() != 1 {
		t.Fatalf("constructor calls = %d, want 1", calls.Load())
	}
}

func assertPanicString(t *testing.T, fn func(), substr string) {
	t.Helper()
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic")
		}
		msg, ok := r.(string)
		if !ok || !strings.Contains(msg, substr) {
			t.Fatalf("panic = %#v, want substring %q", r, substr)
		}
	}()
	fn()
}

func TestBaseInstance_BackwardCompatible(t *testing.T) {
	resetGlobalSingletonsForTest(t)

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
