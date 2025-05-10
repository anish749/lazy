package lazy

import "sync"

type Lazy[T any, E error] struct {
	once  sync.Once
	value T
	err   E
	init  func() (T, E)
}

func NewLazy[T any, E error](init func() (T, E)) *Lazy[T, E] {
	return &Lazy[T, E]{
		init: init,
	}
}

// A helper constructor to create a lazy that wraps an already initialized value.
// This allows for calling funcs that can work with a lazy,
// however we already have the value.
func InitializedLazy[T any, E error](value T) *Lazy[T, E] {
	return &Lazy[T, E]{
		value: value,
		init:  nil, // No init function needed
	}
}

func (l *Lazy[T, E]) Get() (T, E) {
	// Skip once.Do if init is nil, meaning the value was pre-initialized
	if l.init != nil {
		l.once.Do(func() {
			l.value, l.err = l.init()
		})
	}
	return l.value, l.err
}

// lazyType is an interface constraint for lazy value getters
// It allows for materializing lazies to be called for different types.
type lazyType interface {
	getAndDiscard()
}

// getAndDiscard calls Get() but discards the results
func (l *Lazy[T, E]) getAndDiscard() {
	l.Get()
}

// GoEvaluateLazies concurrently evaluates all the provided lazy values.
// It starts each evaluation in a separate goroutine and waits for all of them to complete.
// This function is useful for pre-computing values that will be needed later,
// especially when dealing with computations that can be performed independently.
// After calling this function, subsequent calls to Get() on the lazy values
// will return immediately without blocking.
func GoEvaluateLazies(lazies ...lazyType) {
	wg := sync.WaitGroup{}
	for _, lazy := range lazies {
		wg.Add(1)
		go func(l lazyType) {
			defer wg.Done()
			l.getAndDiscard()
		}(lazy)
	}
	wg.Wait()
}
