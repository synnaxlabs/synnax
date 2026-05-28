// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// A tiny manual harness for eyeballing real crash-handler output. Install the handler,
// then trigger a crash of the requested kind:
//
//   bazel run //x/cpp/crash:crash_demo -- segfault   # null dereference (SIGSEGV)
//   bazel run //x/cpp/crash:crash_demo -- abort      # std::abort (SIGABRT)
//   bazel run //x/cpp/crash:crash_demo -- throw      # unhandled C++ exception
//   bazel run //x/cpp/crash:crash_demo -- overflow   # stack overflow

#include <cstdlib>
#include <iostream>
#include <stdexcept>
#include <string>

#include "x/cpp/crash/crash.h"

namespace {
[[noreturn]] void usage(const char *prog) {
    std::cerr << "usage: " << prog << " <segfault|abort|throw|overflow>\n";
    std::exit(2);
}

/// @brief never zero at runtime, but opaque to the optimizer so the recursion below is
/// not flagged as unconditional (-Winfinite-recursion) and is not turned into a loop.
volatile int keep_recursing = 1;

int overflow(volatile char *prev) {
    volatile char buf[8192];
    buf[0] = prev == nullptr ? 1 : prev[0];
    buf[sizeof(buf) - 1] = buf[0];
    if (!keep_recursing) return buf[0];
    return overflow(buf) + buf[sizeof(buf) - 1];
}
}

int main(const int argc, char **argv) {
    x::crash::install("crash-demo");
    if (argc < 2) usage(argv[0]);
    const std::string mode = argv[1];
    if (mode == "segfault") {
        volatile int *p = nullptr;
        *p = 1;
    } else if (mode == "abort") {
        std::abort();
    } else if (mode == "throw") {
        throw std::runtime_error("demo unhandled exception");
    } else if (mode == "overflow") {
        volatile int sink = overflow(nullptr);
        static_cast<void>(sink);
    } else {
        usage(argv[0]);
    }
    return 0;
}
