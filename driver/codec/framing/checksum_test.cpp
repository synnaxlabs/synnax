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

#include "driver/codec/framing/checksum.h"

namespace driver::codec::framing {
namespace {
const std::string CHECK = "123456789";

std::span<const std::uint8_t> check() {
    return {reinterpret_cast<const std::uint8_t *>(CHECK.data()), CHECK.size()};
}
}

TEST(Checksum, ComputesTheCRC16CCITTFalseCheckValue) {
    EXPECT_EQ(checksum(Checksum::CRC16_CCITT_FALSE, check()), 0x29B1);
    EXPECT_EQ(width(Checksum::CRC16_CCITT_FALSE), 2);
}

TEST(Checksum, ComputesTheCRC16ModbusCheckValue) {
    EXPECT_EQ(checksum(Checksum::CRC16_MODBUS, check()), 0x4B37);
    EXPECT_EQ(width(Checksum::CRC16_MODBUS), 2);
}

TEST(Checksum, ComputesTheCRC32CheckValue) {
    EXPECT_EQ(checksum(Checksum::CRC32, check()), 0xCBF43926);
    EXPECT_EQ(width(Checksum::CRC32), 4);
}

TEST(Checksum, HasNoWidthForNone) {
    EXPECT_EQ(checksum(Checksum::NONE, check()), 0);
    EXPECT_EQ(width(Checksum::NONE), 0);
}
}
