// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <atomic>
#include <cstddef>
#include <memory>
#include <new>

#include "x/cpp/notify/notify.h"

namespace x::queue {
template<typename T>
class SPSC {
    class RingBuffer {
        struct alignas(64) {
            std::atomic<size_t> value{0};
        } head;

        struct alignas(64) {
            std::atomic<size_t> value{0};
        } tail;

        struct Slot {
            alignas(T) std::byte storage[sizeof(T)];

            T *ptr() { return reinterpret_cast<T *>(storage); }

            const T *ptr() const { return reinterpret_cast<const T *>(storage); }
        };

        size_t cap;
        size_t mask;
        std::unique_ptr<Slot[]> buffer;

        static size_t next_power_of_2(size_t n) {
            if (n == 0) return 1;
            n--;
            n |= n >> 1;
            n |= n >> 2;
            n |= n >> 4;
            n |= n >> 8;
            n |= n >> 16;
            if constexpr (sizeof(size_t) > 4) { n |= n >> 32; }
            return n + 1;
        }

    public:
        explicit RingBuffer(const size_t capacity = 1024):
            cap(next_power_of_2(capacity + 1)),
            mask(cap - 1),
            buffer(std::make_unique<Slot[]>(cap)) {}

        ~RingBuffer() {
            T value;
            while (this->try_pop(value)) {}
        }

        RingBuffer(const RingBuffer &) = delete;
        RingBuffer &operator=(const RingBuffer &) = delete;
        RingBuffer(RingBuffer &&) = delete;
        RingBuffer &operator=(RingBuffer &&) = delete;

        bool try_push(T value) {
            const size_t h = this->head.value.load(std::memory_order_relaxed);
            const size_t next = (h + 1) & this->mask;
            if (next == this->tail.value.load(std::memory_order_acquire)) return false;
            new (this->buffer[h].ptr()) T(std::move(value));
            this->head.value.store(next, std::memory_order_release);
            return true;
        }

        bool try_pop(T &value) {
            const size_t t = this->tail.value.load(std::memory_order_relaxed);
            if (t == this->head.value.load(std::memory_order_acquire)) return false;
            T *ptr = this->buffer[t].ptr();
            value = std::move(*ptr);
            ptr->~T();
            this->tail.value.store((t + 1) & this->mask, std::memory_order_release);
            return true;
        }

        [[nodiscard]] bool empty() const {
            return this->tail.value.load(std::memory_order_acquire) ==
                   this->head.value.load(std::memory_order_acquire);
        }

        [[nodiscard]] size_t size() const {
            const size_t h = this->head.value.load(std::memory_order_acquire);
            const size_t t = this->tail.value.load(std::memory_order_acquire);
            return (h - t) & this->mask;
        }

        [[nodiscard]] size_t capacity() const { return this->cap - 1; }
    };

    RingBuffer buffer;
    std::unique_ptr<notify::Notifier> notif;
    std::atomic<bool> is_closed{false};

public:
    struct Config {
        size_t capacity = 1024;
        std::unique_ptr<notify::Notifier> notifier = nullptr;
    };

    SPSC(): SPSC(Config{}) {}

    explicit SPSC(const size_t capacity): SPSC(Config{.capacity = capacity}) {}

    explicit SPSC(Config config):
        buffer(config.capacity),
        notif(config.notifier ? std::move(config.notifier) : notify::create()) {}

    SPSC(const SPSC &) = delete;
    SPSC &operator=(const SPSC &) = delete;
    SPSC(SPSC &&) = delete;
    SPSC &operator=(SPSC &&) = delete;

    bool push(T value) {
        if (this->is_closed.load(std::memory_order_acquire)) return false;
        if (!this->buffer.try_push(std::move(value))) return false;
        this->notif->signal();
        return true;
    }

    bool pop(T &value) {
        while (true) {
            if (this->buffer.try_pop(value)) return true;
            if (this->is_closed.load(std::memory_order_acquire)) {
                return this->buffer.try_pop(value);
            }
            this->notif->wait();
        }
    }

    bool try_pop(T &value) { return this->buffer.try_pop(value); }

    [[nodiscard]] bool empty() const { return this->buffer.empty(); }

    [[nodiscard]] size_t size() const { return this->buffer.size(); }

    [[nodiscard]] size_t capacity() const { return this->buffer.capacity(); }

    void close() {
        this->is_closed.store(true, std::memory_order_release);
        this->notif->signal();
    }

    [[nodiscard]] bool closed() const {
        return this->is_closed.load(std::memory_order_acquire);
    }

    [[nodiscard]] notify::Notifier &notifier() { return *this->notif; }
};

}
