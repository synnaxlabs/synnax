# Task: Add Operator Overloads to telem::Series

## Overview

Add element-wise arithmetic and comparison operator overloads to the `telem::Series` class in `x/cpp/telem/series.h`. Then refactor `arc/cpp/runtime/wasm/bindings.cpp` to use these operators instead of the current manual implementations.

## Background

`telem::Series` is a strongly-typed array of telemetry samples. It currently has in-place scalar operations (`add_inplace`, `multiply_inplace`, etc.) but lacks:
1. Non-mutating scalar operations that return a new Series
2. Series-to-series element-wise operations

The Arc runtime bindings (`arc/cpp/runtime/wasm/bindings.cpp`) currently implement series operations manually with custom loops. These should be moved to `telem::Series` for reusability and testability.

## Files to Modify

1. **`x/cpp/telem/series.h`** - Add operator overloads
2. **`x/cpp/telem/series_test.cpp`** - Add tests (create if doesn't exist, or add to existing test file)
3. **`arc/cpp/runtime/wasm/bindings.cpp`** - Refactor to use new operators

## Design Decisions

### 1. Return New Series (Immutable)

All operators return a new `Series`, they do not mutate:

```cpp
Series result = a + b;  // a and b unchanged
```

### 2. Require Equal Lengths

The `telem::Series` operators should require equal lengths and throw on mismatch. This is the standard element-wise semantic (like NumPy with equal-shaped arrays).

```cpp
if (this->size() != other.size()) {
    throw std::runtime_error("series length mismatch");
}
```

The Arc-specific "last-value repetition" semantic for mismatched lengths stays in the bindings as a wrapper.

### 3. Comparison Operators Return uint8 Series

Comparison operators return a `Series` of `UINT8_T` with values 0 (false) or 1 (true):

```cpp
Series mask = a > b;  // uint8 series with 0s and 1s
```

## API to Implement

### Series-Series Arithmetic

```cpp
Series operator+(const Series& other) const;
Series operator-(const Series& other) const;
Series operator*(const Series& other) const;
Series operator/(const Series& other) const;
```

### Scalar Arithmetic (Series on left)

```cpp
template<typename T> Series operator+(T scalar) const;
template<typename T> Series operator-(T scalar) const;
template<typename T> Series operator*(T scalar) const;
template<typename T> Series operator/(T scalar) const;
```

### Scalar Arithmetic (Scalar on left) - Friend Functions

```cpp
template<typename T>
friend Series operator+(T scalar, const Series& s) { return s + scalar; }

template<typename T>
friend Series operator-(T scalar, const Series& s);  // Note: not commutative

template<typename T>
friend Series operator*(T scalar, const Series& s) { return s * scalar; }

template<typename T>
friend Series operator/(T scalar, const Series& s);  // Note: scalar / series
```

### Series-Series Comparison

```cpp
Series operator>(const Series& other) const;   // Returns UINT8_T series
Series operator<(const Series& other) const;
Series operator>=(const Series& other) const;
Series operator<=(const Series& other) const;
Series operator==(const Series& other) const;
Series operator!=(const Series& other) const;
```

## Implementation Pattern

Use a private helper template similar to the existing `cast_and_apply_numeric_op`:

```cpp
private:
    template<typename Op>
    Series apply_binary_op(const Series& other, Op op) const {
        if (this->size() != other.size()) {
            throw std::runtime_error("series length mismatch for binary operation");
        }

        const auto dt = this->data_type();
        auto result = Series(dt, this->size());
        result.resize(this->size());

        // Dispatch based on data type
        if (dt == FLOAT64_T) {
            apply_binary_op_typed<double>(other, result, op);
        } else if (dt == FLOAT32_T) {
            apply_binary_op_typed<float>(other, result, op);
        }
        // ... etc for all numeric types

        return result;
    }

    template<typename T, typename Op>
    void apply_binary_op_typed(const Series& other, Series& result, Op op) const {
        auto* lhs = reinterpret_cast<const T*>(this->data());
        auto* rhs = reinterpret_cast<const T*>(other.data());
        auto* out = reinterpret_cast<T*>(result.data());
        for (size_t i = 0; i < this->size(); i++) {
            out[i] = op(lhs[i], rhs[i]);
        }
    }
```

For comparisons, the output type is always `UINT8_T`:

```cpp
private:
    template<typename Op>
    Series apply_comparison_op(const Series& other, Op op) const {
        if (this->size() != other.size()) {
            throw std::runtime_error("series length mismatch for comparison");
        }

        auto result = Series(UINT8_T, this->size());
        result.resize(this->size());

        // Dispatch based on this series' data type, output to uint8
        // ...

        return result;
    }
```

## Refactoring the Arc Bindings

After implementing the operators, refactor `arc/cpp/runtime/wasm/bindings.cpp`:

### Before (scalar):
```cpp
uint32_t Bindings::series_element_add_f64(uint32_t handle, double value) {
    auto it = series.find(handle);
    if (it == series.end()) return 0;
    auto result = it->second.deep_copy();
    result.add_inplace(value);
    uint32_t new_handle = series_handle_counter++;
    series.emplace(new_handle, std::move(result));
    return new_handle;
}
```

### After (scalar):
```cpp
uint32_t Bindings::series_element_add_f64(uint32_t handle, double value) {
    auto it = series.find(handle);
    if (it == series.end()) return 0;
    auto result = it->second + value;
    uint32_t new_handle = series_handle_counter++;
    series.emplace(new_handle, std::move(result));
    return new_handle;
}
```

### Before (series-series):
```cpp
uint32_t Bindings::series_series_add_f64(uint32_t a, uint32_t b) {
    auto it_a = series.find(a);
    auto it_b = series.find(b);
    if (it_a == series.end() || it_b == series.end()) return 0;
    auto result = telem::Series(telem::FLOAT64_T, 0);
    series_series_op<double>(
        it_a->second, it_b->second, result,
        [](double x, double y) { return x + y; }
    );
    uint32_t new_handle = series_handle_counter++;
    series.emplace(new_handle, std::move(result));
    return new_handle;
}
```

### After (series-series with length handling):
```cpp
uint32_t Bindings::series_series_add_f64(uint32_t a, uint32_t b) {
    auto it_a = series.find(a);
    auto it_b = series.find(b);
    if (it_a == series.end() || it_b == series.end()) return 0;

    // Arc-specific: extend shorter series with last-value repetition
    auto [lhs, rhs] = extend_to_match_length(it_a->second, it_b->second);
    auto result = lhs + rhs;

    uint32_t new_handle = series_handle_counter++;
    series.emplace(new_handle, std::move(result));
    return new_handle;
}
```

The `extend_to_match_length` helper stays in bindings.cpp (Arc-specific):

```cpp
static std::pair<telem::Series, telem::Series> extend_to_match_length(
    const telem::Series& a,
    const telem::Series& b
) {
    if (a.size() == b.size()) {
        return {a.deep_copy(), b.deep_copy()};
    }
    // Extend shorter series by repeating last value
    // ... implementation ...
}
```

## Testing Requirements

Add tests to `x/cpp/telem/` (check existing test structure):

1. **Arithmetic operators**: Test +, -, *, / for all numeric types
2. **Scalar operators**: Test series + scalar, scalar + series
3. **Comparison operators**: Test >, <, >=, <=, ==, != returning uint8 series
4. **Edge cases**:
   - Empty series
   - Single element series
   - Division by zero (should handle gracefully or throw)
   - Length mismatch (should throw)
   - Type mismatch between series (decide on behavior)

Example test structure:

```cpp
TEST(SeriesOperators, AdditionSameLength) {
    auto a = telem::Series(std::vector<double>{1.0, 2.0, 3.0});
    auto b = telem::Series(std::vector<double>{4.0, 5.0, 6.0});
    auto result = a + b;
    EXPECT_EQ(result.size(), 3);
    EXPECT_DOUBLE_EQ(result.at<double>(0), 5.0);
    EXPECT_DOUBLE_EQ(result.at<double>(1), 7.0);
    EXPECT_DOUBLE_EQ(result.at<double>(2), 9.0);
}

TEST(SeriesOperators, AdditionLengthMismatchThrows) {
    auto a = telem::Series(std::vector<double>{1.0, 2.0, 3.0});
    auto b = telem::Series(std::vector<double>{4.0, 5.0});
    EXPECT_THROW(a + b, std::runtime_error);
}

TEST(SeriesOperators, ComparisonReturnsUint8) {
    auto a = telem::Series(std::vector<double>{1.0, 5.0, 3.0});
    auto b = telem::Series(std::vector<double>{2.0, 3.0, 3.0});
    auto result = a > b;
    EXPECT_EQ(result.data_type(), telem::UINT8_T);
    EXPECT_EQ(result.at<uint8_t>(0), 0);  // 1.0 > 2.0 = false
    EXPECT_EQ(result.at<uint8_t>(1), 1);  // 5.0 > 3.0 = true
    EXPECT_EQ(result.at<uint8_t>(2), 0);  // 3.0 > 3.0 = false
}
```

## Build Commands

```bash
# Build telem library
bazel build //x/cpp/telem:telem

# Run telem tests
bazel test //x/cpp/telem:...

# Build Arc bindings
bazel build //arc/cpp/runtime/wasm:wasm

# Run Arc bindings tests
bazel test //arc/cpp/runtime/wasm:bindings_test
```

## Type Mismatch Behavior

When two series have different data types (e.g., `float32 + int64`), decide on one of:

1. **Throw error** - Safest, requires explicit casting
2. **Promote to larger type** - Like C++ numeric promotion
3. **Use left-hand-side type** - Simple but potentially lossy

Recommendation: **Throw error** for now. Keep it simple and explicit. Users can cast explicitly if needed.

## Summary of Changes

1. Add ~14 operator overloads to `telem::Series` (4 arithmetic + 4 scalar + 6 comparison)
2. Add private helper templates for implementation
3. Add comprehensive tests
4. Refactor Arc bindings to use operators (keep length-extension logic in bindings)
5. Remove the manual `series_series_op` and `series_compare_op` templates from bindings.cpp
