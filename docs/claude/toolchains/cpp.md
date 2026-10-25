# C++ development

## Components

`/driver/` (real-time hardware integration — the main C++ component), `/client/cpp/`,
`/freighter/cpp/`, `/x/cpp/` (shared utilities). All built with Bazel.

## Bazel

```bash
bazel build //...               # everything
bazel build //driver/cmd:driver # specific target
bazel build //driver/... --define platform=nilinuxrt
bazel test //driver/...
```

Platform-specific code uses `config_setting` + `select()` — prefer it over `#ifdef` —
for sources (`daemon_linux.cpp` vs `daemon_noop.cpp`), deps (Modbus excluded on NI Linux
Real-Time), copts (`/DWIN32_LEAN_AND_MEAN`, `/DNOMINMAX`, `/D_WIN32_WINNT=0x0601` on
Windows), and linkopts (`ws2_32.lib`, `Iphlpapi.lib`).

## Style

clang-format (repo-root `.clang-format`: LLVM base, 88-char lines, 4-space indent).
Format with `scripts/clang_format.sh <path>`, check with
`scripts/check_clang_format.sh <path>` (see `scripts/CLAUDE.md`); install the pinned
version via `scripts/install_clang_format.sh`, never a plain package-manager install.
Include order: system `<...>`, then vendor, then internal `"..."`. Naming: `PascalCase`
types, `camelCase`/`snake_case` functions (consistent within a component), `snake_case`
variables, `UPPER_CASE`/`kPascalCase` constants, lowercase namespaces.

🚨 **NO NAMESPACE TERMINATION COMMENTS.** Never `} // namespace foo` — bare closing
brace only. Claude sessions add these constantly. Delete on sight when editing a file
that has them.

Docs: Doxygen `///` with `@brief`/`@param`/`@returns` on public functions (`@` tags,
never `\brief`); `/// @brief` above each public struct/class member. A private member
gets one only when its name cannot carry the meaning. The universal body-comment and
doc-comment rules in the root CLAUDE.md apply.

```cpp
/// @brief computes the number of days from the civil date to the Unix epoch.
/// @param date the civil date to convert.
/// @returns the number of days since 1970-01-01.
[[nodiscard]] constexpr int32_t days_from_civil(const Date &date);
```

## Linting

clang-tidy (repo-root `.clang-tidy`) runs as a Bazel aspect. Each source and header is
one cached action, and any finding fails the build. CI lints each component on Ubuntu.
Bazel fetches the clang-tidy that `MODULE.bazel` pins, so linting needs no install. Pin
clang-tidy and clang-format (`scripts/install_clang_format.sh`) to one LLVM version, and
change both in one pull request that fixes the new findings.

```bash
bazel build --config=clang-tidy //driver/... # lint
scripts/clang_tidy_fix.sh //driver/...       # apply the fixes clang-tidy offers
```

The aspect lints each header as its own file, so a header must include, and its target
must depend on, everything it uses. Suppress one finding with
`// NOLINTNEXTLINE(<check>): <reason>` above its line, never a bare `NOLINT`. The aspect
skips a file whose header says `Code generated`. A target that holds vendor SDK headers,
or embeds a large generated table, takes `tags = ["no-clang-tidy"]`. In a test, read an
optional with `.value()`: the lint cannot see that `ASSERT_TRUE(opt.has_value())` makes
`*opt` safe.

## Memory

RAII for all resources. `std::unique_ptr` (exclusive), `std::shared_ptr` (shared),
`std::weak_ptr` (non-owning); raw pointers only as non-owning references. `std::move`
when transferring ownership. Const-correct methods; prefer stack allocation. Dependency
injection via constructor parameters.

## Errors

```cpp
#include "x/cpp/xerrors/xerrors.h"

auto [result, err] = operation();
if (err) return {nullptr, err.wrap("failed to perform operation")};
```

## Testing (Google Test + xtest)

`*_test.cpp` co-located with source. `TEST` / `TEST_F` with `SetUp`/`TearDown` fixtures;
mock via interface implementations. xtest utilities from `x/cpp/xtest/xtest.h`:

- Async assertions: `ASSERT_EVENTUALLY_EQ/GE/LE/TRUE/FALSE/NIL`.
- `pair<T, Error>` success path: `const auto val = ASSERT_NIL_P(fn());`
- `pair<T, Error>` error path: `ASSERT_OCCURRED_AS_P(fn(), EXPECTED_ERR);` — never
  structured bindings + `ASSERT_OCCURRED_AS`, which is only for bare `Error` values.
- Always verify the specific error type, never just that an error occurred.

## Gotchas

- SDKs required to compile: LabJack LJM, NI-DAQmx.
- Modbus is excluded on NI Linux Real-Time via Bazel config.
- Include paths differ per platform — conditional includes where unavoidable.
- Check leaks with valgrind/ASAN. `#pragma once` or traditional guards.
- Target platforms: Windows, macOS, Linux, NI Linux Real-Time.
