# Lazy: Simple and Efficient Computation Graphs in Go

Lazy is a lightweight library for constructing and managing computation graphs that allows for clean dependency management, concurrent execution, and integrated caching strategies in Go applications.

## Overview

Lazy provides a simple pattern for managing complex dependency graphs consisting of multiple layers of processing. Each lazy value holds a computation that is only executed when needed and cached after the first evaluation. This allows for:

1. On-demand evaluation
2. Result caching
3. Dependency expression
4. Concurrent execution
5. Integration with external caching systems

## Installation

```go
import "github.com/anish749/lazy"
```

## Core Concepts

### Lazy Values

A `Lazy[T, E]` represents a computation that returns a value of type `T` or an error of type `E`. The computation is only executed when `Get()` is called, and only once.

```go
// Create a lazy computation
lazyValue := lazy.NewLazy(func() (string, error) {
    // Expensive operation here
    return "result", nil
})

// Execute the computation and get the result
result, err := lazyValue.Get()
```

### Initialized Lazy Values

For values that are already computed, you can wrap them in a lazy container:

```go
// Wrap an existing value
initializedLazy := lazy.InitializedLazy[string, error]("pre-computed value")

// No computation happens when Get() is called
value, _ := initializedLazy.Get() // "pre-computed value"
```

### Concurrent Execution

Execute multiple lazy values concurrently:

```go
// Create multiple lazy values
lazy1 := lazy.NewLazy(func() (string, error) {
    // Computation 1
    return "result1", nil
})

lazy2 := lazy.NewLazy(func() (int, error) {
    // Computation 2
    return 42, nil
})

// Run them concurrently
lazy.GoEvaluateLazies(lazy1, lazy2)

// Values are now ready to be retrieved without waiting
value1, _ := lazy1.Get() // Immediate retrieval
value2, _ := lazy2.Get() // Immediate retrieval
```

## Building Computation Graphs

Lazy values can depend on other lazy values, creating a computation graph:

```go
// Base computation
lazyData := lazy.NewLazy(func() ([]int, error) {
    return []int{1, 2, 3, 4, 5}, nil
})

// Dependent computation
lazyProcessed := lazy.NewLazy(func() (int, error) {
    // Get the result of the first computation
    data, err := lazyData.Get()
    if err != nil {
        return 0, err
    }
    
    // Process the data
    sum := 0
    for _, val := range data {
        sum += val
    }
    return sum, nil
})

// The dependency is only resolved when needed
result, _ := lazyProcessed.Get() // 15
```

## Building a complex computation graph with cached computations

Now, let's explore how to build a multi-layer cache system with the help of lazy values, where different computations require different caching strategies, with the goal of minimizing the number of computations and maximizing the reuse of cached results.

Imagine a simple cache interface as below, which can be used to cache parts of a computation graph:

```go
type Cache[T any] interface {
    // GetOrLoadWith retrieves a value from cache, or uses the provided lazy
    // to compute and store the value if not found or stale
    GetOrLoadWith(
        ctx context.Context,
        key string,
        lazyValue *lazy.Lazy[T, error],
    ) (T, error)
}
```

Let's say we have some raw data, and we want to compute insights and a data profile from it.
The dependencies are as follows:

- Raw data -> Processed data -> Insights
- Raw data -> Data profile

Consider that all the 5 computations are computation-heavy, and we want to cache the results of the computations,
in different ways.

Additionally, we only want to compute the final result by evaluating the lazy values concurrently.
We also want to run the computations in concurrent goroutines, based on the dependencies, and wait for them to finish.

This set of computations can be modelled as below:

```go
type Result struct {
    Insights Insights
    DataProfile DataProfile
}

func CreateMultiLayerCachedComputation(
    ctx context.Context,
    rawDataCache Cache[RawData],
    processedDataCache Cache[ProcessedData],
    insightsCache Cache[Insights],
    id string,
) (*Result, error) {
    // Layer 1: Basic data retrieval
    lazyRawData := lazy.NewLazy(func() (RawData, error) {
        return fetchRawData(ctx, id)
    })
    
    // Insert caching at layer 1
    cachedRawData := lazy.NewLazy(func() (RawData, error) {
        return rawDataCache.GetOrLoadWith(ctx, "raw:"+id, lazyRawData)
    })
    
    // Layer 2: Process raw data (computation-heavy)
    lazyProcessedData := lazy.NewLazy(func() (ProcessedData, error) {
        raw, err := cachedRawData.Get()
        if err != nil {
            return ProcessedData{}, err
        }
        return processData(raw)
    })
    
    // Insert caching at layer 2 that caches the processed data.
    cachedProcessedData := lazy.NewLazy(func() (ProcessedData, error) {
        return processedDataCache.GetOrLoadWith(ctx, "processed:"+id, lazyProcessedData)
    })
    
    // Layer 3: Generate insights (very computation-heavy)
    lazyInsights := lazy.NewLazy(func() (Insights, error) {
        processed, err := cachedProcessedData.Get()
        if err != nil {
            return Insights{}, err
        }
        return generateInsights(processed)
    })
    
    // Insert caching at layer 3 that caches the insights.
    cachedInsights := lazy.NewLazy(func() (Insights, error) {
        return insightsCache.GetOrLoadWith(ctx, "insights:"+id, lazyInsights)
    })

    // A second computation that directly depends on the raw data.
    lazyDataProfile := lazy.NewLazy(func() (DataProfile, error) {
        return dataProfileCache.GetOrLoadWith(ctx, "data_profile:"+id, lazyRawData)
    })

    // Insert caching at layer 4 that caches the data profile.
    cachedDataProfile := lazy.NewLazy(func() (DataProfile, error) {
        return dataProfileCache.GetOrLoadWith(ctx, "data_profile:"+id, lazyDataProfile)
    })

    // Evaluate the lazy values concurrently.
    lazy.GoEvaluateLazies(lazyInsights, cachedDataProfile)

    // Handle errors
    insights, err := cachedInsights.Get()
    if err != nil {
        return nil, err
    }

    dataProfile, err := cachedDataProfile.Get()
    if err != nil {
        return nil, err
    }

    // compute the final result by evaluating the lazy value.
    // This will trigger the computation if the cache is stale.
    return &Result{
        Insights: insights,
        DataProfile: dataProfile,
    }, nil
}
```

This approach has several advantages:
This pattern elegantly solves several problems in complex computational systems:

1. **Separation of Concerns**: The computation logic (in the lazy value) is separated from the caching mechanism.
2. **Consistent Error Handling**: Both cache errors and computation errors flow through the same path.
3. **Composability**: Multiple caches can be stacked and composed easily.
4. **Different Cache Types**: Each computation layer is cached using a different cache type, allowing for different caching strategies for different computations.
5. **Partial Computation**: If only the data profile is needed, the expensive insight generation can be skipped, however if both are needed, the results of the raw data fetching is reused.
6. **Reuse of Intermediate Results**: Shared dependencies between multiple computations are only computed once.

## Best Practices

1. **Keep dependencies clear**: When a lazy value depends on other lazy values, make sure the relationship is clear and well-documented.
2. **Error handling**: Always check for errors returned by `Get()` calls.
3. **Concurrent execution**: Use `GoEvaluateLazies` to run independent computations concurrently.
4. **Clean abstraction**: Use lazy values to abstract away complex fetching and caching logic.
5. **Combine with caching**: For maximum efficiency, integrate lazy with a caching system.
