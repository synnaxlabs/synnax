# Testing Guide

Synnax uses different testing frameworks for each language, all following BDD-style
organization.

## Cross-Language Patterns

All testing frameworks share these patterns:

- **BDD Style**: Descriptive test organization
- **Fixtures/Setup**: Reusable test setup
- **Async Support**: Native async testing
- **Mocking**: Interface-based or function mocks
- **Co-location**: Tests near source code

## TypeScript Testing (Vitest)

See @docs/claude/toolchains/typescript.md for complete details.

### Quick Reference

```typescript
import { describe, it, expect, vi } from "vitest";

describe("Feature", () => {
  it("should work correctly", () => {
    expect(result).toEqual(expected);
  });
});
```

**Files**: `*.spec.ts` co-located with source

**Key Features**:

- Mocking: `vi.fn()`
- React Testing: `@testing-library/react`
- Assertions: `expect().toEqual()`, `toBe()`, `toHaveBeenCalled()`

## Go Testing (Ginkgo/Gomega)

See @docs/claude/toolchains/go.md for complete details.

### Quick Reference

```go
var _ = Describe("Feature", func() {
    It("Should work correctly", func() {
        Expect(result).To(Equal(expected))
    })
})
```

**Files**: `*_test.go` with suite in `*_suite_test.go`

**Key Features**:

- BDD: `Describe`, `Context`, `It`, `Specify`
- Matchers: `Equal`, `BeTrue`, `Eventually`, `Succeed`
- Lifecycle: `BeforeAll`, `AfterAll`, `BeforeEach`, `AfterEach`
- Helpers: `MustSucceed`, `ShouldNotLeakGoroutines()`

## Python Testing (pytest)

See @docs/claude/toolchains/python.md for complete details.

### Quick Reference

```python
import pytest

@pytest.mark.channel
class TestChannel:
    def test_should_create_channel(self, client):
        channel = client.channels.create(name="test")
        assert channel.name == "test"
```

**Files**: `test_*.py` in `tests/` directory

**Key Features**:

- Fixtures: Defined in `conftest.py`
- Markers: `@pytest.mark.channel`, `@pytest.mark.framer`, etc.
- Exceptions: `with pytest.raises(Exception)`
- Async: `@pytest.mark.asyncio` with pytest-asyncio

## C++ Testing (Google Test)

See @docs/claude/toolchains/cpp.md for complete details.

### Quick Reference

```cpp
#include "gtest/gtest.h"
#include "x/cpp/xtest/xtest.h"

TEST(TestSuite, TestName) {
    EXPECT_EQ(actual, expected);
    ASSERT_TRUE(condition);
}

TEST_F(MyFixture, TestMethod) {
    ASSERT_EVENTUALLY_EQ(async_value, expected);
}
```

**Files**: `*_test.cpp` co-located with source

**Key Features**:

- Assertions: `EXPECT_*`, `ASSERT_*`
- Fixtures: `TEST_F` with `SetUp`/`TearDown`
- Eventually: `ASSERT_EVENTUALLY_*` for async tests
- Errors: `ASSERT_NIL`, `ASSERT_OCCURRED_AS`

## Integration Testing

Location: `/integration/` directory

### Overview

- **Conductor**: Python-based test framework
- **Cross-language**: Tests Go server + TS/Python/C++ clients
- **Full stack**: Validates entire system from client → server → storage
- **Performance**: Built-in benchmarking and stress testing

### Running Integration Tests

```bash
cd integration
pytest
```

### Configuration

- Configurable parameters for stress testing
- Performance measurement
- Multi-node cluster testing
- Custom pytest markers

## Test Organization

### TypeScript

```
package/
├── src/
│   ├── feature/
│   │   ├── feature.ts
│   │   └── feature.spec.ts   ← Tests co-located
```

### Go

```
package/
├── feature.go
├── feature_test.go
└── package_suite_test.go
```

### Python

```
package/
├── src/
│   └── feature/
│       └── feature.py
└── tests/
    ├── conftest.py
    └── test_feature.py
```

### C++

```
package/
├── feature.h
├── feature.cpp
└── feature_test.cpp
```

## Common Testing Patterns

### Async Testing

**TypeScript:**

```typescript
it("should handle async", async () => {
  const result = await asyncOperation();
  expect(result).toBeDefined();
});
```

**Go:**

```go
It("Should handle async", func() {
    Eventually(func() bool {
        return condition()
    }).Should(BeTrue())
})
```

**Python:**

```python
@pytest.mark.asyncio
async def test_async_operation():
    result = await async_operation()
    assert result is not None
```

**C++:**

```cpp
TEST(AsyncTest, EventuallySucceeds) {
    ASSERT_EVENTUALLY_TRUE(async_condition());
}
```

### Mocking

**TypeScript:**

```typescript
const mockFn = vi.fn().mockReturnValue(42);
expect(mockFn()).toBe(42);
```

**Go:**

```go
// Interface-based mocking
type MockReader struct{}
func (m *MockReader) Read(ctx context.Context) (Frame, error) {
    return Frame{}, nil
}
```

**Python:**

```python
# Use fixtures for dependency injection
@pytest.fixture
def mock_client():
    return MockClient()
```

**C++:**

```cpp
// Mock implementations
class MockSource : public Source {
    ReadResult read(breaker::Breaker& b, telem::Frame& fr) override {
        return {0, nil};
    }
};
```

### Setup/Teardown

**TypeScript:**

```typescript
beforeEach(() => {
  // Setup before each test
});

afterAll(() => {
  // Cleanup after all tests
});
```

**Go:**

```go
BeforeEach(func() {
    // Setup
})

AfterAll(func() {
    // Cleanup
})
```

**Python:**

```python
@pytest.fixture
def setup_data():
    # Setup
    yield data
    # Teardown
```

**C++:**

```cpp
class MyTest : public ::testing::Test {
protected:
    void SetUp() override { /* Setup */ }
    void TearDown() override { /* Cleanup */ }
};
```

## Best Practices

### All Languages

- **Descriptive names**: Use "should" or behavior descriptions
- **One assertion per test**: Focus on single behavior
- **AAA pattern**: Arrange, Act, Assert
- **Fast tests**: Keep unit tests under 100ms
- **Deterministic**: No flaky tests, no random data
- **Independent**: Tests don't depend on each other

### TypeScript

- Use `describe` blocks for organization
- Mock external dependencies with `vi.fn()`
- Test React components with `@testing-library/react`
- Avoid testing implementation details

### Go

- Use `Describe`/`Context` for hierarchical organization
- Test exported interfaces, not private implementation
- Use `Eventually` for async assertions
- Check for goroutine leaks with `ShouldNotLeakGoroutines()`

### Python

- Use markers to categorize tests
- Define fixtures in `conftest.py` for reuse
- Add docstrings to test functions
- Use parametrize for table-driven tests

### C++

- Use fixtures for complex setup
- Use `ASSERT_EVENTUALLY_*` for async operations
- Mock via interface implementations
- Check for memory leaks with valgrind/ASAN

## Running Tests

### All Tests

```bash
# TypeScript
pnpm test

# Go (in module directory)
go test ./...

# Python
poetry run pytest

# C++
bazel test //...
```

### Specific Tests

```bash
# TypeScript
pnpm test:console

# Go
go test ./cesium/...

# Python
poetry run pytest -m channel

# C++
bazel test //driver/modbus:modbus_test
```

### With Coverage

```bash
# TypeScript
pnpm test --coverage

# Go
go test -cover ./...

# Python
poetry run pytest --cov

# C++
bazel coverage //...
```
