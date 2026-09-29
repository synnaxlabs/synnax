// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <memory>
#include <utility>

#include "x/cpp/notify/notify.h"

#include "driver/can/can.h"
#include "driver/can/pcan/prod.h"

namespace driver::can::pcan {
namespace {
/// @brief a Windows auto-reset event that PCAN-Basic signals.
class Event final : public ReceiveEvent {
    std::unique_ptr<x::notify::Notifier> notifier;

public:
    explicit Event(std::unique_ptr<x::notify::Notifier> notifier):
        notifier(std::move(notifier)) {}

    x::errors::Error wait(const x::telem::TimeSpan timeout) override {
        this->notifier->wait(timeout);
        return x::errors::NIL;
    }
};
}

std::pair<std::unique_ptr<ReceiveEvent>, x::errors::Error>
ProdAPI::OpenReceiveEvent(const TPCANHandle channel) {
    auto notifier = x::notify::create();
    void *handle = notifier->native_handle();
    const auto
        status = this->SetValue(channel, PCAN_RECEIVE_EVENT, &handle, sizeof(handle));
    if (status != PCAN_ERROR_OK)
        return {
            nullptr,
            x::errors::Error(
                CRITICAL_HARDWARE_ERROR,
                "cannot register the receive event: " + describe(*this, status)
            )
        };
    return {std::make_unique<Event>(std::move(notifier)), x::errors::NIL};
}
}
