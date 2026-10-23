// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cstdlib>
#include <memory>
#include <optional>
#include <string>
#include <utility>

#include "driver/can/backends/backends.h"
#include "driver/can/contract.h"
#include "driver/can/loopback/loopback.h"

namespace driver::can::contract {
namespace {
synnax::can::Properties
properties(const std::string &backend, const std::string &channel) {
    synnax::can::Properties p;
    p.backend = backend;
    p.channel = channel;
    p.bitrate = 500000;
    p.data_bitrate = 2000000;
    return p;
}

/// @brief channel can0 on an exclusive loopback network, with a node on its wire.
class Loopback final : public Rig {
    loopback::Backend backend{{}, true};

public:
    [[nodiscard]] synnax::can::Properties props() const override {
        return properties(loopback::BACKEND, "can0");
    }

    std::pair<std::unique_ptr<Bus>, x::errors::Error>
    open(const synnax::can::Properties &props) override {
        return this->backend.open(props);
    }

    std::unique_ptr<Bus> peer(const bool fd) override {
        auto p = this->props();
        p.fd = fd;
        return this->backend.node(p);
    }

    bool fd() override { return true; }

    bool unplug() override {
        this->backend.unplug("can0");
        return true;
    }

    void plug() override { this->backend.plug("can0"); }
};

/// @returns the properties of the adapter that the environment variable names as
/// <backend>:<channel>, or nullopt when it is unset.
std::optional<synnax::can::Properties> named(const char *variable) {
    const char *value = std::getenv(variable);
    if (value == nullptr) return std::nullopt;
    const std::string address = value;
    const auto colon = address.find(':');
    if (colon == std::string::npos) return properties(address, "");
    return properties(address.substr(0, colon), address.substr(colon + 1));
}

/// @brief the adapter that SYNNAX_TEST_CAN_BENCH names as <backend>:<channel>, such as
/// canlib:73-30130-00685-0:12345:1, at 500 kbit/s with a 2 Mbit/s data phase.
/// SYNNAX_TEST_CAN_BENCH_PEER names a second adapter on the same wire in the same way.
class Bench final : public Rig {
    std::optional<synnax::can::Properties> target = named("SYNNAX_TEST_CAN_BENCH");
    std::optional<synnax::can::Properties> other = named("SYNNAX_TEST_CAN_BENCH_PEER");
    Backends backends;

public:
    Bench() {
        if (this->target.has_value()) this->backends = backends::load();
    }

    [[nodiscard]] std::optional<std::string> skipped() const override {
        if (this->target.has_value()) return std::nullopt;
        return "set SYNNAX_TEST_CAN_BENCH=<backend>:<channel> to run on an adapter";
    }

    [[nodiscard]] synnax::can::Properties props() const override {
        return *this->target;
    }

    std::pair<std::unique_ptr<Bus>, x::errors::Error>
    open(const synnax::can::Properties &props) override {
        return can::open(this->backends, props);
    }

    std::unique_ptr<Bus> peer(const bool fd) override {
        if (!this->other.has_value()) return nullptr;
        auto p = *this->other;
        p.fd = fd;
        auto [bus, err] = can::open(this->backends, p);
        EXPECT_FALSE(err) << err;
        return std::move(bus);
    }

    bool fd() override {
        auto p = *this->target;
        p.fd = true;
        return !can::open(this->backends, p).second;
    }
};

std::unique_ptr<Rig> loopback() {
    return std::make_unique<Loopback>();
}

std::unique_ptr<Rig> bench() {
    return std::make_unique<Bench>();
}
}

INSTANTIATE_TEST_SUITE_P(Loopback, BusContract, ::testing::Values(&loopback));
INSTANTIATE_TEST_SUITE_P(Bench, BusContract, ::testing::Values(&bench));
}
