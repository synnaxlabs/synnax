// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <filesystem>
#include <fstream>
#include <memory>
#include <optional>
#include <string>
#include <utility>

#include "driver/can/contract.h"
#include "driver/can/socketcan/socketcan.h"

namespace driver::can::contract {
namespace {
const std::filesystem::path VCAN = "/sys/class/net/vcan0";

/// @brief the virtual interface vcan0, with a second socket on it as the peer. Create
/// it with: ip link add dev vcan0 type vcan && ip link set vcan0 mtu 72 up
class Vcan final : public Rig {
    std::shared_ptr<can::Backend> backend = socketcan::load();

public:
    [[nodiscard]] std::optional<std::string> missing() const override {
        if (std::filesystem::exists(VCAN)) return std::nullopt;
        return "vcan0 is absent";
    }

    [[nodiscard]] synnax::can::Properties props() const override {
        synnax::can::Properties p;
        p.backend = synnax::can::BACKEND_SOCKETCAN;
        p.channel = "vcan0";
        return p;
    }

    std::pair<std::unique_ptr<Bus>, x::errors::Error>
    open(const synnax::can::Properties &props) override {
        return this->backend->open(props);
    }

    std::unique_ptr<Bus> peer(const bool fd) override {
        auto p = this->props();
        p.fd = fd;
        auto [bus, err] = this->backend->open(p);
        EXPECT_FALSE(err) << err;
        return std::move(bus);
    }

    bool fd() override {
        std::ifstream in(VCAN / "mtu");
        std::string mtu;
        std::getline(in, mtu);
        return mtu == "72";
    }
};

std::unique_ptr<Rig> vcan() {
    return std::make_unique<Vcan>();
}
}

INSTANTIATE_TEST_SUITE_P(SocketCAN, BusContract, ::testing::Values(&vcan));
}
