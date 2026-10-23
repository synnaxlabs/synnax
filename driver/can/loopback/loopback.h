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
#include <cstddef>
#include <cstdint>
#include <deque>
#include <memory>
#include <mutex>
#include <string>
#include <unordered_map>
#include <utility>
#include <vector>

#include "driver/can/can.h"

/// @brief an in-memory CAN network for tests that behaves like a set of adapters. Every
/// bus on a channel receives the frames the other buses on the channel send. A
/// channel's adapter can be unplugged and plugged back, can refuse a second open, and
/// can fail any call on demand.
namespace driver::can::loopback {
/// @brief the backend name that scan reports.
const std::string BACKEND = "loopback";

/// @brief a call that a test can make fail.
enum class Call : std::uint8_t { SCAN, OPEN, RECEIVE, TRANSMIT, CLOSE };

class Bus;

/// @brief the shared medium that the buses of a Backend attach to.
class Network {
    friend class Bus;
    friend class Backend;

    /// @brief the adapter of one channel and the wire behind it.
    struct Channel {
        std::vector<Bus *> buses;
        bool unplugged = false;
        /// @brief the buses open through the backend, not counting nodes.
        std::size_t open = 0;
        /// @brief every attempt to open the channel through the backend.
        std::size_t attempts = 0;
        /// @brief every bus opened through the backend.
        std::size_t opens = 0;
    };

    std::mutex mu;
    std::unordered_map<std::string, Channel> channels;
    std::unordered_map<Call, std::deque<x::errors::Error>> faults;

    void detach(Bus *bus);
    void deliver(const Bus *from, const Frame &frame);
    /// @returns the next error a call fails with, or NIL.
    x::errors::Error fault(Call call);
};

/// @brief a bus on a Network.
class Bus final : public can::Bus {
    friend class Network;
    friend class Backend;
    std::shared_ptr<Network> network;
    /// @brief true for another node on the wire, which unplug and faults never touch.
    const bool node;
    bool closed = false;
    std::mutex mu;
    std::condition_variable cv;
    std::deque<Frame> inbox;
    /// @brief true once the bus's adapter is unplugged. It stays true after a replug.
    bool unplugged = false;

    [[nodiscard]] x::errors::Error transmit(const Frame &frame) override;

public:
    Bus(std::shared_ptr<Network> network,
        const synnax::can::Properties &props,
        bool node);
    ~Bus() override;

    [[nodiscard]] std::pair<bool, x::errors::Error>
    receive(Frame &frame, x::telem::TimeSpan timeout) override;

    x::errors::Error close() override;
};

/// @brief a backend whose buses share one Network.
class Backend final : public can::Backend {
    std::shared_ptr<Network> network = std::make_shared<Network>();
    std::vector<std::string> names;
    bool exclusive;

public:
    /// @param channels the channel names scan returns. open accepts any name.
    /// @param exclusive true when a channel refuses a second open while a bus on it is
    /// open, as gs_usb, PCAN, and CANlib adapters do.
    explicit Backend(std::vector<std::string> channels = {}, bool exclusive = false):
        names(std::move(channels)), exclusive(exclusive) {}

    [[nodiscard]] std::pair<std::vector<can::Channel>, x::errors::Error>
    scan() override;

    /// @returns UNREACHABLE_ERROR when the channel is unplugged, or when the backend is
    /// exclusive and a bus on the channel is open.
    [[nodiscard]] std::pair<std::unique_ptr<can::Bus>, x::errors::Error>
    open(const synnax::can::Properties &props) override;

    /// @returns a bus for another node on the wire of a channel. The node never counts
    /// as an open of the channel, and unplug and faults leave it alone.
    [[nodiscard]] std::unique_ptr<can::Bus> node(const synnax::can::Properties &props);

    /// @brief unplugs the adapter of a channel. open fails, and every receive and send
    /// on a bus open on the channel fails with UNREACHABLE_ERROR until it closes.
    void unplug(const std::string &channel);

    /// @brief plugs the adapter of a channel back in. Buses open during the unplug stay
    /// failed, as on a real adapter.
    void plug(const std::string &channel);

    /// @brief delivers a bus error frame to every bus opened on the channel, as an
    /// adapter reports a fault on its bus.
    void raise(const std::string &channel, const Frame &error);

    /// @brief makes the next times calls of a kind fail with err. Faults queue in order
    /// and apply to buses opened through the backend.
    void fail(Call call, const x::errors::Error &err, std::size_t times = 1);

    /// @returns the number of attempts to open the channel.
    [[nodiscard]] std::size_t attempts(const std::string &channel);

    /// @returns the number of buses opened on the channel.
    [[nodiscard]] std::size_t opens(const std::string &channel);

    /// @returns the number of buses open on the channel now.
    [[nodiscard]] std::size_t in_use(const std::string &channel);
};
}
