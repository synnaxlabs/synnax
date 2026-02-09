// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// Pipe-based notifier for macOS.
///
/// Why not kqueue EVFILT_USER?
/// EVFILT_USER events are internal to a single kqueue - they cannot be watched
/// from another kqueue via EVFILT_READ. Since the runtime's event loop has its
/// own kqueue that watches notifier fds, we need an fd that becomes readable
/// when signaled. A pipe provides this: write to one end, read-end becomes readable.
///
/// This matches Linux's eventfd semantics and preserves soft RT guarantees:
/// - No userspace mutexes (kernel-managed buffer)
/// - O(1) bounded latency for signal/wait
/// - No memory allocation (fixed kernel buffer, 1-byte writes)

#include <system_error>

#include <fcntl.h>
#include <poll.h>
#include <unistd.h>

#include "x/cpp/notify/notify.h"

namespace x::notify {

class PipeNotifier final : public Notifier {
    int read_fd = -1;
    int write_fd = -1;

public:
    PipeNotifier() {
        int fds[2];
        if (pipe(fds) == -1)
            throw std::system_error(errno, std::system_category(), "pipe");

        this->read_fd = fds[0];
        this->write_fd = fds[1];

        fcntl(this->read_fd, F_SETFL, O_NONBLOCK);
        fcntl(this->write_fd, F_SETFL, O_NONBLOCK);
        fcntl(this->read_fd, F_SETFD, FD_CLOEXEC);
        fcntl(this->write_fd, F_SETFD, FD_CLOEXEC);
    }

    ~PipeNotifier() override {
        if (this->read_fd != -1) close(this->read_fd);
        if (this->write_fd != -1) close(this->write_fd);
    }

    PipeNotifier(const PipeNotifier &) = delete;
    PipeNotifier &operator=(const PipeNotifier &) = delete;
    PipeNotifier(PipeNotifier &&) = delete;
    PipeNotifier &operator=(PipeNotifier &&) = delete;

    void signal() override {
        const char byte = 1;
        [[maybe_unused]] auto _ = write(this->write_fd, &byte, 1);
    }

    bool wait(const telem::TimeSpan timeout) override {
        if (this->drain()) return true;

        pollfd pfd = {this->read_fd, POLLIN, 0};
        const int timeout_ms = (timeout == telem::TimeSpan::max())
                                 ? -1
                                 : static_cast<int>(timeout.milliseconds());

        if (::poll(&pfd, 1, timeout_ms) > 0) {
            this->drain();
            return true;
        }
        return false;
    }

    bool poll() override { return this->drain(); }

    [[nodiscard]] int fd() const override { return this->read_fd; }

    [[nodiscard]] void *native_handle() const override { return nullptr; }

private:
    bool drain() {
        char buf[64];
        bool drained = false;
        while (read(this->read_fd, buf, sizeof(buf)) > 0)
            drained = true;
        return drained;
    }
};

std::unique_ptr<Notifier> create() {
    return std::make_unique<PipeNotifier>();
}

}
