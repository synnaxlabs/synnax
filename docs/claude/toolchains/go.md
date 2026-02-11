# Go Development

## Go Modules

Individual Go modules in the monorepo:

- `/core/` - Synnax server (4-layer architecture)
- `/aspen/` - Distributed key-value store and cluster management
- `/cesium/` - Time-series database engine
- `/freighter/go/` - Transport layer (gRPC, HTTP, WebSocket)
- `/arc/` - Arc programming language compiler
- `/alamos/go/` - Instrumentation and observability
- `/x/go/` - Shared Go utilities

### Go Workspace

The repo uses a `go.work` file to manage local module replacements:

```go
go 1.26

use (
    ./alamos/go
    ./arc/go
    ./aspen
    ./cesium
    ./core
    ./freighter/go
    ./freighter/integration
    ./x/go
)
```

This allows modules to reference each other without publishing.

## Development Commands

### Building

```bash
cd core && go build ./...
cd cesium && go build ./...
```

### Testing

**IMPORTANT**: Always use `ginkgo` to run tests, not `go test`. The codebase uses
Ginkgo/Gomega for testing.

```bash
# Run all tests in a package
cd core/pkg/distribution/channel && ginkgo

# Run tests with verbose output
cd cesium && ginkgo -v

# Run specific tests by focus
cd core/pkg/distribution/channel && ginkgo --focus "Name Validation"

# Run all tests recursively
cd core && ginkgo -r
```

### Integration Tests

- Integration tests located in `/integration/` directory
- Python-based conductor framework
- Run with `pytest` from integration directory

## Code Style

- **Formatter**: `gofmt` (default Go formatting)
- **Conventions**: Standard Go idioms, interfaces for abstraction
- **Line length**: 88 characters (configured in editor, not enforced by gofmt)
- **Imports**: Group standard library, external, and internal packages
- **Error handling**: Explicit error returns, wrapped with context

### Go Patterns

- **Interfaces for abstraction**: Define small, focused interfaces
- **Dependency injection**: Pass dependencies as parameters, not globals
- **Context propagation**: Use `context.Context` for cancellation and values
- **Structured concurrency**: Use goroutines with proper synchronization
- **Table-driven tests**: Parameterize tests with test cases

## Testing with Ginkgo/Gomega

### Structure

- Suite files: `*_suite_test.go`
- Test files: `*_test.go`
- Package naming: `package_name_test` for blackbox testing

### Example

```go
package cesium_test

import (
    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
    "testing"
)

func TestCesium(t *testing.T) {
    RegisterFailHandler(Fail)
    RunSpecs(t, "Cesium Suite")
}

var _ = Describe("Feature Behavior", func() {
    Context("FS: "+fsName, Ordered, func() {
        BeforeAll(func() {
            // Setup
        })

        AfterAll(func() {
            // Teardown
        })

        Describe("Sub-feature", func() {
            It("Should do something", func() {
                Expect(result).To(Equal(expected))
            })

            Specify("Specific behavior", func() {
                Expect(value).To(BeTrue())
            })
        })
    })
})
```

### Key Features

- **BDD Structure**: `Describe`, `Context`, `It`, `Specify` for organizing tests
- **Lifecycle Hooks**: `BeforeAll`, `AfterAll`, `BeforeEach`, `AfterEach`,
  `JustBeforeEach`
- **Matchers**: Rich Gomega matchers (`Equal`, `BeTrue`, `Succeed`, `HaveLen`,
  `MatchError`)
- **Custom Matchers**: Domain-specific matchers (e.g., `telem.MatchSeriesDataV`)
- **Ordering**: `Ordered` decorator for sequential test execution
- **Async Support**: `Eventually` matcher for polling assertions

### Common Patterns

- Parameterized tests using loops over file systems or configurations
- Table-driven tests via loops with test cases
- Context-based test organization for different scenarios
- Helper functions for database/service setup in suite files
- Goroutine leak detection: `ShouldNotLeakGoroutines()`

### Helpful Utilities

- `MustSucceed(result, err)` - Unwrap result or fail
- `Eventually(func() bool).Should(BeTrue())` - Poll until condition met
- `Consistently(func() bool).Should(BeTrue())` - Assert condition stays true

## 4-Layer Architecture (Server)

The Synnax server (`/core/`) follows strict layering:

```
Interface Layer (HTTP/gRPC APIs)
         ↓
Service Layer (Business logic)
         ↓
Distribution Layer (Aspen clustering)
         ↓
Storage Layer (Cesium + Pebble)
```

**Rules:**

- Dependencies only flow downward
- Each layer exposes interfaces, not implementations
- Services use dependency injection for testability

## Common Patterns

### Dependency Injection

```go
type Service struct {
    db     *DB
    client *Client
}

func New(db *DB, client *Client) *Service {
    return &Service{db: db, client: client}
}
```

### Interface Segregation

```go
// Small, focused interfaces
type Reader interface {
    Read(ctx context.Context) (Frame, error)
}

type Writer interface {
    Write(ctx context.Context, frame Frame) error
}
```

### Error Wrapping

```go
import "github.com/cockroachdb/errors"

if err != nil {
    return errors.Wrap(err, "failed to process frame")
}
```

## Common Gotchas

- **Aspen**: Eventual consistency means metadata updates may take up to 1 second to
  propagate
- **Cesium**: Requires careful handling of overlapping time ranges to prevent write
  conflicts
- **Context**: Always pass `context.Context` as first parameter to functions
- **Goroutines**: Use `sync.WaitGroup` or `errgroup.Group` for proper cleanup
- **Testing**: Ensure tests don't leak goroutines using `ShouldNotLeakGoroutines()`

## Development Best Practices

- **Interfaces over concrete types**: Define interfaces for dependencies
- **Dependency injection**: Pass dependencies explicitly, avoid global state
- **Context propagation**: Use `context.Context` for cancellation and request-scoped
  values
- **Error wrapping**: Add context to errors using `errors.Wrap`
- **Table-driven tests**: Parameterize tests with slices of test cases
- **Goroutine safety**: Use mutexes or channels for shared state
- **Clean shutdown**: Implement graceful shutdown with context cancellation
