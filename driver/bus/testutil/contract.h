// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstddef>
#include <functional>
#include <memory>
#include <ostream>
#include <string>

#include "gtest/gtest.h"

#include "driver/bus/transport.h"

/// @brief the contract that every bus transport keeps, as gtest suites that a
/// transport's tests instantiate with a Rig for each stack they run on. The suites live
/// in contract.cpp, so a test binary links them once.
namespace driver::bus::testutil {
/// @brief plays the device behind a transport for the contract suites.
class Rig {
public:
    virtual ~Rig() = default;

    /// @returns an Opener for the device. The suites open through it more than once.
    virtual Opener opener() = 0;

    /// @returns an Opener for a device that is absent.
    virtual Opener absent() = 0;

    /// @brief sends data from the device to the transport.
    virtual void send(const std::string &data) = 0;

    /// @brief waits up to a second for n bytes from the transport.
    /// @returns the bytes that arrived.
    virtual std::string receive(std::size_t n) = 0;

    /// @brief takes the device away, as when it is unplugged or closes its end.
    virtual void hang_up() = 0;

    /// @brief brings the device back after hang_up.
    virtual void restore() = 0;
};

/// @brief one stack that the contract suites run on.
struct Case {
    /// @brief the name of the instantiation, such as "pty" or "model".
    std::string name;
    /// @brief makes a fresh rig for each test.
    std::function<std::unique_ptr<Rig>()> rig;
};

inline void PrintTo(const Case &c, std::ostream *os) {
    *os << c.name;
}

/// @returns the name of a case, for INSTANTIATE_TEST_SUITE_P.
inline std::string name(const ::testing::TestParamInfo<Case> &info) {
    return info.param.name;
}

/// @brief the contract that every transport keeps.
class TransportContract : public ::testing::TestWithParam<Case> {
protected:
    std::unique_ptr<Rig> rig;

    void SetUp() override { this->rig = GetParam().rig(); }
};

/// @brief the contract that transports with a connection keep: serial and TCP.
class StreamContract : public TransportContract {};
}
