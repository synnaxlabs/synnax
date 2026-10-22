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

/// @brief the most frames send queues before it refuses more, and the most received
/// frames the bus holds before it drops the oldest.
constexpr std::size_t MAX_QUEUED = 1024;

/// @returns err from a serial port as a CAN error: UNREACHABLE_ERROR for a port that
/// cannot open or that fails, CONFIG_ERROR for port settings that the platform refuses,
/// and CRITICAL_HARDWARE_ERROR for any other error.
[[nodiscard]] x::errors::Error translate(const x::errors::Error &err);

/// @brief an slcan adapter on a serial port. A thread owns the port once the bus opens,
/// so a sent frame waits up to a few milliseconds before it reaches the port. The
/// adapter reports no receive times, so frames carry the host time at which their bytes
/// arrived.
class Bus final : public can::Bus {
    std::unique_ptr<serial::Port> port;
    Firmware firmware = Firmware::CANABLE;
    Decoder decoder;
    std::mutex mu;
    std::condition_variable cv;
    std::deque<Frame> inbox;
    /// @brief true from the first received frame the inbox drops until it empties.
    bool overflowed = false;
    std::vector<std::string> outbox;
    x::errors::Error failure = x::errors::NIL;
    std::atomic<bool> running = false;
    std::thread thread;
    bool closed = false;

    Bus(std::unique_ptr<serial::Port> port, const synnax::can::Properties &props);

    /// @brief reads what the adapter sends within timeout and queues its frames.
    /// @returns the messages that are not frames.
    std::pair<std::vector<Message>, x::errors::Error> read(x::telem::TimeSpan timeout);
    x::errors::Error write(std::string_view line);
    /// @returns the first message that matches within REPLY_TIMEOUT, or nullopt.
    std::pair<std::optional<Message>, x::errors::Error>
    await(bool (*matches)(const Message &));
    /// @returns the adapter's answer to VERSION_COMMAND.
    std::pair<std::string, x::errors::Error> version();
    x::errors::Error configure(std::string_view command);
    void run();
    x::errors::Error exchange();

    /// @returns TEMPORARY_HARDWARE_ERROR when MAX_QUEUED frames wait for the port, and
    /// the port's error once the port fails.
    [[nodiscard]] x::errors::Error transmit(const Frame &frame) override;

public:
    /// @brief identifies the adapter's firmware on an open port, configures the
    /// adapter, and puts it on the bus.
    /// @returns CONFIG_ERROR when the firmware has no command for a setting or the
    /// adapter rejects a command, and UNREACHABLE_ERROR when the device does not answer
    /// or the port fails.
    [[nodiscard]] static std::pair<std::unique_ptr<Bus>, x::errors::Error>
    open(std::unique_ptr<serial::Port> port, const synnax::can::Properties &props);

    ~Bus() override;

    /// @returns the port's error once the port fails.
    [[nodiscard]] std::pair<bool, x::errors::Error>
    receive(Frame &frame, x::telem::TimeSpan timeout) override;

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
