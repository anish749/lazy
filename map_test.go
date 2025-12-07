package lazy

import (
	"errors"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestMap_Basic(t *testing.T) {
	// Test basic transformation from string to int
	strLazy := NewLazy(func() (string, error) {
		return "hello", nil
	})

	intLazy := Map(strLazy, func(s string) int {
		return len(s)
	})

	result, err := intLazy.Get()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != 5 {
		t.Errorf("Expected 5, got %d", result)
	}
}

func TestMap_ErrorPropagation(t *testing.T) {
	// Test that errors from the original lazy are propagated
	expectedErr := errors.New("original error")
	strLazy := NewLazy(func() (string, error) {
		return "", expectedErr
	})

	// The mapping function should NOT be called when there's an error
	mapCalled := false
	intLazy := Map(strLazy, func(s string) int {
		mapCalled = true
		return len(s)
	})

	result, err := intLazy.Get()
	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
	if result != 0 {
		t.Errorf("Expected zero value, got %d", result)
	}
	if mapCalled {
		t.Error("Map function should not be called when original lazy has error")
	}
}

func TestMap_Laziness(t *testing.T) {
	// Test that Map doesn't evaluate until Get is called
	var originalEvaluated bool
	var mapEvaluated bool

	strLazy := NewLazy(func() (string, error) {
		originalEvaluated = true
		return "hello", nil
	})

	intLazy := Map(strLazy, func(s string) int {
		mapEvaluated = true
		return len(s)
	})

	// Neither should be evaluated yet
	if originalEvaluated {
		t.Error("Original lazy should not be evaluated before Get()")
	}
	if mapEvaluated {
		t.Error("Map function should not be evaluated before Get()")
	}

	// Now trigger evaluation
	_, _ = intLazy.Get()

	if !originalEvaluated {
		t.Error("Original lazy should be evaluated after Get()")
	}
	if !mapEvaluated {
		t.Error("Map function should be evaluated after Get()")
	}
}

func TestMap_CachingBehavior(t *testing.T) {
	// Test that the mapped lazy caches its result
	var callCount int32

	strLazy := NewLazy(func() (string, error) {
		atomic.AddInt32(&callCount, 1)
		return "hello", nil
	})

	intLazy := Map(strLazy, func(s string) int {
		return len(s)
	})

	// Call Get multiple times
	for i := 0; i < 5; i++ {
		result, err := intLazy.Get()
		if err != nil {
			t.Errorf("Unexpected error on call %d: %v", i, err)
		}
		if result != 5 {
			t.Errorf("Expected 5 on call %d, got %d", i, result)
		}
	}

	// Original should only be evaluated once
	if atomic.LoadInt32(&callCount) != 1 {
		t.Errorf("Expected original lazy to be called once, got %d calls", callCount)
	}
}

func TestMap_ChainedMaps(t *testing.T) {
	// Test chaining multiple Map operations
	intLazy := NewLazy(func() (int, error) {
		return 42, nil
	})

	strLazy := Map(intLazy, func(i int) string {
		return strconv.Itoa(i)
	})

	upperLazy := Map(strLazy, func(s string) string {
		return "VALUE: " + s
	})

	result, err := upperLazy.Get()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != "VALUE: 42" {
		t.Errorf("Expected 'VALUE: 42', got '%s'", result)
	}
}

func TestMap_WithInitializedLazy(t *testing.T) {
	// Test Map with an already initialized lazy
	strLazy := InitializedLazy[string, error]("preset")

	intLazy := Map(strLazy, func(s string) int {
		return len(s)
	})

	result, err := intLazy.Get()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != 6 {
		t.Errorf("Expected 6, got %d", result)
	}
}

func TestMap_Concurrency(t *testing.T) {
	// Test concurrent access to a mapped lazy
	strLazy := NewLazy(func() (string, error) {
		time.Sleep(10 * time.Millisecond)
		return "hello", nil
	})

	intLazy := Map(strLazy, func(s string) int {
		return len(s)
	})

	const numGoroutines = 50
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	var errorCount int32

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			result, err := intLazy.Get()
			if err != nil {
				atomic.AddInt32(&errorCount, 1)
				return
			}
			if result != 5 {
				atomic.AddInt32(&errorCount, 1)
			}
		}()
	}

	wg.Wait()

	if errorCount > 0 {
		t.Errorf("%d errors occurred during concurrent access", errorCount)
	}
}

func TestFlatMap_Basic(t *testing.T) {
	// Test basic FlatMap operation
	intLazy := NewLazy(func() (int, error) {
		return 42, nil
	})

	strLazy := FlatMap(intLazy, func(i int) *Lazy[string, error] {
		return NewLazy(func() (string, error) {
			return strconv.Itoa(i), nil
		})
	})

	result, err := strLazy.Get()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != "42" {
		t.Errorf("Expected '42', got '%s'", result)
	}
}

func TestFlatMap_ErrorPropagation_FromOriginal(t *testing.T) {
	// Test that errors from the original lazy are propagated
	expectedErr := errors.New("original error")
	intLazy := NewLazy(func() (int, error) {
		return 0, expectedErr
	})

	// The FlatMap function should NOT be called when there's an error
	flatMapCalled := false
	strLazy := FlatMap(intLazy, func(i int) *Lazy[string, error] {
		flatMapCalled = true
		return NewLazy(func() (string, error) {
			return strconv.Itoa(i), nil
		})
	})

	result, err := strLazy.Get()
	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
	if result != "" {
		t.Errorf("Expected zero value, got '%s'", result)
	}
	if flatMapCalled {
		t.Error("FlatMap function should not be called when original lazy has error")
	}
}

func TestFlatMap_ErrorPropagation_FromInnerLazy(t *testing.T) {
	// Test that errors from the inner lazy are propagated
	expectedErr := errors.New("inner lazy error")
	intLazy := NewLazy(func() (int, error) {
		return 42, nil
	})

	strLazy := FlatMap(intLazy, func(i int) *Lazy[string, error] {
		return NewLazy(func() (string, error) {
			return "", expectedErr
		})
	})

	result, err := strLazy.Get()
	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
	if result != "" {
		t.Errorf("Expected zero value, got '%s'", result)
	}
}

func TestFlatMap_Laziness(t *testing.T) {
	// Test that FlatMap doesn't evaluate until Get is called
	var originalEvaluated bool
	var flatMapCalled bool
	var innerEvaluated bool

	intLazy := NewLazy(func() (int, error) {
		originalEvaluated = true
		return 42, nil
	})

	strLazy := FlatMap(intLazy, func(i int) *Lazy[string, error] {
		flatMapCalled = true
		return NewLazy(func() (string, error) {
			innerEvaluated = true
			return strconv.Itoa(i), nil
		})
	})

	// Nothing should be evaluated yet
	if originalEvaluated {
		t.Error("Original lazy should not be evaluated before Get()")
	}
	if flatMapCalled {
		t.Error("FlatMap function should not be called before Get()")
	}
	if innerEvaluated {
		t.Error("Inner lazy should not be evaluated before Get()")
	}

	// Now trigger evaluation
	_, _ = strLazy.Get()

	if !originalEvaluated {
		t.Error("Original lazy should be evaluated after Get()")
	}
	if !flatMapCalled {
		t.Error("FlatMap function should be called after Get()")
	}
	if !innerEvaluated {
		t.Error("Inner lazy should be evaluated after Get()")
	}
}

func TestFlatMap_CachingBehavior(t *testing.T) {
	// Test that the FlatMapped lazy caches its result
	var originalCallCount int32
	var innerCallCount int32

	intLazy := NewLazy(func() (int, error) {
		atomic.AddInt32(&originalCallCount, 1)
		return 42, nil
	})

	strLazy := FlatMap(intLazy, func(i int) *Lazy[string, error] {
		return NewLazy(func() (string, error) {
			atomic.AddInt32(&innerCallCount, 1)
			return strconv.Itoa(i), nil
		})
	})

	// Call Get multiple times
	for i := 0; i < 5; i++ {
		result, err := strLazy.Get()
		if err != nil {
			t.Errorf("Unexpected error on call %d: %v", i, err)
		}
		if result != "42" {
			t.Errorf("Expected '42' on call %d, got '%s'", i, result)
		}
	}

	// Both should only be evaluated once due to caching
	if atomic.LoadInt32(&originalCallCount) != 1 {
		t.Errorf("Expected original lazy to be called once, got %d calls", originalCallCount)
	}
	if atomic.LoadInt32(&innerCallCount) != 1 {
		t.Errorf("Expected inner lazy to be called once, got %d calls", innerCallCount)
	}
}

func TestFlatMap_ChainedFlatMaps(t *testing.T) {
	// Test chaining multiple FlatMap operations
	intLazy := NewLazy(func() (int, error) {
		return 5, nil
	})

	// Chain: int -> string (length) -> []rune
	step1 := FlatMap(intLazy, func(i int) *Lazy[string, error] {
		return NewLazy(func() (string, error) {
			return strings.Repeat("a", i), nil
		})
	})

	step2 := FlatMap(step1, func(s string) *Lazy[int, error] {
		return NewLazy(func() (int, error) {
			return len(s) * 2, nil
		})
	})

	result, err := step2.Get()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != 10 {
		t.Errorf("Expected 10, got %d", result)
	}
}

func TestFlatMap_WithInitializedInnerLazy(t *testing.T) {
	// Test FlatMap returning an already initialized lazy
	intLazy := NewLazy(func() (int, error) {
		return 42, nil
	})

	strLazy := FlatMap(intLazy, func(i int) *Lazy[string, error] {
		return InitializedLazy[string, error](strconv.Itoa(i))
	})

	result, err := strLazy.Get()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != "42" {
		t.Errorf("Expected '42', got '%s'", result)
	}
}

func TestFlatMap_Concurrency(t *testing.T) {
	// Test concurrent access to a FlatMapped lazy
	intLazy := NewLazy(func() (int, error) {
		time.Sleep(10 * time.Millisecond)
		return 42, nil
	})

	strLazy := FlatMap(intLazy, func(i int) *Lazy[string, error] {
		return NewLazy(func() (string, error) {
			time.Sleep(5 * time.Millisecond)
			return strconv.Itoa(i), nil
		})
	})

	const numGoroutines = 50
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	var errorCount int32

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			result, err := strLazy.Get()
			if err != nil {
				atomic.AddInt32(&errorCount, 1)
				return
			}
			if result != "42" {
				atomic.AddInt32(&errorCount, 1)
			}
		}()
	}

	wg.Wait()

	if errorCount > 0 {
		t.Errorf("%d errors occurred during concurrent access", errorCount)
	}
}

func TestMap_FlatMap_Composition(t *testing.T) {
	// Test composing Map and FlatMap together
	intLazy := NewLazy(func() (int, error) {
		return 3, nil
	})

	// Map: int -> string
	strLazy := Map(intLazy, func(i int) string {
		return strings.Repeat("x", i)
	})

	// FlatMap: string -> lazy int (length * 10)
	finalLazy := FlatMap(strLazy, func(s string) *Lazy[int, error] {
		return NewLazy(func() (int, error) {
			return len(s) * 10, nil
		})
	})

	result, err := finalLazy.Get()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != 30 {
		t.Errorf("Expected 30, got %d", result)
	}
}

func TestMap_WithDifferentTypes(t *testing.T) {
	// Test Map with various type combinations
	type Person struct {
		Name string
		Age  int
	}

	personLazy := NewLazy(func() (Person, error) {
		return Person{Name: "Alice", Age: 30}, nil
	})

	nameLazy := Map(personLazy, func(p Person) string {
		return p.Name
	})

	ageLazy := Map(personLazy, func(p Person) int {
		return p.Age
	})

	name, err := nameLazy.Get()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if name != "Alice" {
		t.Errorf("Expected 'Alice', got '%s'", name)
	}

	age, err := ageLazy.Get()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if age != 30 {
		t.Errorf("Expected 30, got %d", age)
	}
}

func TestFlatMap_DependentComputation(t *testing.T) {
	// Simulate a real-world scenario: fetch user ID, then fetch user details
	type User struct {
		ID   int
		Name string
	}

	// First lazy: get user ID
	userIDLazy := NewLazy(func() (int, error) {
		return 123, nil
	})

	// Second lazy depends on the first: fetch user by ID
	userLazy := FlatMap(userIDLazy, func(id int) *Lazy[User, error] {
		return NewLazy(func() (User, error) {
			// Simulate fetching user from database
			return User{ID: id, Name: "User-" + strconv.Itoa(id)}, nil
		})
	})

	user, err := userLazy.Get()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if user.ID != 123 {
		t.Errorf("Expected user ID 123, got %d", user.ID)
	}
	if user.Name != "User-123" {
		t.Errorf("Expected user name 'User-123', got '%s'", user.Name)
	}
}

type customError struct {
	Code    int
	Message string
}

func (e *customError) Error() string {
	return e.Message
}

func TestMap_PreservesErrorType(t *testing.T) {
	// Test that Map preserves the error type
	intLazy := NewLazy(func() (int, *customError) {
		return 42, nil
	})

	strLazy := Map(intLazy, func(i int) string {
		return strconv.Itoa(i)
	})

	result, err := strLazy.Get()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != "42" {
		t.Errorf("Expected '42', got '%s'", result)
	}
}

func TestFlatMap_PreservesErrorType(t *testing.T) {
	// Test that FlatMap preserves the error type
	intLazy := NewLazy(func() (int, *customError) {
		return 42, nil
	})

	strLazy := FlatMap(intLazy, func(i int) *Lazy[string, *customError] {
		return NewLazy(func() (string, *customError) {
			return strconv.Itoa(i), nil
		})
	})

	result, err := strLazy.Get()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != "42" {
		t.Errorf("Expected '42', got '%s'", result)
	}
}

func TestGoEvaluateLazies_WithMappedLazies(t *testing.T) {
	// Test that mapped lazies work with GoEvaluateLazies
	var counter int32

	lazy1 := NewLazy(func() (int, error) {
		atomic.AddInt32(&counter, 1)
		time.Sleep(20 * time.Millisecond)
		return 1, nil
	})

	lazy2 := Map(lazy1, func(i int) string {
		return strconv.Itoa(i)
	})

	lazy3 := NewLazy(func() (int, error) {
		atomic.AddInt32(&counter, 1)
		time.Sleep(20 * time.Millisecond)
		return 3, nil
	})

	// Evaluate all concurrently
	start := time.Now()
	GoEvaluateLazies(lazy2, lazy3)
	elapsed := time.Since(start)

	// Should be concurrent, not sequential
	if elapsed >= 35*time.Millisecond {
		t.Errorf("Expected concurrent execution, but took %v", elapsed)
	}

	// Verify results
	val2, err2 := lazy2.Get()
	if err2 != nil || val2 != "1" {
		t.Errorf("lazy2 incorrect: val=%s, err=%v", val2, err2)
	}

	val3, err3 := lazy3.Get()
	if err3 != nil || val3 != 3 {
		t.Errorf("lazy3 incorrect: val=%d, err=%v", val3, err3)
	}
}
