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
#include <string>
#include <utility>

#include "driver/errors/errors.h"
#include "driver/mil1553/write.h"

namespace driver::mil1553 {
Output::Output(
    const bus::WriteConfig &cfg,
    synnax::mil1553::Properties props,
    Acquire acquire
):
    props(std::move(props)), acquire(std::move(acquire)) {
    for (const auto &m: cfg.messages)
        this->commands.push_back(
            codec::mil1553::Command::from(
                std::get<synnax::library::Mil1553Identifier>(*m.entry.identifier)
            )
        );
}

x::errors::Error Output::start() {
    auto [conn, err] = this->acquire();
    if (err) return err;
    this->conn = std::move(conn);
    auto guard = this->conn->lock();
    return guard.transport().second;
}

void Output::stop() {
    this->conn.reset();
}

x::errors::Error
Output::send(const std::size_t message, const std::span<const std::uint8_t> payload) {
    const auto &cmd = this->commands[message];
    // The payload holds only the bytes the fields reach, so pad it to the full count.
    std::array<std::uint8_t, 2 * codec::mil1553::MAX_WORDS> bytes{};
    const std::size_t size = 2 * cmd.count;
    std::copy_n(payload.begin(), std::min(payload.size(), size), bytes.begin());
    std::array<std::uint16_t, codec::mil1553::MAX_WORDS> words{};
    codec::mil1553::from_payload(std::span(bytes.data(), size), words);
    const std::span data(words.data(), cmd.count);
    auto guard = this->conn->lock();
    auto [ch, err] = guard.transport();
    if (err) return err;
    if (this->props.role == synnax::mil1553::ROLE_REMOTE_TERMINAL) {
        err = ch->respond(cmd.rt, cmd.subaddress, data);
        if (err) guard.close();
        return err;
    }
    auto [t, t_err] = ch->transact(cmd, data);
    if (t_err) {
        guard.close();
        return t_err;
    }
    const auto name = address(cmd);
    if (!t.answered)
        return x::errors::Error(
            errors::TEMPORARY_HARDWARE_ERROR,
            name + " did not answer"
        );
    if (t.status.message_error)
        return x::errors::Error(
            errors::TEMPORARY_HARDWARE_ERROR,
            name + " reported a message error"
        );
    if (t.status.busy)
        return x::errors::Error(errors::TEMPORARY_HARDWARE_ERROR, name + " was busy");
    return x::errors::NIL;
}
}
