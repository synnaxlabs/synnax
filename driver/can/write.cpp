// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <string>
#include <utility>

#include "driver/can/write.h"

namespace driver::can {
Transmitter::Transmitter(const bus::WriteConfig &cfg, Acquire acquire):
    acquirer(std::move(acquire)) {
    for (const auto &m: cfg.messages) {
        const auto &id = std::get<synnax::library::CanIdentifier>(*m.entry.identifier);
        this->frames.push_back({.id = id.id, .extended = id.extended, .fd = id.fd});
    }
}

x::errors::Error Transmitter::acquire() {
    auto [link, err] = this->acquirer();
    this->link = std::move(link);
    return err;
}

void Transmitter::release() {
    this->link.reset();
}

x::errors::Error Transmitter::send(
    const std::size_t message,
    const std::span<const std::uint8_t> payload
) {
    auto frame = this->frames[message];
    if (payload.size() > frame.data.size())
        return {
            FRAME_ERROR,
            std::to_string(payload.size()) + " bytes do not fit in a CAN frame"
        };
    frame.length = static_cast<std::uint8_t>(payload.size());
    std::ranges::copy(payload, frame.data.begin());
    auto [bus, err] = this->link->bus();
    if (err) return err;
    err = bus->send(frame);
    if (err.matches(CRITICAL_HARDWARE_ERROR)) this->link->close(bus);
    return err;
}
}
