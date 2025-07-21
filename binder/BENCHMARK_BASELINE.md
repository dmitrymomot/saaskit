# Binder Package Baseline Performance

## System Information
- CPU: Apple M3 Max
- OS: Darwin (macOS)
- Architecture: arm64
- Go Version: (inferred from output)

## File Binding Benchmarks

### Single File Upload
- **Performance**: 5,772 ns/op
- **Memory**: 17,214 B/op
- **Allocations**: 57 allocs/op

### Multiple Files (10 files)
- **Performance**: 28,291 ns/op
- **Memory**: 73,436 B/op
- **Allocations**: 303 allocs/op

### Large Struct (50+ fields)
- **Performance**: 7,965 ns/op
- **Memory**: 18,047 B/op
- **Allocations**: 57 allocs/op

### Reflection Overhead Only
- **Performance**: 3,864 ns/op
- **Memory**: 11,034 B/op
- **Allocations**: 28 allocs/op

### File Size Impact
- 1KB: 5,685 ns/op (17,199 B/op, 56 allocs)
- 10KB: 17,099 ns/op (91,488 B/op, 67 allocs)
- 100KB: 121,135 ns/op (790,360 B/op, 80 allocs)
- 1MB: 1,062,102 ns/op (9,456,900 B/op, 102 allocs)

## Reflection Binding Benchmarks

### Small Struct (5 fields)
- **Performance**: 2,021 ns/op
- **Memory**: 5,857 B/op
- **Allocations**: 22 allocs/op

### Large Struct (50 fields)
- **Performance**: 13,930 ns/op
- **Memory**: 15,546 B/op
- **Allocations**: 120 allocs/op

### Mixed Types Struct
- **Performance**: 32,935 ns/op
- **Memory**: 52,840 B/op
- **Allocations**: 425 allocs/op

## Helper Function Benchmarks

### GetFile
- **Performance**: 5,491 ns/op
- **Memory**: 17,088 B/op
- **Allocations**: 55 allocs/op

### GetFiles (5 files)
- **Performance**: 14,979 ns/op
- **Memory**: 40,540 B/op
- **Allocations**: 163 allocs/op

## Key Observations

1. **Reflection overhead**: ~3.8μs for empty struct with 10 fields
2. **Per-field cost**: ~280ns per field in large structs
3. **File I/O dominates**: Memory allocation scales linearly with file size
4. **Allocation count**: Relatively high for small operations (28-57 allocs)

## Optimization Opportunities

1. **Reflection caching**: Could save ~3-4μs per request by caching struct metadata
2. **Memory pooling**: Reduce allocations for FileUpload structs
3. **Lazy parsing**: Defer multipart parsing until actually needed
4. **Field lookup optimization**: Cache field names and tags

## Performance Goals

After optimization, we should aim for:
- Single file binding: < 3,000 ns/op (50% improvement)
- Large struct: < 5,000 ns/op (40% improvement)
- Allocations: < 30 allocs/op (50% reduction)