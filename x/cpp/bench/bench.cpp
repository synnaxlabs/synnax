// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cstdlib>
#include <new>

#include "x/cpp/bench/bench.h"

namespace x::bench {
std::atomic<int64_t> alloc_count{0};
std::atomic<int64_t> alloc_bytes{0};
}

// A replacement operator new may not be declared inline, so these live here
// rather than in the header.
void *operator new(const std::size_t n) {
    x::bench::alloc_count.fetch_add(1, std::memory_order_relaxed);
    x::bench::alloc_bytes.fetch_add(static_cast<int64_t>(n), std::memory_order_relaxed);
    if (auto *p = std::malloc(n)) return p;
    throw std::bad_alloc();
}

void *operator new[](const std::size_t n) {
    x::bench::alloc_count.fetch_add(1, std::memory_order_relaxed);
    x::bench::alloc_bytes.fetch_add(static_cast<int64_t>(n), std::memory_order_relaxed);
    if (auto *p = std::malloc(n)) return p;
    throw std::bad_alloc();
}

void operator delete(void *p) noexcept {
    std::free(p);
}
void operator delete[](void *p) noexcept {
    std::free(p);
}
void operator delete(void *p, std::size_t) noexcept {
    std::free(p);
}
void operator delete[](void *p, std::size_t) noexcept {
    std::free(p);
}
