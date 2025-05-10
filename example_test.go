package lazy

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

// Example test that demonstrates a multi-layer computation graph with caching
func TestLazyComputationGraphExample(t *testing.T) {
	// Simple in-memory cache implementation for the example
	cache := NewSimpleCache[int]()

	// Create a computation graph that calculates factorial of a number
	// with multiple layers that can be cached

	// Layer 1: Fetch the input number (simulating a database call)
	fetchNumberCalls := int32(0)
	lazyFetchNumber := NewLazy(func() (int, error) {
		atomic.AddInt32(&fetchNumberCalls, 1)
		time.Sleep(10 * time.Millisecond) // Simulate network delay
		return 5, nil
	})

	// Layer 1 with caching
	cachedNumber := NewLazy(func() (int, error) {
		val, err := cache.GetOrLoadWith(context.Background(), "input_number", lazyFetchNumber)
		if err != nil {
			return 0, err
		}
		return val, nil
	})

	// Layer 2: Calculate factorial
	factorialCalls := int32(0)
	lazyFactorial := NewLazy(func() (int, error) {
		num, err := cachedNumber.Get()
		if err != nil {
			return 0, err
		}

		atomic.AddInt32(&factorialCalls, 1)
		time.Sleep(20 * time.Millisecond) // Simulate heavy computation

		result := 1
		for i := 2; i <= num; i++ {
			result *= i
		}
		return result, nil
	})

	// Layer 2 with caching
	cachedFactorial := NewLazy(func() (int, error) {
		val, err := cache.GetOrLoadWith(context.Background(), "factorial", lazyFactorial)
		if err != nil {
			return 0, err
		}
		return val, nil
	})

	// Layer 3: Format factorial as string (another computation)
	formattedCalls := int32(0)
	lazyFormatted := NewLazy(func() (string, error) {
		num, err := cachedNumber.Get()
		if err != nil {
			return "", err
		}

		factorial, err := cachedFactorial.Get()
		if err != nil {
			return "", err
		}

		atomic.AddInt32(&formattedCalls, 1)
		time.Sleep(5 * time.Millisecond) // Simulate string formatting

		return fmt.Sprintf("%d! = %d", num, factorial), nil
	})

	// First execution - all computations should run
	result, err := lazyFormatted.Get()
	if err != nil {
		t.Fatalf("Failed to execute computation: %v", err)
	}

	if result != "5! = 120" {
		t.Errorf("Expected '5! = 120', got '%s'", result)
	}

	if atomic.LoadInt32(&fetchNumberCalls) != 1 {
		t.Errorf("Expected fetchNumberCalls to be 1, got %d", fetchNumberCalls)
	}

	if atomic.LoadInt32(&factorialCalls) != 1 {
		t.Errorf("Expected factorialCalls to be 1, got %d", factorialCalls)
	}

	if atomic.LoadInt32(&formattedCalls) != 1 {
		t.Errorf("Expected formattedCalls to be 1, got %d", formattedCalls)
	}

	// Second execution - should use cached values
	result2, err := lazyFormatted.Get()
	if err != nil {
		t.Fatalf("Failed to execute computation second time: %v", err)
	}

	if result2 != "5! = 120" {
		t.Errorf("Expected '5! = 120', got '%s'", result2)
	}

	// Call counts should not increase as values should be cached
	if atomic.LoadInt32(&fetchNumberCalls) != 1 {
		t.Errorf("Expected fetchNumberCalls to still be 1, got %d", fetchNumberCalls)
	}

	if atomic.LoadInt32(&factorialCalls) != 1 {
		t.Errorf("Expected factorialCalls to still be 1, got %d", factorialCalls)
	}

	if atomic.LoadInt32(&formattedCalls) != 1 {
		t.Errorf("Expected formattedCalls to still be 1, got %d", formattedCalls)
	}
}

// Example demonstrating concurrent execution of lazies
func TestConcurrentComputationExample(t *testing.T) {
	// Create a cache for string values
	stringCache := NewSimpleCache[string]()

	// Create three separate computations
	computation1Calls := int32(0)
	lazy1 := NewLazy(func() (string, error) {
		atomic.AddInt32(&computation1Calls, 1)
		time.Sleep(50 * time.Millisecond)
		return "result1", nil
	})

	computation2Calls := int32(0)
	lazy2 := NewLazy(func() (string, error) {
		atomic.AddInt32(&computation2Calls, 1)
		time.Sleep(40 * time.Millisecond)
		return "result2", nil
	})

	// This one is pre-initialized
	lazy3 := InitializedLazy[string, error]("result3")

	// Add caching layer to lazy1
	cachedLazy1 := NewLazy(func() (string, error) {
		return stringCache.GetOrLoadWith(context.Background(), "cached_result1", lazy1)
	})

	// Execute them concurrently with caching
	start := time.Now()
	GoEvaluateLazies(cachedLazy1, lazy2, lazy3)
	elapsed := time.Since(start)

	// Get results
	result1, err1 := cachedLazy1.Get()
	if err1 != nil || result1 != "result1" {
		t.Errorf("cachedLazy1 error or unexpected result: %v, %s", err1, result1)
	}

	result2, err2 := lazy2.Get()
	if err2 != nil || result2 != "result2" {
		t.Errorf("lazy2 error or unexpected result: %v, %s", err2, result2)
	}

	result3, err3 := lazy3.Get()
	if err3 != nil || result3 != "result3" {
		t.Errorf("lazy3 error or unexpected result: %v, %s", err3, result3)
	}

	// Check that computations ran only once
	if atomic.LoadInt32(&computation1Calls) != 1 {
		t.Errorf("computation1 should be called once, got %d", computation1Calls)
	}

	if atomic.LoadInt32(&computation2Calls) != 1 {
		t.Errorf("computation2 should be called once, got %d", computation2Calls)
	}

	// Verify concurrency actually happened - elapsed time should be close to the longest task
	if elapsed >= 90*time.Millisecond {
		t.Errorf("Expected concurrent execution, but took %v", elapsed)
	}

	// Second execution to test caching - should be instant
	secondStart := time.Now()
	result1Again, _ := cachedLazy1.Get()
	secondElapsed := time.Since(secondStart)

	if result1Again != "result1" {
		t.Errorf("Expected cached value 'result1', got '%s'", result1Again)
	}

	// The cached result should be retrieved almost instantly
	if secondElapsed >= 10*time.Millisecond {
		t.Errorf("Cached retrieval took too long: %v", secondElapsed)
	}

	// Computation count should remain the same
	if atomic.LoadInt32(&computation1Calls) != 1 {
		t.Errorf("computation1 should still be called only once, got %d", computation1Calls)
	}
}

// Example showing error handling
func TestErrorHandlingExample(t *testing.T) {
	// Create a cache for string values
	stringCache := NewSimpleCache[string]()

	// A lazy that will fail
	failingLazy := NewLazy(func() (string, error) {
		return "", errors.New("simulated failure")
	})

	// A dependent lazy that should handle the error
	dependentLazy := NewLazy(func() (string, error) {
		val, err := failingLazy.Get()
		if err != nil {
			return "handled: " + err.Error(), nil
		}
		return "value: " + val, nil
	})

	// Try caching the successful result
	successfulLazy := NewLazy(func() (string, error) {
		return "cached success", nil
	})

	// Cache the successful lazy computation
	cachedVal, err := stringCache.GetOrLoadWith(context.Background(), "success_key", successfulLazy)
	if err != nil {
		t.Errorf("Unexpected error caching value: %v", err)
	}
	if cachedVal != "cached success" {
		t.Errorf("Expected 'cached success', got '%s'", cachedVal)
	}

	// Verify the value stays in cache
	cachedAgain, err := stringCache.GetOrLoadWith(context.Background(), "success_key", NewLazy(func() (string, error) {
		t.Errorf("This should not be called as value should be retrieved from cache")
		return "", nil
	}))
	if err != nil {
		t.Errorf("Unexpected error retrieving from cache: %v", err)
	}
	if cachedAgain != "cached success" {
		t.Errorf("Cache retrieval failed, expected 'cached success', got '%s'", cachedAgain)
	}

	// Execute and verify error handling
	result, err := dependentLazy.Get()
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}

	if result != "handled: simulated failure" {
		t.Errorf("Expected error handling message, got: %s", result)
	}
}

// SimpleCache implements a minimal Cache for the example
type SimpleCache[T any] struct {
	data map[string]T
}

func NewSimpleCache[T any]() *SimpleCache[T] {
	return &SimpleCache[T]{
		data: make(map[string]T),
	}
}

// GetOrLoadWith retrieves a value from cache or computes it
func (c *SimpleCache[T]) GetOrLoadWith(
	ctx context.Context,
	key string,
	lazyValue *Lazy[T, error],
) (T, error) {
	// Check if value exists in cache
	if val, ok := c.data[key]; ok {
		return val, nil
	}

	// If not in cache, compute the value
	val, err := lazyValue.Get()
	if err != nil {
		var zero T
		return zero, err
	}

	// Store in cache for future use
	c.data[key] = val
	return val, nil
}
