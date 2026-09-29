// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <condition_variable>
#include <cstdint>
#include <deque>
#include <memory>
#include <mutex>
#include <span>
#include <thread>
#include <unordered_map>
#include <utility>
#include <vector>

#include "x/cpp/telem/frame.h"
#include "x/cpp/telem/telem.h"

#include "driver/bus/config.h"
#include "driver/bus/connection.h"
#include "driver/common/status.h"
#include "driver/common/write_task.h"
#include "driver/task/task.h"

namespace driver::bus {
/// @brief where a Sink sends the messages it encodes. The Sink calls start, send, and
/// stop from one thread at a time.
class Output {
public:
    virtual ~Output() = default;

    /// @brief prepares to send, when the task starts.
    /// @returns an error that fails the start.
    virtual x::errors::Error start() = 0;

    /// @brief releases what start acquired, when the task stops.
    virtual void stop() = 0;

    /// @brief sends the encoded payload of the message at index message of the write
    /// config.
    /// @returns an error the task reports as a warning.
    virtual x::errors::Error
    send(std::size_t message, std::span<const std::uint8_t> payload) = 0;
};

/// @brief sends each framed payload on a device's shared transport. It reopens the
/// transport after a failed send.
class TransportOutput final : public Output {
public:
    /// @param acquire acquires the device's connection on start.
    explicit TransportOutput(Acquire acquire): acquire(std::move(acquire)) {}

    /// @returns transport::CONFIG_ERROR when another task has the device open with
    /// other properties.
    x::errors::Error start() override;

    /// @brief releases the connection, which closes when no other task uses it.
    void stop() override;

    x::errors::Error
    send(std::size_t message, std::span<const std::uint8_t> payload) override;

private:
    Acquire acquire;
    std::shared_ptr<Connection> conn;
};

/// @brief encodes command channel values into messages and sends them on one I/O
/// thread. A message with a period is sent on that period with the latest values, and
/// one without is sent once per command frame that changes it. A message is sent only
/// after each of its command channels has a value; unmapped fields are zero. Stopping
/// discards every value, so a restarted task never replays a stale one.
class Sink final : public common::Sink {
public:
    /// @param cfg the resolved write config.
    /// @param acquire acquires the device's connection on start. The sink reopens the
    /// connection after a failed send.
    /// @param ctx reports send failures as task status warnings.
    /// @param task the task the sink belongs to.
    Sink(
        WriteConfig cfg,
        Acquire acquire,
        const std::shared_ptr<task::Context> &ctx,
        const synnax::task::Task &task
    );

    /// @param cfg the resolved write config.
    /// @param output where the sink sends each encoded, framed message.
    /// @param ctx reports send failures as task status warnings.
    /// @param task the task the sink belongs to.
    Sink(
        WriteConfig cfg,
        std::unique_ptr<Output> output,
        const std::shared_ptr<task::Context> &ctx,
        const synnax::task::Task &task
    );

    ~Sink() override;

    Sink(const Sink &) = delete;
    Sink &operator=(const Sink &) = delete;

    /// @brief starts the output and the I/O thread.
    /// @returns the error of the output's start, such as transport::CONFIG_ERROR when
    /// another task has the device open with other properties.
    x::errors::Error start() override;

    /// @brief stops the I/O thread, drops unsent messages, and stops the output. The
    /// transport output releases the
    /// connection, which closes when no other task uses it.
    x::errors::Error stop() override;

    /// @brief applies the last sample of each command channel in frame and queues the
    /// messages without a period that it changed.
    x::errors::Error write(x::telem::Frame &frame) override;

private:
    /// @brief the send state of one message.
    struct State {
        /// @brief the latest value of each field.
        codec::Values values;
        /// @brief true for each slot a command has set.
        std::vector<bool> seen;
        /// @brief the number of bound slots no command has set.
        std::size_t unseen = 0;
        /// @brief when the message is next sent, for messages with a period.
        x::telem::TimeStamp due;
    };

    WriteConfig cfg;
    std::unique_ptr<Output> output;
    /// @brief the message slots each command channel drives.
    std::unordered_map<
        synnax::channel::Key,
        std::vector<std::pair<std::size_t, std::size_t>>>
        targets;

    std::mutex mu;
    std::condition_variable cv;
    bool running = false;
    std::vector<State> states;
    /// @brief each unsent message and its framed payload.
    std::deque<std::pair<std::size_t, std::vector<std::uint8_t>>> queue;
    std::thread thread;

    /// @brief guards status, which the control and I/O threads both report through.
    std::mutex status_mu;
    common::StatusHandler status;

    void reset();
    void run();
    void enqueue(std::size_t message);
    void send(std::size_t message, const std::vector<std::uint8_t> &payload);
    void warn(const std::string &message);
};
}
