// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstdint>
#include <memory>
#include <optional>
#include <span>
#include <string>
#include <vector>

#include "x/cpp/breaker/breaker.h"
#include "x/cpp/telem/frame.h"
#include "x/cpp/telem/telem.h"

#include "driver/bus/config.h"
#include "driver/bus/connection.h"
#include "driver/common/read_task.h"

namespace driver::bus {
/// @brief how long a warning stays on the task status after its cause stops.
const auto WARNING_HOLD = 1 * x::telem::SECOND;

/// @brief reads a byte stream or datagrams from a device's connection, splits it into
/// frames, decodes each frame whose message the task reads, and writes the values to
/// the messages' channels. It polls messages that have a query at the configured
/// rate. Each poll holds the connection from its query until its reply or timeout, so
/// no other task's write lands between them.
class Source final : public common::Source {
public:
    /// @param cfg the resolved read config.
    /// @param acquire acquires the device's connection on start.
    Source(ReadConfig cfg, Acquire acquire);

    [[nodiscard]] synnax::framer::WriterConfig writer_config() const override;

    [[nodiscard]] std::vector<synnax::channel::Channel> channels() const override;

    /// @brief acquires and opens the connection.
    /// @returns transport::CONFIG_ERROR when the device properties are invalid or
    /// another task has the device open with other properties. An unreachable device
    /// is not an error here: read retries it.
    x::errors::Error start() override;

    /// @brief releases the connection, which closes when no other task uses it.
    x::errors::Error stop() override;

    /// @brief sends one query and reads until its reply, its timeout, or the stop of
    /// breaker, or reads once for at most READ_TIMEOUT, and decodes what arrived into
    /// fr.
    /// @returns transport::UNREACHABLE_ERROR when the connection fails, after which
    /// the next read reconnects. Framing errors, short payloads, unreadable text
    /// fields, and missed replies come back as warnings.
    common::ReadResult read(x::breaker::Breaker &breaker, x::telem::Frame &fr) override;

private:
    /// @brief decoded samples of one message waiting to go into a frame.
    struct Pending {
        /// @brief one decoded sample per arrival, reused across reads.
        std::vector<codec::Values> values;
        /// @brief the arrival time of each sample.
        std::vector<x::telem::TimeStamp> times;
        /// @brief the last time stamped on the message's index.
        x::telem::TimeStamp last;
    };

    ReadConfig cfg;
    Acquire acquire;
    std::shared_ptr<Connection> conn;
    /// @brief the open count of the connection when the framer last read from it.
    std::uint64_t opens = 0;
    std::vector<Pending> pending;
    std::vector<std::string> raw;
    /// @brief the index into cfg.polled of the next query to send in this cycle.
    std::size_t next_query;
    /// @brief the message whose query waits for its reply.
    std::optional<std::size_t> awaiting;
    x::telem::TimeStamp next_poll;
    std::vector<std::string> warnings;
    std::string held_warning;
    x::telem::TimeStamp held_at;

    x::errors::Error exchange(Transport &t, const x::breaker::Breaker &breaker);
    x::errors::Error
    query(Transport &t, const x::breaker::Breaker &breaker, x::telem::TimeStamp now);
    x::errors::Error receive(Transport &t, x::telem::TimeSpan timeout);
    void handle(std::span<const std::uint8_t> frame, x::telem::TimeStamp time);
    void decode(
        std::size_t message,
        std::span<const std::uint8_t> frame,
        x::telem::TimeStamp time
    );
    void flush(x::telem::Frame &fr);
    std::string warning(x::telem::TimeStamp now);
};
}
