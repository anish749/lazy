package lazy

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestLazy_Get(t *testing.T) {
	tests := []struct {
		name     string
		initFunc func() (string, error)
		want     string
		wantErr  bool
	}{
		{
			name: "successful initialization",
			initFunc: func() (string, error) {
				return "test value", nil
			},
			want:    "test value",
			wantErr: false,
		},
		{
			name: "error during initialization",
			initFunc: func() (string, error) {
				return "", errors.New("initialization error")
			},
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lazy := NewLazy(tt.initFunc)

			// First call should initialize
			got, err := lazy.Get()
			if (err != nil) != tt.wantErr {
				t.Errorf("Lazy.Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Lazy.Get() = %v, want %v", got, tt.want)
			}

			// Second call should reuse the cached value
			got2, err2 := lazy.Get()
			if (err2 != nil) != tt.wantErr {
				t.Errorf("Lazy.Get() second call error = %v, wantErr %v", err2, tt.wantErr)
				return
			}
			if got2 != tt.want {
				t.Errorf("Lazy.Get() second call = %v, want %v", got2, tt.want)
			}
		})
	}
}

func TestLazy_InitCalledOnce(t *testing.T) {
	// This test ensures init function is only called once
	callCount := 0
	initFunc := func() (int, error) {
		callCount++
		return callCount, nil
	}

	lazy := NewLazy(initFunc)

	// First call
	val1, err1 := lazy.Get()
	if err1 != nil {
		t.Errorf("Unexpected error: %v", err1)
	}
	if val1 != 1 {
		t.Errorf("Expected value 1, got %d", val1)
	}
	if callCount != 1 {
		t.Errorf("Expected init to be called once, got %d calls", callCount)
	}

	// Second call should reuse the cached value
	val2, err2 := lazy.Get()
	if err2 != nil {
		t.Errorf("Unexpected error: %v", err2)
	}
	if val2 != 1 {
		t.Errorf("Expected value 1, got %d", val2)
	}
	if callCount != 1 {
		t.Errorf("Expected init to still be called once, got %d calls", callCount)
	}
}

func TestLazy_WithDifferentTypes(t *testing.T) {
	// Test with a slice
	sliceLazy := NewLazy(func() ([]int, error) {
		return []int{1, 2, 3}, nil
	})

	sliceVal, err := sliceLazy.Get()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(sliceVal) != 3 || sliceVal[0] != 1 || sliceVal[1] != 2 || sliceVal[2] != 3 {
		t.Errorf("Unexpected slice value: %v", sliceVal)
	}

	// Test with a struct
	type testStruct struct {
		Value string
	}

	structLazy := NewLazy(func() (testStruct, error) {
		return testStruct{Value: "test"}, nil
	})

	structVal, err := structLazy.Get()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if structVal.Value != "test" {
		t.Errorf("Unexpected struct value: %v", structVal)
	}
}

func TestGoEvaluateLazies(t *testing.T) {
	// Test that all lazy values are evaluated
	var counter int32 = 0

	// Create multiple lazy values that increment the counter when evaluated
	lazy1 := NewLazy(func() (bool, error) {
		atomic.AddInt32(&counter, 1)
		time.Sleep(50 * time.Millisecond) // Small delay to test concurrency
		return true, nil
	})

	lazy2 := NewLazy(func() (bool, error) {
		atomic.AddInt32(&counter, 1)
		time.Sleep(30 * time.Millisecond)
		return true, nil
	})

	lazy3 := NewLazy(func() (bool, error) {
		atomic.AddInt32(&counter, 1)
		time.Sleep(10 * time.Millisecond)
		return true, nil
	})

	// Time how long it takes to evaluate all lazies
	start := time.Now()
	GoEvaluateLazies(lazy1, lazy2, lazy3)
	elapsed := time.Since(start)

	// Check that all lazies were evaluated
	if atomic.LoadInt32(&counter) != 3 {
		t.Errorf("Expected 3 lazy evaluations, got %d", counter)
	}

	// Verify concurrency by checking elapsed time
	// The total time should be closer to the longest task (50ms) than the sum (90ms)
	// Adding some buffer for test environment variations
	if elapsed >= 70*time.Millisecond {
		t.Errorf("Expected concurrent execution, but took %v", elapsed)
	}

	// Verify that values are actually computed
	val1, err1 := lazy1.Get()
	if !val1 || err1 != nil {
		t.Errorf("lazy1 not properly evaluated: val=%v, err=%v", val1, err1)
	}

	val2, err2 := lazy2.Get()
	if !val2 || err2 != nil {
		t.Errorf("lazy2 not properly evaluated: val=%v, err=%v", val2, err2)
	}

	val3, err3 := lazy3.Get()
	if !val3 || err3 != nil {
		t.Errorf("lazy3 not properly evaluated: val=%v, err=%v", val3, err3)
	}
}

func TestGoEvaluateLaziesConcurrency(t *testing.T) {
	// This test verifies that evaluations happen concurrently

	// Use sync mechanism to carefully control execution order
	var mutex sync.Mutex
	cond := sync.NewCond(&mutex)

	var started, finished sync.WaitGroup
	started.Add(3)
	finished.Add(3)

	var order []int

	// Create lazies that will add their index to the order slice when done
	lazy1 := NewLazy(func() (int, error) {
		started.Done()

		// Wait for all goroutines to start
		mutex.Lock()
		cond.Wait() // Will be signaled to continue execution
		mutex.Unlock()

		time.Sleep(30 * time.Millisecond) // Longest sleep

		mutex.Lock()
		order = append(order, 1)
		mutex.Unlock()

		finished.Done()
		return 1, nil
	})

	lazy2 := NewLazy(func() (int, error) {
		started.Done()

		mutex.Lock()
		cond.Wait()
		mutex.Unlock()

		time.Sleep(20 * time.Millisecond) // Medium sleep

		mutex.Lock()
		order = append(order, 2)
		mutex.Unlock()

		finished.Done()
		return 2, nil
	})

	lazy3 := NewLazy(func() (int, error) {
		started.Done()

		mutex.Lock()
		cond.Wait()
		mutex.Unlock()

		time.Sleep(10 * time.Millisecond) // Shortest sleep

		mutex.Lock()
		order = append(order, 3)
		mutex.Unlock()

		finished.Done()
		return 3, nil
	})

	// Start all lazies and wait for them to reach the barrier
	go GoEvaluateLazies(lazy1, lazy2, lazy3)

	// Wait for all to start
	started.Wait()

	// Signal all to continue
	mutex.Lock()
	cond.Broadcast()
	mutex.Unlock()

	// Wait for all to finish
	finished.Wait()

	// If they ran concurrently, the fastest should finish first
	if len(order) != 3 {
		t.Errorf("Expected 3 results, got %d", len(order))
	} else if order[0] != 3 || order[1] != 2 || order[2] != 1 {
		t.Errorf("Expected order [3,2,1] (fastest to slowest), got %v", order)
	}
}

func TestInitializedLazy(t *testing.T) {
	// Basic test for initialized lazy
	initialValue := "initial value"
	lazy := InitializedLazy[string, error](initialValue)

	// Call Get multiple times to ensure it returns the same value
	val1, err1 := lazy.Get()
	if err1 != nil {
		t.Errorf("Unexpected error from initialized lazy: %v", err1)
	}
	if val1 != initialValue {
		t.Errorf("Expected %s, got %s", initialValue, val1)
	}

	val2, err2 := lazy.Get()
	if err2 != nil {
		t.Errorf("Unexpected error from second call to initialized lazy: %v", err2)
	}
	if val2 != initialValue {
		t.Errorf("Expected %s, got %s", initialValue, val2)
	}
}

func TestInitializedLazyVsNew(t *testing.T) {
	// This test compares behavior of initialized vs. non-initialized lazy values
	var counter int32 = 0

	// Normal lazy should call the init function once
	nonInitializedLazy := NewLazy(func() (string, error) {
		atomic.AddInt32(&counter, 1)
		return "computed value", nil
	})

	// Initialized lazy should never call an init function
	initializedLazy := InitializedLazy[string, error]("preset value")

	// Both Get calls should succeed
	nonInitVal, nonInitErr := nonInitializedLazy.Get()
	initVal, initErr := initializedLazy.Get()

	// Check normal lazy behavior
	if nonInitErr != nil {
		t.Errorf("Unexpected error from non-initialized lazy: %v", nonInitErr)
	}
	if nonInitVal != "computed value" {
		t.Errorf("Expected 'computed value', got '%s'", nonInitVal)
	}
	if atomic.LoadInt32(&counter) != 1 {
		t.Errorf("Expected init function to be called once, got %d calls", counter)
	}

	// Check initialized lazy behavior
	if initErr != nil {
		t.Errorf("Unexpected error from initialized lazy: %v", initErr)
	}
	if initVal != "preset value" {
		t.Errorf("Expected 'preset value', got '%s'", initVal)
	}

	// Call both again to verify behavior
	val, err := nonInitializedLazy.Get()
	if err != nil {
		t.Errorf("Unexpected error from second call to non-initialized lazy: %v", err)
	}
	if val != "computed value" {
		t.Errorf("Expected 'computed value', got '%s'", val)
	}

	val, err = initializedLazy.Get()
	if err != nil {
		t.Errorf("Unexpected error from second call to initialized lazy: %v", err)
	}
	if val != "preset value" {
		t.Errorf("Expected 'preset value', got '%s'", val)
	}

	// Init function still called exactly once overall
	if atomic.LoadInt32(&counter) != 1 {
		t.Errorf("Expected init function to still be called once, got %d calls", counter)
	}
}

func TestInitializedLazyConcurrency(t *testing.T) {
	// Test concurrent access to an initialized lazy value
	initialValue := "concurrent test value"
	lazy := InitializedLazy[string, error](initialValue)

	// Number of concurrent goroutines to test with
	const numGoroutines = 100
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Track any errors
	var errorCount int32 = 0

	// Launch multiple goroutines that all access the same lazy value
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			val, err := lazy.Get()
			if err != nil {
				atomic.AddInt32(&errorCount, 1)
				return
			}
			if val != initialValue {
				atomic.AddInt32(&errorCount, 1)
			}
		}()
	}

	wg.Wait()

	if errorCount > 0 {
		t.Errorf("%d errors occurred during concurrent access to initialized lazy", errorCount)
	}
}

func TestGoEvaluateWithInitializedLazies(t *testing.T) {
	// Test that GoEvaluateLazies handles a mix of initialized and non-initialized lazy values
	var computationCounter int32 = 0

	// Create a mix of lazy values
	lazy1 := InitializedLazy[string, error]("pre-computed")

	lazy2 := NewLazy(func() (string, error) {
		atomic.AddInt32(&computationCounter, 1)
		time.Sleep(20 * time.Millisecond)
		return "computed", nil
	})

	lazy3 := InitializedLazy[string, error]("also pre-computed")

	start := time.Now()
	GoEvaluateLazies(lazy1, lazy2, lazy3)
	elapsed := time.Since(start)

	// Only the non-initialized lazy should have resulted in a computation
	if atomic.LoadInt32(&computationCounter) != 1 {
		t.Errorf("Expected 1 computation, got %d", computationCounter)
	}

	// Verify that both pre-computed values are accessible
	val1, err1 := lazy1.Get()
	if val1 != "pre-computed" || err1 != nil {
		t.Errorf("Incorrect value from initialized lazy1: val=%v, err=%v", val1, err1)
	}

	val2, err2 := lazy2.Get()
	if val2 != "computed" || err2 != nil {
		t.Errorf("Incorrect value from computed lazy2: val=%v, err=%v", val2, err2)
	}

	val3, err3 := lazy3.Get()
	if val3 != "also pre-computed" || err3 != nil {
		t.Errorf("Incorrect value from initialized lazy3: val=%v, err=%v", val3, err3)
	}

	// The elapsed time should be close to the computation time of lazy2 (~20ms)
	// Add some buffer for test environment variations
	if elapsed < 10*time.Millisecond {
		t.Errorf("GoEvaluateLazies completed too quickly (%v), suggesting it didn't wait for computations", elapsed)
	}
}
