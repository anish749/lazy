package lazy

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

// BenchmarkLazyGet measures the performance of creating and getting a value from a Lazy
func BenchmarkLazyGet(b *testing.B) {
	b.Run("Simple", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			lazy := NewLazy(func() (int, error) {
				return 42, nil
			})
			val, _ := lazy.Get()
			_ = val
		}
	})

	b.Run("WithComputation", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			lazy := NewLazy(func() (int, error) {
				result := 0
				for j := 0; j < 100; j++ {
					result += j
				}
				return result, nil
			})
			val, _ := lazy.Get()
			_ = val
		}
	})
}

// BenchmarkLazyReuse measures the performance of reusing a cached value
func BenchmarkLazyReuse(b *testing.B) {
	// First create a lazy value and initialize it
	lazy := NewLazy(func() (int, error) {
		// Simulate some computation
		result := 0
		for j := 0; j < 1000; j++ {
			result += j
		}
		return result, nil
	})

	// Initialize the value
	_, _ = lazy.Get()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		val, _ := lazy.Get()
		_ = val
	}
}

// BenchmarkLazyComputationDepth measures the performance with different depths of computation
func BenchmarkLazyComputationDepth(b *testing.B) {
	// Test with different depths of computation
	depths := []int{1, 5, 10, 20}

	for _, depth := range depths {
		b.Run(fmt.Sprintf("Depth-%d", depth), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				// Create a chain of lazy values
				var lastLazy *Lazy[int, error]

				// Initialize with the first lazy
				lastLazy = NewLazy(func() (int, error) {
					return 1, nil
				})

				// Create a chain of dependent lazy values
				for d := 1; d < depth; d++ {
					currentLazy := lastLazy
					lastLazy = NewLazy(func() (int, error) {
						val, err := currentLazy.Get()
						if err != nil {
							return 0, err
						}
						return val + 1, nil
					})
				}

				// Get the final result
				val, _ := lastLazy.Get()
				_ = val
			}
		})
	}
}

// BenchmarkInitializedVsNewLazy compares performance between InitializedLazy and NewLazy
func BenchmarkInitializedVsNewLazy(b *testing.B) {
	b.Run("InitializedLazy", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			lazy := InitializedLazy[int, error](42)
			val, _ := lazy.Get()
			_ = val
		}
	})

	b.Run("NewLazyPrecomputed", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			lazy := NewLazy(func() (int, error) {
				return 42, nil
			})
			val, _ := lazy.Get()
			_ = val
		}
	})
}

// BenchmarkGoEvaluateLazies measures concurrent evaluation performance
func BenchmarkGoEvaluateLazies(b *testing.B) {
	createLazies := func(count int) []*Lazy[int, error] {
		lazies := make([]*Lazy[int, error], count)
		for i := 0; i < count; i++ {
			idx := i // Capture loop variable
			lazies[i] = NewLazy(func() (int, error) {
				// Simple computation
				return idx * 10, nil
			})
		}
		return lazies
	}

	lazyCountsToTest := []int{2, 5, 10, 50, 100}

	for _, count := range lazyCountsToTest {
		b.Run(fmt.Sprintf("Count-%d", count), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				// Create new lazies for each iteration
				lazies := createLazies(count)

				// Convert to lazyType interface
				lazyInterfaces := make([]lazyType, len(lazies))
				for j, l := range lazies {
					lazyInterfaces[j] = l
				}

				// Evaluate concurrently
				GoEvaluateLazies(lazyInterfaces...)

				// Verify values are computed
				for _, l := range lazies {
					_, _ = l.Get()
				}
			}
		})
	}
}

// BenchmarkLazyWithDelayedComputation simulates real-world scenarios with delays
func BenchmarkLazyWithDelayedComputation(b *testing.B) {
	// Only run a few iterations for this benchmark as it includes sleeps
	if b.N > 10 {
		b.N = 10
	}

	delaysMs := []int{1, 5, 10}

	for _, delay := range delaysMs {
		b.Run(fmt.Sprintf("Delay-%dms", delay), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				// Create lazies with different delays
				lazy1 := NewLazy(func() (int, error) {
					time.Sleep(time.Duration(delay) * time.Millisecond)
					return 1, nil
				})

				lazy2 := NewLazy(func() (int, error) {
					time.Sleep(time.Duration(delay) * time.Millisecond)
					return 2, nil
				})

				lazy3 := NewLazy(func() (int, error) {
					time.Sleep(time.Duration(delay) * time.Millisecond)
					return 3, nil
				})

				// Sequential vs Concurrent execution
				b.StopTimer()
				startSeq := time.Now()
				_, _ = lazy1.Get()
				_, _ = lazy2.Get()
				_, _ = lazy3.Get()
				seqTime := time.Since(startSeq)

				// Reset lazies
				lazy1 = NewLazy(func() (int, error) {
					time.Sleep(time.Duration(delay) * time.Millisecond)
					return 1, nil
				})

				lazy2 = NewLazy(func() (int, error) {
					time.Sleep(time.Duration(delay) * time.Millisecond)
					return 2, nil
				})

				lazy3 = NewLazy(func() (int, error) {
					time.Sleep(time.Duration(delay) * time.Millisecond)
					return 3, nil
				})

				startConcurrent := time.Now()
				GoEvaluateLazies(lazy1, lazy2, lazy3)
				concurrentTime := time.Since(startConcurrent)

				b.StartTimer()

				// Record custom metrics
				b.ReportMetric(float64(seqTime.Milliseconds()), "seq-ms")
				b.ReportMetric(float64(concurrentTime.Milliseconds()), "conc-ms")
				b.ReportMetric(float64(seqTime)/float64(concurrentTime), "speedup")
			}
		})
	}
}

// BenchmarkLazyComplexGraph measures performance of a computation graph similar to the README example
func BenchmarkLazyComplexGraph(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Create counter variables to track function calls
		var counter1, counter2, counter3 int32

		// Layer 1: Base data
		baseData := NewLazy(func() ([]int, error) {
			atomic.AddInt32(&counter1, 1)
			return []int{1, 2, 3, 4, 5}, nil
		})

		// Layer 2: Two separate computations on the base data
		processed1 := NewLazy(func() (int, error) {
			data, err := baseData.Get()
			if err != nil {
				return 0, err
			}

			atomic.AddInt32(&counter2, 1)
			sum := 0
			for _, v := range data {
				sum += v
			}
			return sum, nil
		})

		processed2 := NewLazy(func() (float64, error) {
			data, err := baseData.Get()
			if err != nil {
				return 0, err
			}

			atomic.AddInt32(&counter3, 1)
			sum := 0
			for _, v := range data {
				sum += v
			}
			return float64(sum) / float64(len(data)), nil
		})

		// Layer 3: Combine results
		result := NewLazy(func() (string, error) {
			sum, err := processed1.Get()
			if err != nil {
				return "", err
			}

			avg, err := processed2.Get()
			if err != nil {
				return "", err
			}

			return fmt.Sprintf("Sum: %d, Average: %.2f", sum, avg), nil
		})

		// Execute the computation graph
		val, _ := result.Get()
		_ = val

		// Verify each function was called exactly once
		if atomic.LoadInt32(&counter1) != 1 ||
			atomic.LoadInt32(&counter2) != 1 ||
			atomic.LoadInt32(&counter3) != 1 {
			b.Fatalf("Expected each function to be called once, got: %d, %d, %d",
				atomic.LoadInt32(&counter1),
				atomic.LoadInt32(&counter2),
				atomic.LoadInt32(&counter3))
		}
	}
}
