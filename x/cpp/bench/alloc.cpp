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
#ifdef _WIN32
#include <malloc.h>
#endif

#include "x/cpp/bench/alloc.h"

namespace x::bench {
// NOLINTBEGIN(cppcoreguidelines-avoid-non-const-global-variables): the global operator
// new counts into these.
std::atomic<int64_t> alloc_count{0};
std::atomic<int64_t> alloc_bytes{0};
// NOLINTEND(cppcoreguidelines-avoid-non-const-global-variables)
}

// These replace every form of the global operator new and delete. A form left to the
// runtime could free memory that a replaced form allocated, which ASan aborts on.
namespace {
void *allocate(const std::size_t n) noexcept {
    x::bench::alloc_count.fetch_add(1, std::memory_order_relaxed);
    x::bench::alloc_bytes.fetch_add(static_cast<int64_t>(n), std::memory_order_relaxed);
    // operator new(0) must return a unique pointer, but malloc(0) can return null.
    return std::malloc(n == 0 ? 1 : n);
}

void *allocate(const std::size_t n, const std::align_val_t alignment) noexcept {
    x::bench::alloc_count.fetch_add(1, std::memory_order_relaxed);
    x::bench::alloc_bytes.fetch_add(static_cast<int64_t>(n), std::memory_order_relaxed);
    const auto align = static_cast<std::size_t>(alignment);
#ifdef _WIN32
    return _aligned_malloc(n == 0 ? 1 : n, align);
#else
    // aligned_alloc takes only a nonzero multiple of the alignment.
    return std::aligned_alloc(align, n == 0 ? align : (n + align - 1) / align * align);
#endif
}

void release(void *p) noexcept {
    std::free(p);
}

void release(void *p, std::align_val_t) noexcept {
#ifdef _WIN32
    _aligned_free(p);
#else
    std::free(p);
#endif
}

void *or_throw(void *p) {
    if (p == nullptr) throw std::bad_alloc();
    return p;
}
}

void *operator new(const std::size_t n) {
    return or_throw(allocate(n));
}
void *operator new[](const std::size_t n) {
    return or_throw(allocate(n));
}
void *operator new(const std::size_t n, const std::nothrow_t &) noexcept {
    return allocate(n);
}
void *operator new[](const std::size_t n, const std::nothrow_t &) noexcept {
    return allocate(n);
}
void *operator new(const std::size_t n, const std::align_val_t a) {
    return or_throw(allocate(n, a));
}
void *operator new[](const std::size_t n, const std::align_val_t a) {
    return or_throw(allocate(n, a));
}
void *operator new(
    const std::size_t n,
    const std::align_val_t a,
    const std::nothrow_t &
) noexcept {
    return allocate(n, a);
}
void *operator new[](
    const std::size_t n,
    const std::align_val_t a,
    const std::nothrow_t &
) noexcept {
    return allocate(n, a);
}

void operator delete(void *p) noexcept {
    release(p);
}
void operator delete[](void *p) noexcept {
    release(p);
}
void operator delete(void *p, std::size_t) noexcept {
    release(p);
}
void operator delete[](void *p, std::size_t) noexcept {
    release(p);
}
void operator delete(void *p, const std::nothrow_t &) noexcept {
    release(p);
}
void operator delete[](void *p, const std::nothrow_t &) noexcept {
    release(p);
}
void operator delete(void *p, const std::align_val_t a) noexcept {
    release(p, a);
}
void operator delete[](void *p, const std::align_val_t a) noexcept {
    release(p, a);
}
void operator delete(void *p, std::size_t, const std::align_val_t a) noexcept {
    release(p, a);
}
void operator delete[](void *p, std::size_t, const std::align_val_t a) noexcept {
    release(p, a);
}
void operator delete(
    void *p,
    const std::align_val_t a,
    const std::nothrow_t &
) noexcept {
    release(p, a);
}
void operator delete[](
    void *p,
    const std::align_val_t a,
    const std::nothrow_t &
) noexcept {
    release(p, a);
}
