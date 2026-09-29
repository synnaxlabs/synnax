// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <array>
#include <utility>

#include "driver/arinc429/write.h"

namespace driver::arinc429 {
Transmitter::Transmitter(
    const bus::WriteConfig &cfg,
    std::shared_ptr<Backend> backend,
    synnax::arinc429::Properties props
):
    backend(std::move(backend)), props(std::move(props)) {
    for (const auto &m: cfg.messages)
        this->labels.push_back(
            std::get<synnax::library::Arinc429Identifier>(*m.entry.identifier)
        );
}

x::errors::Error Transmitter::acquire() {
    auto [ch, err] = this->backend->open(this->props, Direction::TRANSMIT);
    if (err) return err;
    this->channel = std::move(ch);
    return x::errors::NIL;
}

void Transmitter::release() {
    this->channel.reset();
}

x::errors::Error Transmitter::send(
    const std::size_t message,
    const std::span<const std::uint8_t> payload
) {
    if (this->channel == nullptr) {
        auto [ch, err] = this->backend->open(this->props, Direction::TRANSMIT);
        if (err) return err;
        this->channel = std::move(ch);
    }
    std::array<std::uint8_t, codec::arinc429::PAYLOAD_SIZE> bytes{};
    std::copy_n(payload.begin(), std::min(payload.size(), bytes.size()), bytes.begin());
    const auto &id = this->labels[message];
    const std::array words{
        codec::arinc429::Word::pack(id.label, id.sdi, id.sdi_matched, bytes)
    };
    const auto err = this->channel->write(words);
    if (err) this->channel.reset();
    return err;
}
}
