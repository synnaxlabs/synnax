// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <atomic>
#include <condition_variable>
#include <deque>
#include <memory>
#include <mutex>
#include <optional>
#include <string>
#include <string_view>
#include <thread>
#include <utility>
#include <vector>

#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"

#include "driver/can/can.h"
#include "driver/can/slcan/protocol.h"
#include "driver/serial/port.h"

namespace driver::can::slcan {
/// @brief the serial baud rate the backend opens adapters at. USB adapters ignore it.
constexpr std::uint32_t BAUD_RATE = 115200;
/// @brief how long the adapter has to answer a command while the bus opens.
const x::telem::TimeSpan REPLY_TIMEOUT = x::telem::SECOND;

/// @brief the most frames send queues before it refuses more.
constexpr std::size_t MAX_QUEUED = 1024;

/// @brief an slcan adapter on a serial port. A thread owns the port once the bus
/// opens, so a sent frame waits up to a few milliseconds before it reaches the port.
/// The adapter reports no receive times, so frames carry the host time at which their
/// bytes arrived.
class Bus final : public can::Bus {
    std::unique_ptr<serial::Port> port;
    std::string name;
    bool fd;
    bool listen_only;
    Decoder decoder;
    std::mutex mu;
    std::condition_variable cv;
    std::deque<Frame> inbox;
    std::vector<std::string> outbox;
    x::errors::Error failure = x::errors::NIL;
    std::atomic<bool> running = false;
    std::thread thread;
    bool closed = false;

    Bus(std::unique_ptr<serial::Port> port, const synnax::can::Properties &props);

    std::pair<std::optional<Reply>, x::errors::Error> read(x::telem::TimeSpan timeout);
    std::pair<Reply, x::errors::Error> command(std::string_view command);
    void run();
    x::errors::Error exchange();

public:
    /// @brief configures the adapter on an open port and puts it on the bus.
    /// @returns CONFIG_ERROR when slcan has no command for a bitrate or the adapter
    /// rejects a command, and TEMPORARY_HARDWARE_ERROR when the device does not answer.
    [[nodiscard]] static std::pair<std::unique_ptr<Bus>, x::errors::Error>
    open(std::unique_ptr<serial::Port> port, const synnax::can::Properties &props);

    ~Bus() override;

    /// @returns the port's error once the port fails.
    [[nodiscard]] std::pair<bool, x::errors::Error>
    receive(Frame &frame, x::telem::TimeSpan timeout) override;

    /// @returns TEMPORARY_HARDWARE_ERROR when MAX_QUEUED frames wait for the port, and
    /// the port's error once the port fails.
    [[nodiscard]] x::errors::Error send(const Frame &frame) override;

    x::errors::Error close() override;
};

/// @brief opens slcan adapters on serial ports. A channel's name is the port's path,
/// such as /dev/ttyACM0 or COM3.
class Backend final : public can::Backend {
public:
    /// @returns no channels. A serial port does not show whether it holds an slcan
    /// adapter, so the user adds each adapter by its port path.
    [[nodiscard]] std::pair<std::vector<Channel>, x::errors::Error> scan() override;

    [[nodiscard]] std::pair<std::unique_ptr<can::Bus>, x::errors::Error>
    open(const synnax::can::Properties &props) override;
};
}
