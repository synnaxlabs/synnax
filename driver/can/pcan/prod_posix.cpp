// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <cerrno>
#include <chrono>
#include <climits>
#include <cstring>
#include <string>

#include <poll.h>

#include "driver/can/can.h"
#include "driver/can/pcan/prod.h"

namespace driver::can::pcan {
namespace {
/// @brief a descriptor that polls readable while the receive queue holds frames.
/// PCAN-Basic owns the descriptor and closes it on Uninitialize.
class Descriptor final : public ReceiveEvent {
    int fd;

public:
    explicit Descriptor(const int fd): fd(fd) {}

    x::errors::Error wait(const x::telem::TimeSpan timeout) override {
        const auto ms = std::chrono::ceil<std::chrono::milliseconds>(timeout.chrono());
        pollfd target{.fd = this->fd, .events = POLLIN, .revents = 0};
        const auto n = ::poll(
            &target,
            1,
            static_cast<int>(std::min<std::int64_t>(ms.count(), INT_MAX))
        );
        if (n < 0 && errno != EINTR)
            return {
                CRITICAL_HARDWARE_ERROR,
                std::string("cannot wait on the receive event: ") + std::strerror(errno)
            };
        if ((target.revents & (POLLERR | POLLNVAL)) != 0)
            return {CRITICAL_HARDWARE_ERROR, "the receive event descriptor failed"};
        return x::errors::NIL;
    }
};
}

std::pair<std::unique_ptr<ReceiveEvent>, x::errors::Error>
ProdAPI::OpenReceiveEvent(const TPCANHandle channel) {
    int fd = -1;
    const auto status = this->GetValue(channel, PCAN_RECEIVE_EVENT, &fd, sizeof(fd));
    if (status != PCAN_ERROR_OK)
        return {
            nullptr,
            x::errors::Error(
                CRITICAL_HARDWARE_ERROR,
                "cannot get the receive event: " + describe(*this, status)
            )
        };
    return {std::make_unique<Descriptor>(fd), x::errors::NIL};
}
}
