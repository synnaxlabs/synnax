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
/// @brief sends a write task's encoded messages to its device.
class Transmitter {
public:
    virtual ~Transmitter() = default;

    /// @brief acquires the device's connection. Called on start.
    /// @returns transport::CONFIG_ERROR when another task has the device open with
    /// other properties.
    virtual x::errors::Error acquire() = 0;

    /// @brief releases the connection. Called on stop, after the last send.
    virtual void release() = 0;

    /// @brief sends one encoded message.
    /// @param message the index of the message in the write config.
    /// @param payload the encoded, framed message.
    virtual x::errors::Error
    send(std::size_t message, std::span<const std::uint8_t> payload) = 0;
};

/// @brief sends each message as bytes on a shared byte-stream or datagram
/// connection, and reopens the connection after a failed send.
class ConnectionTransmitter final : public Transmitter {
    Acquire acquirer;
    std::shared_ptr<Connection> conn;

public:
    explicit ConnectionTransmitter(Acquire acquire): acquirer(std::move(acquire)) {}

    x::errors::Error acquire() override;
    void release() override;
    x::errors::Error
    send(std::size_t message, std::span<const std::uint8_t> payload) override;
};

/// @brief encodes command channel values into messages and sends them on one I/O
/// thread. A message with a period is sent on that period with the latest values, and
/// one without is sent once per command frame that changes it. A message is sent only
/// after each of its command channels has a value; unmapped fields are zero. Stopping
/// discards every value, so a restarted task never replays a stale one.
class Sink final : public common::Sink {
public:
    /// @param cfg the resolved write config.
    /// @param transmitter sends the encoded messages to the device.
    /// @param ctx reports send failures as task status warnings.
    /// @param task the task the sink belongs to.
    Sink(
        WriteConfig cfg,
        std::unique_ptr<Transmitter> transmitter,
        const std::shared_ptr<task::Context> &ctx,
        const synnax::task::Task &task
    );

    ~Sink() override;

    Sink(const Sink &) = delete;
    Sink &operator=(const Sink &) = delete;

    /// @brief acquires the connection and starts the I/O thread.
    /// @returns transport::CONFIG_ERROR when another task has the device open with
    /// other properties.
    x::errors::Error start() override;

    /// @brief stops the I/O thread, drops unsent messages, and releases the
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
    std::unique_ptr<Transmitter> transmitter;
    /// @brief the message slots each command channel drives.
    std::unordered_map<
        synnax::channel::Key,
        std::vector<std::pair<std::size_t, std::size_t>>>
        targets;

    std::mutex mu;
    std::condition_variable cv;
    bool running = false;
    std::vector<State> states;
    /// @brief each message waiting to send, by index, with its payload.
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
