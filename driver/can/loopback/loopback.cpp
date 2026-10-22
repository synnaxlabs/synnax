// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>

#include "driver/can/loopback/loopback.h"

namespace driver::can::loopback {
void Network::detach(Bus *bus) {
    std::lock_guard lock(this->mu);
    auto &channel = this->channels[bus->name];
    std::erase(channel.buses, bus);
    if (!bus->node) channel.open--;
}

void Network::deliver(const Bus *from, const Frame &frame) {
    std::lock_guard lock(this->mu);
    for (auto *bus: this->channels[from->name].buses) {
        if (bus == from) continue;
        {
            std::lock_guard bus_lock(bus->mu);
            if (bus->unplugged) continue;
            bus->inbox.push_back(frame);
            bus->inbox.back().time = x::telem::TimeStamp::now();
            bus->inbox.back().clock = Clock::HOST;
        }
        bus->cv.notify_one();
    }
}

x::errors::Error Network::fault(const Call call) {
    std::lock_guard lock(this->mu);
    auto &queue = this->faults[call];
    if (queue.empty()) return x::errors::NIL;
    auto err = queue.front();
    queue.pop_front();
    return err;
}

Bus::Bus(
    std::shared_ptr<Network> network,
    const synnax::can::Properties &props,
    const bool node
):
    can::Bus(props.channel, props.fd, props.listen_only),
    network(std::move(network)),
    node(node) {}

Bus::~Bus() {
    this->close();
}

std::pair<bool, x::errors::Error>
Bus::receive(Frame &frame, const x::telem::TimeSpan timeout) {
    if (!this->node)
        if (auto err = this->network->fault(Call::RECEIVE)) return {false, err};
    std::unique_lock lock(this->mu);
    this->cv.wait_for(lock, timeout.chrono(), [this] {
        return this->unplugged || !this->inbox.empty();
    });
    if (this->unplugged)
        return {false, {UNREACHABLE_ERROR, "channel " + this->name + " was unplugged"}};
    if (this->inbox.empty()) return {false, x::errors::NIL};
    frame = this->inbox.front();
    this->inbox.pop_front();
    return {true, x::errors::NIL};
}

x::errors::Error Bus::transmit(const Frame &frame) {
    if (!this->node)
        if (auto err = this->network->fault(Call::TRANSMIT)) return err;
    {
        std::lock_guard lock(this->mu);
        if (this->unplugged)
            return {UNREACHABLE_ERROR, "channel " + this->name + " was unplugged"};
    }
    this->network->deliver(this, frame);
    return x::errors::NIL;
}

x::errors::Error Bus::close() {
    if (this->closed) return x::errors::NIL;
    this->closed = true;
    this->network->detach(this);
    if (this->node) return x::errors::NIL;
    return this->network->fault(Call::CLOSE);
}

std::pair<std::vector<can::Channel>, x::errors::Error> Backend::scan() {
    if (auto err = this->network->fault(Call::SCAN)) return {{}, err};
    std::vector<can::Channel> found;
    std::lock_guard lock(this->network->mu);
    for (const auto &name: this->names)
        if (!this->network->channels[name].unplugged)
            found.push_back({.backend = BACKEND, .name = name, .description = name});
    return {found, x::errors::NIL};
}

std::pair<std::unique_ptr<can::Bus>, x::errors::Error>
Backend::open(const synnax::can::Properties &props) {
    {
        std::lock_guard lock(this->network->mu);
        auto &channel = this->network->channels[props.channel];
        channel.attempts++;
    }
    if (auto err = this->network->fault(Call::OPEN)) return {nullptr, err};
    std::lock_guard lock(this->network->mu);
    auto &channel = this->network->channels[props.channel];
    if (channel.unplugged)
        return {
            nullptr,
            {UNREACHABLE_ERROR, "channel " + props.channel + " is not attached"}
        };
    if (this->exclusive && channel.open > 0)
        return {
            nullptr,
            {UNREACHABLE_ERROR, "channel " + props.channel + " is in use"}
        };
    channel.open++;
    channel.opens++;
    auto bus = std::make_unique<Bus>(this->network, props, false);
    channel.buses.push_back(bus.get());
    return {std::move(bus), x::errors::NIL};
}

std::unique_ptr<can::Bus> Backend::node(const synnax::can::Properties &props) {
    std::lock_guard lock(this->network->mu);
    auto bus = std::make_unique<Bus>(this->network, props, true);
    this->network->channels[props.channel].buses.push_back(bus.get());
    return bus;
}

void Backend::unplug(const std::string &channel) {
    std::lock_guard lock(this->network->mu);
    auto &ch = this->network->channels[channel];
    ch.unplugged = true;
    for (auto *bus: ch.buses) {
        if (bus->node) continue;
        {
            std::lock_guard bus_lock(bus->mu);
            bus->unplugged = true;
            bus->inbox.clear();
        }
        bus->cv.notify_all();
    }
}

void Backend::plug(const std::string &channel) {
    std::lock_guard lock(this->network->mu);
    this->network->channels[channel].unplugged = false;
}

void Backend::raise(const std::string &channel, const Frame &error) {
    std::lock_guard lock(this->network->mu);
    for (auto *bus: this->network->channels[channel].buses) {
        if (bus->node) continue;
        {
            std::lock_guard bus_lock(bus->mu);
            bus->inbox.push_back(error);
        }
        bus->cv.notify_one();
    }
}

void Backend::fail(const Call call, const x::errors::Error &err, std::size_t times) {
    std::lock_guard lock(this->network->mu);
    auto &queue = this->network->faults[call];
    for (; times > 0; times--)
        queue.push_back(err);
}

std::size_t Backend::attempts(const std::string &channel) {
    std::lock_guard lock(this->network->mu);
    return this->network->channels[channel].attempts;
}

std::size_t Backend::opens(const std::string &channel) {
    std::lock_guard lock(this->network->mu);
    return this->network->channels[channel].opens;
}

std::size_t Backend::in_use(const std::string &channel) {
    std::lock_guard lock(this->network->mu);
    return this->network->channels[channel].open;
}
}
