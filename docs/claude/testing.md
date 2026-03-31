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

- **Test Conductor**: Custom Python-based test orchestration framework
- **Playwright**: Console UI tests use Playwright for browser automation
- **Cross-language**: Tests Go server + TS/Python/C++ clients
- **Full stack**: Validates entire system from client → server → storage
- **Performance**: Built-in benchmarking and stress testing

### Build & Run Sequence

Integration tests require a running Synnax server with the embedded Console. The full
build sequence is:

```bash
# 1. Build Console web assets
pnpm build:console-vite

# 2. Copy assets into Core's embed directory
cp -r console/dist/* core/pkg/console/dist/

# 3. Build Go Core server with embedded Console
cd core && go build -tags=console -ldflags="-w -s" -o synnax . && cd ..

# 4. Start the server (in-memory mode for testing)
mkdir -p ~/synnax-data && cd ~/synnax-data
./path/to/synnax start -mi &

# 5. Run tests via test conductor
cd integration
uv run tc console              # all console tests
uv run tc console/.../label    # filter by case name
uv run tc driver/modbus        # driver tests matching "modbus"
```

The `-tags=console` build tag activates `core/pkg/console/enabled.go`, which uses
`//go:embed all:dist` to embed the Console assets into the binary.

### Test Conductor CLI (`tc`)

The test conductor (`uv run tc`) supports flexible target filtering:

```bash
# 1-part: run all tests in a file
uv run tc console                    # all console_tests.json

# 2-part: file + case substring filter
uv run tc console/label              # cases matching "label"

# 3-part: file + sequence + case filter
uv run tc console/channel/calc       # sequence "channel", cases "calc"

# Global filter across all test files
uv run tc -f modbus                  # all files, cases matching "modbus"

# Options
uv run tc console --headed           # run Playwright in headed mode
uv run tc driver -d my_rack          # specify driver rack name
```

### Test Organization

Tests are defined in JSON sequence files (`*_tests.json`) in `/integration/tests/`:

- `console_tests.json` - Console UI tests (Playwright-based)
- `driver_tests.json` - Hardware driver tests
- `arc_tests.json` - Arc language tests
- `client_tests.json` - Client library tests

Each JSON file defines sequences of test cases that can run sequentially or
asynchronously with configurable pool sizes.

### Console Test Helpers

Console integration tests use helper classes in `/integration/console/`:

- `layout.py` - Page navigation, toolbar management, keyboard interactions
- `context_menu.py` - Context menu opening and option selection
- `ranges.py`, `tasks.py`, `statuses.py` - Resource-specific helpers
- `case.py` - Base `ConsoleCase` class with Playwright setup

### Environment Dependencies

Some test suites require external services:

- **Console tests**: Only require the Synnax server (most tests)
- **Driver tests**: Require hardware simulators (OPC UA, Modbus, etc.)
- **Task lifecycle tests**: Require OPC UA simulator (`OPCUASim`)
- **NI form tests**: May fail with notification overlays if no driver is connected

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
    ReadResult read(breaker::Breaker& b, synnax::Frame& fr) override {
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
uv run pytest

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
uv run pytest -m channel

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
uv run pytest --cov

# C++
bazel coverage //...
```
