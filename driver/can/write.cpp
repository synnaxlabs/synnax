// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <limits>
#include <string>
#include <utility>

#include "driver/can/write.h"

namespace driver::can {
std::vector<Frame>
frames(const x::json::Parser &parser, const bus::WriteConfig &cfg, const bool fd) {
    std::vector<Frame> out;
    for (const auto &m: cfg.messages) {
        const auto &payload = std::get<synnax::library::BinaryPayload>(m.entry.payload);
        const auto &id = std::get<synnax::library::CanIdentifier>(*payload.identifier);
        const auto length = m.plan.length();
        if (length > std::numeric_limits<std::uint8_t>::max()) {
            parser.field_err(
                "messages",
                "message " + m.entry.name + " is " + std::to_string(length) +
                    " bytes, longer than any CAN frame"
            );
            continue;
        }
        const Frame frame{
            .id = id.id,
            .extended = id.extended,
            .fd = id.fd,
            .length = static_cast<std::uint8_t>(length),
        };
        if (const auto err = validate(frame, fd))
            parser.field_err("messages", "message " + m.entry.name + ": " + err.data);
        out.push_back(frame);
    }
    return out;
}

Transmitter::Transmitter(std::vector<Frame> frames, Acquire acquire):
    frames(std::move(frames)), acquirer(std::move(acquire)) {}

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
