// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <sys/event.h>
#include <sys/time.h>
#include <unistd.h>

#include <system_error>

#include "x/cpp/notify/notify.h"

namespace notify {

class KQueueNotifier final : public Notifier {
    int kq = -1;
    static constexpr uintptr_t IDENT = 1;

public:
    KQueueNotifier() : kq(kqueue()) {
        if (this->kq == -1)
            throw std::system_error(errno, std::system_category(), "kqueue");

        struct kevent kev {};
        EV_SET(&kev, IDENT, EVFILT_USER, EV_ADD | EV_CLEAR, 0, 0, nullptr);
        if (kevent(this->kq, &kev, 1, nullptr, 0, nullptr) == -1) {
            close(this->kq);
            throw std::system_error(errno, std::system_category(), "kevent register");
        }
    }

    ~KQueueNotifier() override {
        if (this->kq != -1) close(this->kq);
    }

    KQueueNotifier(const KQueueNotifier&) = delete;
    KQueueNotifier& operator=(const KQueueNotifier&) = delete;
    KQueueNotifier(KQueueNotifier&&) = delete;
    KQueueNotifier& operator=(KQueueNotifier&&) = delete;

    void signal() override {
        struct kevent kev {};
        EV_SET(&kev, IDENT, EVFILT_USER, 0, NOTE_TRIGGER, 0, nullptr);
        kevent(this->kq, &kev, 1, nullptr, 0, nullptr);
    }

    bool wait(const telem::TimeSpan timeout) override {
        struct kevent kev {};
        timespec ts {};
        timespec* ts_ptr = nullptr;

        if (timeout != telem::TimeSpan::MAX()) {
            ts.tv_sec = timeout.seconds();
            ts.tv_nsec = timeout.nanoseconds() % 1000000000LL;
            ts_ptr = &ts;
        }

        return kevent(this->kq, nullptr, 0, &kev, 1, ts_ptr) > 0;
    }

    bool poll() override {
        struct kevent kev {};
        timespec ts = {0, 0};
        return kevent(this->kq, nullptr, 0, &kev, 1, &ts) > 0;
    }

    [[nodiscard]] int fd() const override { return this->kq; }
};

std::unique_ptr<Notifier> create() { return std::make_unique<KQueueNotifier>(); }

}
