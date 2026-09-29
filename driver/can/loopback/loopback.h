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
#include <deque>
#include <memory>
#include <mutex>
#include <string>
#include <unordered_map>
#include <utility>
#include <vector>

#include "driver/can/can.h"

/// @brief an in-memory CAN network for tests. Every bus opened on a channel receives
/// the frames the other buses on that channel send.
namespace driver::can::loopback {
/// @brief the backend name that scan reports.
const std::string BACKEND = "loopback";

class Bus;

/// @brief the shared medium that the buses of a Backend attach to.
class Network {
    friend class Bus;
    std::mutex mu;
    std::unordered_map<std::string, std::vector<Bus *>> channels;

    void attach(Bus *bus);
    void detach(Bus *bus);
    void deliver(const Bus *from, const Frame &frame);
};

/// @brief a bus on a Network.
class Bus final : public can::Bus {
    friend class Network;
    std::shared_ptr<Network> network;
    std::string channel;
    bool fd;
    bool listen_only;
    bool closed = false;
    std::mutex mu;
    std::condition_variable cv;
    std::deque<Frame> inbox;

public:
    Bus(std::shared_ptr<Network> network, const synnax::can::Properties &props);
    ~Bus() override;

    [[nodiscard]] std::pair<bool, x::errors::Error>
    receive(Frame &frame, x::telem::TimeSpan timeout) override;

    [[nodiscard]] x::errors::Error send(const Frame &frame) override;

    x::errors::Error close() override;
};

/// @brief a backend whose buses share one Network.
class Backend final : public can::Backend {
    std::shared_ptr<Network> network = std::make_shared<Network>();
    std::vector<std::string> channels;

public:
    /// @param channels the channel names scan returns. open accepts any name.
    explicit Backend(std::vector<std::string> channels = {}):
        channels(std::move(channels)) {}

    [[nodiscard]] std::pair<std::vector<Channel>, x::errors::Error> scan() override;

    [[nodiscard]] std::pair<std::unique_ptr<can::Bus>, x::errors::Error>
    open(const synnax::can::Properties &props) override;
};
}
