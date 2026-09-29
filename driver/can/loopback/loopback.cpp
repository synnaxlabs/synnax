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
void Network::attach(Bus *bus) {
    std::lock_guard lock(this->mu);
    this->channels[bus->channel].push_back(bus);
}

void Network::detach(Bus *bus) {
    std::lock_guard lock(this->mu);
    auto &buses = this->channels[bus->channel];
    std::erase(buses, bus);
}

void Network::deliver(const Bus *from, const Frame &frame) {
    std::lock_guard lock(this->mu);
    for (auto *bus: this->channels[from->channel]) {
        if (bus == from) continue;
        {
            std::lock_guard bus_lock(bus->mu);
            bus->inbox.push_back(frame);
            bus->inbox.back().time = x::telem::TimeStamp::now();
            bus->inbox.back().clock = Clock::HOST;
        }
        bus->cv.notify_one();
    }
}

Bus::Bus(std::shared_ptr<Network> network, const synnax::can::Properties &props):
    network(std::move(network)),
    channel(props.channel),
    fd(props.fd),
    listen_only(props.listen_only) {
    this->network->attach(this);
}

Bus::~Bus() {
    this->close();
}

std::pair<bool, x::errors::Error>
Bus::receive(Frame &frame, const x::telem::TimeSpan timeout) {
    std::unique_lock lock(this->mu);
    if (!this->cv.wait_for(lock, timeout.chrono(), [this] {
            return !this->inbox.empty();
        }))
        return {false, x::errors::NIL};
    frame = this->inbox.front();
    this->inbox.pop_front();
    return {true, x::errors::NIL};
}

x::errors::Error Bus::send(const Frame &frame) {
    if (this->listen_only)
        return {LISTEN_ONLY_ERROR, "channel " + this->channel + " is listen only"};
    if (auto err = validate(frame, this->fd)) return err;
    this->network->deliver(this, frame);
    return x::errors::NIL;
}

x::errors::Error Bus::close() {
    if (this->closed) return x::errors::NIL;
    this->closed = true;
    this->network->detach(this);
    return x::errors::NIL;
}

std::pair<std::vector<Channel>, x::errors::Error> Backend::scan() {
    std::vector<Channel> found;
    found.reserve(this->channels.size());
    for (const auto &name: this->channels)
        found.push_back({.backend = BACKEND, .name = name, .description = name});
    return {found, x::errors::NIL};
}

std::pair<std::unique_ptr<can::Bus>, x::errors::Error>
Backend::open(const synnax::can::Properties &props) {
    return {std::make_unique<Bus>(this->network, props), x::errors::NIL};
}
}
