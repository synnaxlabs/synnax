# C++ Development

## C++ Components

C++ code in the monorepo:

- `/driver/` - Real-time hardware integration system (main C++ component)
- `/client/cpp/` - C++ client library
- `/freighter/cpp/` - Transport layer
- `/x/cpp/` - Shared C++ utilities

All C++ components use **Bazel** for building.

## Build System (Bazel)

### Building

```bash
# Build everything
bazel build //...

# Build specific target
bazel build //driver/cmd:driver

# Build with platform flag
bazel build //driver/... --define platform=nilinuxrt

# Run tests
bazel test //driver/...
```

### Cross-Platform Support

Bazel uses `config_setting` and `select()` for platform-specific compilation:

```python
config_setting(
    name = "nilinuxrt",
    values = {"define": "platform=nilinuxrt"},
)

cc_library(
    name = "modbus",
    srcs = select({
        ":nilinuxrt": [],  # Exclude on NI Linux RT
        "//conditions:default": ["modbus.cpp"],
    }),
)
```

### Platform-Specific Patterns

**Compiler Flags:**

```python
copts = select({
    "@platforms//os:windows": [
        "/DWIN32_LEAN_AND_MEAN",
        "/DNOMINMAX",
        "/D_WIN32_WINNT=0x0601",  # Windows 7+
    ],
    "//conditions:default": [],
})
```

**Linker Options:**

```python
linkopts = select({
    "@platforms//os:windows": ["ws2_32.lib", "Iphlpapi.lib"],
    "//conditions:default": [],
})
```

**Source Files:**

```python
srcs = select({
    ":nilinuxrt": ["daemon_nilinuxrt.cpp"],
    "@platforms//os:linux": ["daemon_linux.cpp"],
    "//conditions:default": ["daemon_noop.cpp"],
})
```

## Code Style

### clang-format

Configuration in `.clang-format` at repo root:

- **Based on**: LLVM style
- **Line length**: 88 characters
- **Indentation**: 4 spaces
- **Brace wrapping**: Custom (minimal wrapping)
- **Argument indentation**: Block indentation
- **Constructor initializers**: After colon
- **Include ordering**: System headers (`<...>`) first, then local (`"..."`)

Run formatter:

```bash
clang-format -i file.cpp
```

### Header Organization

```cpp
// System headers first
#include <memory>
#include <string>
#include <vector>

// External libraries
#include "vendor/open62541/open62541.h"

// Internal headers
#include "driver/task/task.h"
#include "x/cpp/telem/telem.h"
```

### Naming Conventions

- **Classes/Structs**: `PascalCase`
- **Functions/Methods**: `camelCase` or `snake_case` (consistent within component)
- **Variables**: `snake_case`
- **Constants**: `UPPER_CASE` or `kPascalCase`
- **Namespaces**: `lowercase`

### Documentation Style

Use Doxygen-style `///` comments with `@brief`, `@param`, and `@returns` tags:

```cpp
/// @brief computes the number of days from the civil date to the Unix epoch.
/// @param date the civil date to convert.
/// @returns the number of days since 1970-01-01.
[[nodiscard]] constexpr int32_t days_from_civil(const Date &date);
```

For struct/class members, use `///<` trailing comments:

```cpp
struct Date {
    uint16_t year; ///< calendar year.
    uint8_t month; ///< month of year [1, 12].
    uint8_t day; ///< day of month [1, 31].
};
```

## Memory Management

### RAII Pattern

All resources use RAII (Resource Acquisition Is Initialization):

```cpp
class Device {
    std::unique_ptr<modbus_t> ctx;

public:
    Device(const Config& cfg) {
        ctx = std::unique_ptr<modbus_t>(
            modbus_new_tcp(cfg.host.c_str(), cfg.port)
        );
        if (modbus_connect(ctx.get()) == -1) {
            throw std::runtime_error("Failed to connect");
        }
    }

    ~Device() {
        if (ctx) modbus_close(ctx.get());
    }
};
```

### Smart Pointers

- `std::unique_ptr<T>` - Exclusive ownership
- `std::shared_ptr<T>` - Shared ownership
- `std::weak_ptr<T>` - Non-owning reference
- Avoid raw pointers except for non-owning references

### Example

```cpp
auto device = std::make_unique<Device>(config);
auto manager = std::make_shared<Manager>(std::move(device));
```

## Testing with Google Test

### Structure

- Test files: `*_test.cpp` co-located with source
- Uses `gtest` framework with custom `xtest` utilities

### Example

```cpp
#include "gtest/gtest.h"
#include "x/cpp/xtest/xtest.h"

TEST(TestSuiteName, TestName) {
    EXPECT_EQ(actual, expected);
    ASSERT_TRUE(condition);
}

// Test fixture
class MyTestFixture : public ::testing::Test {
protected:
    void SetUp() override {
        // Setup
    }

    void TearDown() override {
        // Cleanup
    }
};

TEST_F(MyTestFixture, TestMethod) {
    EXPECT_EQ(value, 42);
}
```

### Custom xtest Utilities

**Eventually Assertions** (for async/concurrent testing):

```cpp
ASSERT_EVENTUALLY_EQ(actual, expected)
ASSERT_EVENTUALLY_GE(actual, expected)
ASSERT_EVENTUALLY_LE(actual, expected)
ASSERT_EVENTUALLY_TRUE(expr)
ASSERT_EVENTUALLY_FALSE(expr)
ASSERT_EVENTUALLY_NIL(error)
```

**Error Handling:**

```cpp
// Success path — use ASSERT_NIL_P to unwrap pair<T, Error> results:
const auto val = ASSERT_NIL_P(some_fn_returning_pair());

// Error path — use ASSERT_OCCURRED_AS_P for pair returns (no structured bindings needed):
ASSERT_OCCURRED_AS_P(some_fn_returning_pair(), EXPECTED_ERR);

// Error path — use ASSERT_OCCURRED_AS for bare Error values:
auto [val, err] = some_fn();
ASSERT_OCCURRED_AS(err, EXPECTED_ERR);

// Always verify the specific error type, not just that an error occurred.
// Prefer ASSERT_OCCURRED_AS / ASSERT_OCCURRED_AS_P over ASSERT_TRUE(err).
```

## Common Patterns

### Dependency Injection

```cpp
class ReadTask {
    std::shared_ptr<synnax::Synnax> client;
    std::unique_ptr<Source> source;

public:
    ReadTask(
        std::shared_ptr<synnax::Synnax> client,
        std::unique_ptr<Source> source
    ) : client(std::move(client)),
        source(std::move(source)) {}
};
```

### Factory Pattern

```cpp
class Factory : public task::Factory {
public:
    std::pair<std::unique_ptr<Task>, bool>
    configure_task(
        const std::shared_ptr<Context>& ctx,
        const synnax::Task& task
    ) override {
        if (task.type != "modbus_read") return {nullptr, false};
        return {std::make_unique<ReadTask>(ctx, task.config), true};
    }
};
```

### Error Handling

```cpp
#include "x/cpp/xerrors/xerrors.h"

auto [result, err] = operation();
if (err) {
    return {nullptr, err.wrap("failed to perform operation")};
}
```

## Platform Considerations

### Windows

- Use `/D` flags for preprocessor defines
- Link with `.lib` files: `ws2_32.lib`, `Iphlpapi.lib`
- Different header paths for some libraries

### Linux

- Standard POSIX APIs
- systemd integration for daemon mode
- NI Linux RT support (subset of Linux)

### macOS

- Similar to Linux
- Some polling workarounds for Tauri limitations

### Cross-Platform Code

```cpp
#ifdef _WIN32
    #include <winsock2.h>
    #include "modbus/modbus.h"
#else
    #include <modbus/modbus.h>
#endif
```

## Common Gotchas

- **Platform-specific code**: Always use Bazel `select()` for platform differences
- **Include paths**: May differ between platforms (use conditional includes)
- **SDKs required**: LabJack LJM, NI-DAQmx must be installed for compilation
- **Modbus on NI Linux RT**: Excluded via Bazel config
- **Smart pointers**: Use `std::move` when transferring ownership
- **Memory leaks**: Run tests with valgrind or ASAN
- **Header guards**: Use `#pragma once` or traditional guards

## Development Best Practices

- **Cross-platform compatibility**: Always consider Windows, macOS, Linux, NI Linux RT
- **RAII**: Use smart pointers and destructors for resource management
- **Const correctness**: Mark methods `const` when they don't modify state
- **Prefer stack allocation**: Use stack when possible, heap when necessary
- **Move semantics**: Use `std::move` for transferring ownership
- **Avoid raw pointers**: Use smart pointers except for non-owning references
- **Platform abstraction**: Use Bazel `select()` instead of `#ifdef` when possible
