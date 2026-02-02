// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <gtest/gtest.h>

#include "x/cpp/xtest/xtest.h"

#include "driver/ethercat/topology/topology.h"

namespace ethercat::topology {

class TopologyValidateTest : public ::testing::Test {
protected:
    static device::SlaveProperties make_props(
        const std::string &key,
        uint16_t position,
        uint32_t vendor_id,
        uint32_t product_code
    ) {
        nlohmann::json j = {
            {"serial", 0},
            {"vendor_id", vendor_id},
            {"product_code", product_code},
            {"revision", 1},
            {"name", key},
            {"network", "eth0"},
            {"position", position}
        };
        auto parser = xjson::Parser(j);
        return device::SlaveProperties(parser);
    }

    static SlaveInfo
    make_slave(uint16_t position, uint32_t vendor_id, uint32_t product_code) {
        SlaveInfo info;
        info.position = position;
        info.vendor_id = vendor_id;
        info.product_code = product_code;
        info.revision = 1;
        info.serial = 0;
        info.name = "TestSlave";
        info.state = SlaveState::PRE_OP;
        return info;
    }
};

TEST_F(TopologyValidateTest, MatchingTopologyReturnsNil) {
    std::vector<SlaveInfo> actual = {
        make_slave(1, 0x00000002, 0x12345678),
        make_slave(2, 0x00000002, 0xABCDEF00),
    };

    std::unordered_map<std::string, device::SlaveProperties> expected;
    expected.emplace("dev1", make_props("dev1", 1, 0x00000002, 0x12345678));
    expected.emplace("dev2", make_props("dev2", 2, 0x00000002, 0xABCDEF00));

    ASSERT_NIL(validate(actual, expected));
}

TEST_F(TopologyValidateTest, MissingSlaveAtPositionReturnsMismatch) {
    std::vector<SlaveInfo> actual = {
        make_slave(1, 0x00000002, 0x12345678),
    };

    std::unordered_map<std::string, device::SlaveProperties> expected;
    expected.emplace("dev1", make_props("dev1", 2, 0x00000002, 0xABCDEF00));

    ASSERT_OCCURRED_AS(validate(actual, expected), TOPOLOGY_MISMATCH);
}

TEST_F(TopologyValidateTest, WrongVendorIdReturnsMismatch) {
    std::vector<SlaveInfo> actual = {
        make_slave(1, 0x00000002, 0x12345678),
    };

    std::unordered_map<std::string, device::SlaveProperties> expected;
    expected.emplace("dev1", make_props("dev1", 1, 0x00000099, 0x12345678));

    ASSERT_OCCURRED_AS(validate(actual, expected), TOPOLOGY_MISMATCH);
}

TEST_F(TopologyValidateTest, WrongProductCodeReturnsMismatch) {
    std::vector<SlaveInfo> actual = {
        make_slave(1, 0x00000002, 0x12345678),
    };

    std::unordered_map<std::string, device::SlaveProperties> expected;
    expected.emplace("dev1", make_props("dev1", 1, 0x00000002, 0x87654321));

    ASSERT_OCCURRED_AS(validate(actual, expected), TOPOLOGY_MISMATCH);
}

TEST_F(TopologyValidateTest, EmptyExpectedReturnsNil) {
    std::vector<SlaveInfo> actual = {
        make_slave(1, 0x00000002, 0x12345678),
    };

    std::unordered_map<std::string, device::SlaveProperties> expected;

    ASSERT_NIL(validate(actual, expected));
}

TEST_F(TopologyValidateTest, MultipleDevicesAllMatchReturnsNil) {
    std::vector<SlaveInfo> actual = {
        make_slave(1, 0x00000002, 0x12345678),
        make_slave(2, 0x00000003, 0xABCDEF00),
        make_slave(3, 0x00000004, 0x11111111),
    };

    std::unordered_map<std::string, device::SlaveProperties> expected;
    expected.emplace("dev1", make_props("dev1", 1, 0x00000002, 0x12345678));
    expected.emplace("dev2", make_props("dev2", 2, 0x00000003, 0xABCDEF00));
    expected.emplace("dev3", make_props("dev3", 3, 0x00000004, 0x11111111));

    ASSERT_NIL(validate(actual, expected));
}

}
