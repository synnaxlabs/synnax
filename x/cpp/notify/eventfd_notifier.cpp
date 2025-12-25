// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <poll.h>
#include <sys/eventfd.h>
#include <unistd.h>

#include <system_error>

#include "x/cpp/notify/notify.h"

namespace notify {

class EventFDNotifier final : public Notifier {
    int event_fd = -1;

public:
    EventFDNotifier() : event_fd(eventfd(0, EFD_NONBLOCK | EFD_CLOEXEC)) {
        if (this->event_fd == -1)
            throw std::system_error(errno, std::system_category(), "eventfd");
    }

    ~EventFDNotifier() override {
        if (this->event_fd != -1) close(this->event_fd);
    }

    EventFDNotifier(const EventFDNotifier&) = delete;
    EventFDNotifier& operator=(const EventFDNotifier&) = delete;
    EventFDNotifier(EventFDNotifier&&) = delete;
    EventFDNotifier& operator=(EventFDNotifier&&) = delete;

    void signal() override {
        const uint64_t val = 1;
        [[maybe_unused]] auto _ = write(this->event_fd, &val, sizeof(val));
    }

    bool wait(const telem::TimeSpan timeout) override {
        pollfd pfd = {this->event_fd, POLLIN, 0};
        const int timeout_ms = (timeout == telem::TimeSpan::MAX())
                                   ? -1
                                   : static_cast<int>(timeout.milliseconds());

        const int result = poll(&pfd, 1, timeout_ms);
        if (result > 0) {
            uint64_t val;
            [[maybe_unused]] auto _ = read(this->event_fd, &val, sizeof(val));
            return true;
        }
        return false;
    }

    bool poll() override {
        uint64_t val;
        return read(this->event_fd, &val, sizeof(val)) == sizeof(val);
    }

    [[nodiscard]] int fd() const override { return this->event_fd; }
};

std::unique_ptr<Notifier> create() { return std::make_unique<EventFDNotifier>(); }

}
