// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <memory>
#include <optional>
#include <string>
#include <utility>

#include "gtest/gtest.h"

#include "x/cpp/test/test.h"

#include "driver/can/can.h"

/// @brief the contract every can::Bus keeps, as one test suite that runs on each rig
/// that instantiates BusContract. A rig's wire may carry other traffic, so the cases
/// look only for their own frames.
namespace driver::can::contract {
/// @brief a channel under test and another node on its wire.
class Rig {
public:
    virtual ~Rig() = default;

    /// @returns why this run leaves the rig out, such as an unset environment variable.
    [[nodiscard]] virtual std::optional<std::string> skipped() const {
        return std::nullopt;
    }

    /// @returns the host capability the rig lacks, such as vcan0.
    [[nodiscard]] virtual std::optional<std::string> missing() const {
        return std::nullopt;
    }

    /// @returns the properties of the channel under test.
    [[nodiscard]] virtual synnax::can::Properties props() const = 0;

    /// @returns a bus on the channel under test.
    [[nodiscard]] virtual std::pair<std::unique_ptr<Bus>, x::errors::Error>
    open(const synnax::can::Properties &props) = 0;

    /// @returns a bus for another node on the wire, or nullptr when the rig has none.
    [[nodiscard]] virtual std::unique_ptr<Bus> peer(bool fd) = 0;

    /// @returns true when the channel runs CAN FD.
    [[nodiscard]] virtual bool fd() = 0;

    /// @brief unplugs the channel's adapter.
    /// @returns false when the rig cannot unplug it.
    virtual bool unplug() { return false; }

    /// @brief plugs the channel's adapter back in.
    virtual void plug() {}
};

/// @brief makes a rig for one test.
using Make = std::unique_ptr<Rig> (*)();

class BusContract : public ::testing::TestWithParam<Make> {
protected:
    std::unique_ptr<Rig> rig;

    void SetUp() override {
        this->rig = this->GetParam()();
        if (const auto reason = this->rig->skipped()) GTEST_SKIP() << *reason;
        if (const auto reason = this->rig->missing()) SKIP_MISSING_CAPABILITY(*reason);
    }

    [[nodiscard]] synnax::can::Properties
    props(const bool fd = false, const bool listen_only = false) const {
        auto p = this->rig->props();
        p.fd = fd;
        p.listen_only = listen_only;
        return p;
    }

    std::unique_ptr<Bus> open(const bool fd = false, const bool listen_only = false) {
        auto [bus, err] = this->rig->open(this->props(fd, listen_only));
        EXPECT_FALSE(err) << err;
        return std::move(bus);
    }
};
}
