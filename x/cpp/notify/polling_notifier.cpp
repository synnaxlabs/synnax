// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <atomic>
#include <chrono>
#include <thread>

#include "x/cpp/notify/notify.h"

namespace notify {

class PollingNotifier final : public Notifier {
    std::atomic<bool> signaled{false};
    static constexpr int64_t POLL_INTERVAL_US = 100;

public:
    PollingNotifier() = default;
    ~PollingNotifier() override = default;

    PollingNotifier(const PollingNotifier &) = delete;
    PollingNotifier &operator=(const PollingNotifier &) = delete;
    PollingNotifier(PollingNotifier &&) = delete;
    PollingNotifier &operator=(PollingNotifier &&) = delete;

    void signal() override { this->signaled.store(true, std::memory_order_release); }

    bool wait(const telem::TimeSpan timeout) override {
        if (this->signaled.exchange(false, std::memory_order_acquire)) return true;
        if (timeout == telem::TimeSpan::ZERO()) return false;

        const auto start = std::chrono::steady_clock::now();
        const bool indefinite = (timeout == telem::TimeSpan::max());

        while (true) {
            std::this_thread::sleep_for(std::chrono::microseconds(POLL_INTERVAL_US));
            if (this->signaled.exchange(false, std::memory_order_acquire)) return true;

            if (!indefinite) {
                const auto elapsed = std::chrono::steady_clock::now() - start;
                const auto
                    elapsed_ns = std::chrono::duration_cast<std::chrono::nanoseconds>(
                                     elapsed
                    )
                                     .count();
                if (elapsed_ns >= timeout.nanoseconds()) return false;
            }
        }
    }

    bool poll() override {
        return this->signaled.exchange(false, std::memory_order_acquire);
    }

    [[nodiscard]] int fd() const override { return -1; }

    [[nodiscard]] void *native_handle() const override { return nullptr; }
};

std::unique_ptr<Notifier> create() {
    return std::make_unique<PollingNotifier>();
}

}
