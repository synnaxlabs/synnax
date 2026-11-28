// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <string>

#include "gtest/gtest.h"

#include "client/cpp/serial/json.gen.h"
#include "client/cpp/serial/types.gen.h"
#include "client/cpp/testutil/testutil.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/json/json.h"
#include "x/cpp/testutil/testutil.h"

#include "driver/serial/ports.h"

namespace driver::serial {
namespace {
/// @brief RetrieveDevice retrieves serial devices from a live Core.
class RetrieveDevice : public ::testing::Test {
protected:
    synnax::Synnax client = new_test_client();

    /// @returns a device named boiler of make with properties that it creates.
    [[nodiscard]] synnax::device::Device
    create(const x::json::json &properties, const std::string &make = "serial") const {
        const auto rack = ASSERT_NIL_P(this->client.racks.create("serial_retrieve"));
        synnax::device::Device dev{
            .key = make_unique_channel_name("serial_device"),
            .rack = rack.key,
            .location = "port",
            .make = make,
            .model = "serial",
            .name = "boiler",
        };
        dev.properties = properties;
        EXPECT_NIL(this->client.devices.create(dev));
        return dev;
    }
};
}

/// @brief it should return the key, the name, and the properties of a device.
TEST_F(RetrieveDevice, ReturnsKeyNameAndProperties) {
    synnax::serial::Properties props;
    props.port = "/dev/ttyUSB0";
    props.baud_rate = 115200;
    const auto dev = this->create(props.to_json());
    const auto got = ASSERT_NIL_P(retrieve_device(this->client.devices, dev.key));
    EXPECT_EQ(got.key, dev.key);
    EXPECT_EQ(got.name, "boiler");
    EXPECT_EQ(got.props.port, "/dev/ttyUSB0");
    EXPECT_EQ(got.props.baud_rate, 115200);
}

/// @brief it should fail on a device of another make whose properties are not serial
/// properties.
TEST_F(RetrieveDevice, FailsOnInvalidProperties) {
    const auto dev = this->create(x::json::json{{"port", 12}}, "labjack");
    ASSERT_OCCURRED_AS_P(
        retrieve_device(this->client.devices, dev.key),
        x::errors::VALIDATION
    );
}
}
