# Lazy Library Benchmarks

This directory contains benchmarks and benchmark-related resources for the Lazy library. The benchmarks are designed to measure various performance characteristics of the library and track them over time.

## Benchmark Categories

The benchmarks are categorized as follows:

### 1. Basic Operations

- `BenchmarkLazyGet`: Measures the performance of creating a lazy value and retrieving its computed result
- `BenchmarkLazyReuse`: Measures the performance benefit of laziness when reusing already computed values

### 2. Computation Depth

- `BenchmarkLazyComputationDepth`: Measures how performance scales with increasing depth of computation chains (dependencies between lazy values)

### 3. Implementation Comparison

- `BenchmarkInitializedVsNewLazy`: Compares performance between pre-initialized lazies via `InitializedLazy` vs. regular lazies via `NewLazy`

### 4. Concurrent Execution

- `BenchmarkGoEvaluateLazies`: Measures the performance of concurrent evaluation of varying numbers of lazy values
- `BenchmarkLazyWithDelayedComputation`: Compares sequential vs. concurrent execution with simulated processing delays

### 5. Complex Scenarios

- `BenchmarkLazyComplexGraph`: Simulates real-world computation graphs with multiple dependencies and shared computations

## Running Benchmarks

To run all benchmarks:

```shell
go test -bench=. -benchmem ./...
```

To run a specific benchmark:

```shell
go test -bench=BenchmarkLazyGet -benchmem ./...
```

## Interpreting Results

Benchmark results include:

- **Operations/sec**: How many operations can be performed per second
- **Bytes/op**: Memory allocated per operation
- **Allocs/op**: Number of distinct memory allocations per operation
- **Custom metrics**: For some benchmarks, custom metrics like sequential vs. concurrent execution time are reported

## Continuous Integration

Benchmarks are run weekly via GitHub Actions (or manually triggered). Results are stored and visualized to track performance over time.

The benchmark visualization can be found on the project's GitHub Pages site.

## Adding New Benchmarks

When adding new benchmarks:

1. Add benchmark functions to `lazy_bench_test.go`
2. Follow the naming convention of `BenchmarkX` where X describes what you're measuring
3. Use `b.Run` for sub-benchmarks to test different parameters or scenarios
4. Use `b.ReportAllocs()` to track memory allocations
5. Use `b.ResetTimer()` before the part you want to measure
6. For custom metrics, use `b.ReportMetric(value, name)`
