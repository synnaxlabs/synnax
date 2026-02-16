// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <memory>

#include "x/cpp/telem/telem.h"

namespace x::notify {

struct Notifier {
    virtual ~Notifier() = default;

    virtual void signal() = 0;

    virtual bool wait(telem::TimeSpan timeout = telem::TimeSpan::max()) = 0;

    virtual bool poll() = 0;

    /// @returns the file descriptor for use with epoll/kqueue, or -1 if not supported.
    [[nodiscard]] virtual int fd() const = 0;

    /// @returns the native handle for platform-specific use. On Windows, this returns
    /// the HANDLE for use with WaitForMultipleObjects. On POSIX systems, returns
    /// nullptr.
    [[nodiscard]] virtual void *native_handle() const = 0;
};

std::unique_ptr<Notifier> create();

}
