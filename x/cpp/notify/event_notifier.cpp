// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#ifndef WIN32_LEAN_AND_MEAN
#define WIN32_LEAN_AND_MEAN
#endif
#ifndef NOMINMAX
#define NOMINMAX
#endif

#include <system_error>

#include <windows.h>

#include "x/cpp/notify/notify.h"

namespace notify {

class WindowsEventNotifier final : public Notifier {
    HANDLE event_handle = nullptr;

public:
    WindowsEventNotifier(): event_handle(CreateEvent(nullptr, FALSE, FALSE, nullptr)) {
        if (this->event_handle == nullptr)
            throw std::system_error(
                static_cast<int>(GetLastError()),
                std::system_category(),
                "CreateEvent"
            );
    }

    ~WindowsEventNotifier() override {
        if (this->event_handle != nullptr) CloseHandle(this->event_handle);
    }

    WindowsEventNotifier(const WindowsEventNotifier &) = delete;
    WindowsEventNotifier &operator=(const WindowsEventNotifier &) = delete;
    WindowsEventNotifier(WindowsEventNotifier &&) = delete;
    WindowsEventNotifier &operator=(WindowsEventNotifier &&) = delete;

    void signal() override { SetEvent(this->event_handle); }

    bool wait(const telem::TimeSpan timeout) override {
        DWORD timeout_ms;
        if (timeout == telem::TimeSpan::MAX()) {
            timeout_ms = INFINITE;
        } else {
            const auto ms = timeout.milliseconds();
            timeout_ms = (ms > static_cast<double>(INFINITE - 1))
                           ? (INFINITE - 1)
                           : static_cast<DWORD>(ms);
        }
        return WaitForSingleObject(this->event_handle, timeout_ms) == WAIT_OBJECT_0;
    }

    bool poll() override {
        return WaitForSingleObject(this->event_handle, 0) == WAIT_OBJECT_0;
    }

    [[nodiscard]] int fd() const override { return -1; }

    [[nodiscard]] void *native_handle() const override { return this->event_handle; }
};

std::unique_ptr<Notifier> create() {
    return std::make_unique<WindowsEventNotifier>();
}

}
