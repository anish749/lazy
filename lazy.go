package lazy

import "sync"

// Lazy represents a value that is initialized on first access.
// It ensures the initialization function is called only once, even in concurrent scenarios.
type Lazy[T any, E error] struct {
	value T
	err   E
	init  func() (T, E)
	once  sync.Once
}

// NewLazy creates a new lazy value with the provided initialization function.
// The function will be called only once when Get is first called.
func NewLazy[T any, E error](init func() (T, E)) *Lazy[T, E] {
	return &Lazy[T, E]{
		init: init,
	}
}

// InitializedLazy creates a lazy value that is already initialized with the given value.
// This allows for calling funcs that can work with a lazy,
// however we already have the value.
func InitializedLazy[T any, E error](value T) *Lazy[T, E] {
	return &Lazy[T, E]{
		value: value,
		init:  nil, // No init function needed
	}
}

// Get retrieves the lazy value, initializing it on first call if needed.
// The initialization function is guaranteed to be called at most once,
// even when Get is called concurrently from multiple goroutines.
// Returns the computed value and any error from initialization.
func (l *Lazy[T, E]) Get() (value T, err E) {
	// Skip once.Do if init is nil, meaning the value was pre-initialized
	if l.init != nil {
		l.once.Do(func() {
			l.value, l.err = l.init()
		})
	}
	value = l.value
	err = l.err
	return
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
