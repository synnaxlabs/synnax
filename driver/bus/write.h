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
#include <optional>
#include <span>
#include <string>
#include <thread>
#include <unordered_map>
#include <utility>
#include <vector>

#include "x/cpp/telem/frame.h"
#include "x/cpp/telem/mono_clock.h"
#include "x/cpp/telem/telem.h"

#include "driver/bus/clock.h"
#include "driver/bus/config.h"
#include "driver/bus/connection.h"
#include "driver/bus/warnings.h"
#include "driver/common/write_task.h"

namespace driver::bus {
/// @brief the wait before a sink sends again after a send failed with a temporary
/// error.
const auto RESEND_INTERVAL = 100 * x::telem::MILLISECOND;

/// @brief sends a write task's encoded messages to its device.
class Transmitter {
public:
    virtual ~Transmitter() = default;

    /// @brief acquires the device's connection and opens it. Called on start.
    /// @returns transport::CONFIG_ERROR when the device properties are invalid or
    /// another task has the device open with other properties, and any error that is
    /// not a driver::errors::TEMPORARY_HARDWARE_ERROR when opening fails in a way that
    /// reopening cannot fix. An unreachable device is not an error here.
    virtual x::errors::Error acquire() = 0;

    /// @brief releases the connection. Called on stop, after the last send.
    virtual void release() = 0;

    /// @brief sends one encoded message.
    /// @param message the index of the message in the write config.
    /// @param payload the encoded, framed message.
    /// @returns a driver::errors::TEMPORARY_HARDWARE_ERROR when a later send may
    /// succeed, and any other error when no later send can.
    virtual x::errors::Error
    send(std::size_t message, std::span<const std::uint8_t> payload) = 0;
};

/// @brief sends each message as bytes on a shared byte-stream or datagram connection. A
/// failed write closes the connection, which reopens after its backoff.
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
/// after each of its command channels has a value; unmapped fields are zero. Each
/// message has at most one payload waiting to send: a newer one replaces it. After a
/// temporary send failure, the sink keeps the newest payload of each message and sends
/// again every RESEND_INTERVAL. Stopping discards every value, so a restarted task
/// never replays a stale one. Warnings go to set_warning.
class Sink final : public common::Sink {
public:
    /// @param cfg the resolved write config.
    /// @param transmitter sends the encoded messages to the device.
    /// @param now the steady clock that schedules periodic sends and times warnings.
    Sink(
        WriteConfig cfg,
        std::unique_ptr<Transmitter> transmitter,
        x::telem::NowFunc now = steady_now
    );

    ~Sink() override;

    Sink(const Sink &) = delete;
    Sink &operator=(const Sink &) = delete;

    /// @brief acquires and opens the connection, and starts the I/O thread.
    /// @returns the error of Transmitter::acquire.
    x::errors::Error start() override;

    /// @brief stops the I/O thread, drops unsent messages, and releases the connection,
    /// which closes when no other task uses it.
    x::errors::Error stop() override;

    /// @brief applies the last sample of each command channel in frame and queues the
    /// messages without a period that it changed. A value outside the range of its
    /// field is refused with a warning, and the field keeps its last value.
    /// @returns the error of a failed send that no later send can fix.
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
        /// @brief the steady time the message is next sent, for messages with a period.
        x::telem::TimeStamp due = x::telem::TimeStamp(0);
        /// @brief the payload waiting to send, when the message is in queue.
        std::optional<std::vector<std::uint8_t>> pending;
    };

    WriteConfig cfg;
    std::unique_ptr<Transmitter> transmitter;
    x::telem::NowFunc now;
    /// @brief the message and binding index of each binding of a command channel.
    std::unordered_map<
        synnax::channel::Key,
        std::vector<std::pair<std::size_t, std::size_t>>>
        targets;

    std::mutex mu;
    std::condition_variable cv;
    bool running = false;
    std::vector<State> states;
    /// @brief the index of each message with a pending payload, in the order each
    /// became pending.
    std::deque<std::size_t> queue;
    /// @brief refused values and messages that failed to encode.
    Warnings warnings;
    /// @brief the error of the last send while sends fail with a temporary error.
    x::errors::Error failure;
    /// @brief the error of a failed send that no later send can fix.
    x::errors::Error fatal;
    /// @brief the steady time before which no send runs.
    x::telem::TimeStamp resend_at = x::telem::TimeStamp(0);
    /// @brief the warning last passed to set_warning.
    std::string shown;
    std::thread thread;

    void reset();
    /// @brief queues each periodic message that is due at now.
    /// @returns the steady time the next periodic message is due.
    x::telem::TimeStamp schedule(x::telem::TimeStamp now);
    void run();
    void enqueue(std::size_t message);
    void push(std::size_t message, std::vector<std::uint8_t> payload);
    /// @brief passes the warnings that hold now to set_warning when they changed.
    void report();
};
}
