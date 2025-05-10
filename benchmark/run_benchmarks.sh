#!/bin/bash

# Set script to exit immediately if any command fails
set -e

# Default output file
OUTPUT_FILE="benchmark-result.txt"

# Parse command line arguments
while getopts "o:" opt; do
    case $opt in
    o) OUTPUT_FILE="$OPTARG" ;;
    *) ;;
    esac
done

echo "Running benchmarks for go-lazy library..."
echo "Results will be saved to: $OUTPUT_FILE"

# Run all benchmarks and save to the output file
# -run=^$ ensures no tests are run, only benchmarks
go test -bench=. -benchmem -run=^$ ./... >"$OUTPUT_FILE"

echo "Benchmarks completed. Results saved to $OUTPUT_FILE"
echo "Summary of benchmarks run:"
grep -E "Benchmark" "$OUTPUT_FILE" | wc -l | xargs echo "Total benchmarks:"

# Print the first few lines of results for quick review
echo -e "\nBenchmark preview:"
head -n 20 "$OUTPUT_FILE"
echo "..."

exit 0
