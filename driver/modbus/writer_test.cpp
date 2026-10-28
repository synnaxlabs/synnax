// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/testutil/testutil.h"

#include "driver/modbus/testutil/slave.h"
#include "driver/modbus/write_task.h"

namespace driver::modbus {
namespace {
channel::OutputCoil coil(
    const synnax::channel::Key key,
    const std::uint16_t address,
    const std::string &name
) {
    auto p = x::json::Parser(
        x::json::json{{"type", "coil"}, {"address", address}, {"channel", key}}
    );
    channel::OutputCoil ch(::synnax::modbus::CoilWriteChannel::parse(p));
    ch.name = name;
    return ch;
}

channel::OutputHoldingRegister holding(
    const synnax::channel::Key key,
    const std::uint16_t address,
    const x::telem::DataType &dt,
    const std::string &name
) {
    auto p = x::json::Parser(
        x::json::json{
            {"type", "holding_register"},
            {"address", address},
            {"channel", key},
            {"data_type", dt.name()},
        }
    );
    const auto cfg = ::synnax::modbus::HoldingRegisterWriteChannel::parse(p);
    channel::OutputHoldingRegister ch(cfg, device::ConnectionConfig{});
    ch.name = name;
    return ch;
}

testutil::SlaveConfig coils_on(const int port) {
    testutil::SlaveConfig cfg;
    cfg.port = port;
    for (int i = 0; i < 10; i++)
        cfg.coils[i] = 1;
    return cfg;
}
}

/// @brief it should write only the commanded coil and leave the coils between the
/// task's channels untouched, even after the device changes them.
TEST(CoilWriter, LeavesGapsUntouched) {
    testutil::Slave slave(coils_on(1640));
    ASSERT_NIL(slave.start());
    auto dev = ASSERT_NIL_P(device::Manager().acquire({"127.0.0.1", 1640}));
    CoilWriter writer({coil(1, 3, "a"), coil(2, 7, "b")});

    x::telem::Frame first(1);
    first.emplace(1, x::telem::Series(static_cast<uint8_t>(0)));
    ASSERT_NIL(writer.write(dev, first));
    slave.set_coil(5, 0);
    x::telem::Frame second(1);
    second.emplace(2, x::telem::Series(static_cast<uint8_t>(0)));
    ASSERT_NIL(writer.write(dev, second));

    EXPECT_EQ(slave.get_coil(3), 0);
    EXPECT_EQ(slave.get_coil(4), 1);
    EXPECT_EQ(slave.get_coil(5), 0);
    EXPECT_EQ(slave.get_coil(6), 1);
    EXPECT_EQ(slave.get_coil(7), 0);
    EXPECT_EQ(slave.requests(MODBUS_FC_READ_COILS), 0);
    EXPECT_EQ(slave.requests(MODBUS_FC_WRITE_MULTIPLE_COILS), 2);
    slave.stop();
}

/// @brief it should write coils on a device that rejects reading coils.
TEST(CoilWriter, WritesWithoutReadingCoils) {
    auto cfg = coils_on(1641);
    cfg.reject_function_codes = {MODBUS_FC_READ_COILS};
    testutil::Slave slave(cfg);
    ASSERT_NIL(slave.start());
    auto dev = ASSERT_NIL_P(device::Manager().acquire({"127.0.0.1", 1641}));
    CoilWriter writer({coil(1, 2, "a"), coil(2, 8, "b")});

    x::telem::Frame fr(1);
    fr.emplace(1, x::telem::Series(static_cast<uint8_t>(0)));
    ASSERT_NIL(writer.write(dev, fr));
    EXPECT_EQ(slave.get_coil(2), 0);
    EXPECT_EQ(slave.get_coil(5), 1);
    slave.stop();
}

/// @brief it should turn a coil on for any non-zero command of any data type.
TEST(CoilWriter, NonZeroCommandOfAnyType) {
    testutil::SlaveConfig cfg;
    cfg.port = 1642;
    testutil::Slave slave(cfg);
    ASSERT_NIL(slave.start());
    auto dev = ASSERT_NIL_P(device::Manager().acquire({"127.0.0.1", 1642}));
    CoilWriter writer({coil(1, 0, "a")});

    x::telem::Frame on(1);
    on.emplace(1, x::telem::Series(1.0f));
    ASSERT_NIL(writer.write(dev, on));
    EXPECT_EQ(slave.get_coil(0), 1);

    x::telem::Frame off(1);
    off.emplace(1, x::telem::Series(0.0));
    ASSERT_NIL(writer.write(dev, off));
    EXPECT_EQ(slave.get_coil(0), 0);

    x::telem::Frame wide(1);
    wide.emplace(1, x::telem::Series(static_cast<uint16_t>(256)));
    ASSERT_NIL(writer.write(dev, wide));
    EXPECT_EQ(slave.get_coil(0), 1);
    slave.stop();
}

/// @brief it should write adjacent coils in one request and coils with a gap in
/// separate requests.
TEST(CoilWriter, MergesAdjacentCoils) {
    testutil::SlaveConfig cfg;
    cfg.port = 1643;
    testutil::Slave slave(cfg);
    ASSERT_NIL(slave.start());
    auto dev = ASSERT_NIL_P(device::Manager().acquire({"127.0.0.1", 1643}));
    CoilWriter writer({coil(1, 3, "a"), coil(2, 4, "b"), coil(3, 6, "c")});

    x::telem::Frame fr(3);
    fr.emplace(1, x::telem::Series(static_cast<uint8_t>(1)));
    fr.emplace(2, x::telem::Series(static_cast<uint8_t>(1)));
    fr.emplace(3, x::telem::Series(static_cast<uint8_t>(1)));
    ASSERT_NIL(writer.write(dev, fr));
    EXPECT_EQ(slave.get_coil(3), 1);
    EXPECT_EQ(slave.get_coil(4), 1);
    EXPECT_EQ(slave.get_coil(5), 0);
    EXPECT_EQ(slave.get_coil(6), 1);
    EXPECT_EQ(slave.requests(MODBUS_FC_WRITE_MULTIPLE_COILS), 2);
    slave.stop();
}

/// @brief it should write adjacent registers in one request and leave the gap
/// between channels untouched.
TEST(RegisterWriter, MergesAdjacentRegisters) {
    testutil::SlaveConfig cfg;
    cfg.port = 1644;
    cfg.holding_registers[5] = static_cast<uint16_t>(999);
    testutil::Slave slave(cfg);
    ASSERT_NIL(slave.start());
    auto dev = ASSERT_NIL_P(device::Manager().acquire({"127.0.0.1", 1644}));
    RegisterWriter writer(
        {holding(1, 0, x::telem::FLOAT32_T, "a"),
         holding(2, 2, x::telem::UINT16_T, "b"),
         holding(3, 6, x::telem::UINT16_T, "c")}
    );

    x::telem::Frame fr(3);
    fr.emplace(1, x::telem::Series(1.5f));
    fr.emplace(2, x::telem::Series(static_cast<uint16_t>(42)));
    fr.emplace(3, x::telem::Series(static_cast<uint16_t>(7)));
    ASSERT_NIL(writer.write(dev, fr));

    uint16_t words[2] = {slave.get_holding_register(0), slave.get_holding_register(1)};
    const auto [value, err] = util::parse_register_value(
        words,
        x::telem::FLOAT32_T,
        false,
        false
    );
    ASSERT_NIL(err);
    EXPECT_FLOAT_EQ(std::get<float>(value), 1.5f);
    EXPECT_EQ(slave.get_holding_register(2), 42);
    EXPECT_EQ(slave.get_holding_register(5), 999);
    EXPECT_EQ(slave.get_holding_register(6), 7);
    EXPECT_EQ(slave.requests(MODBUS_FC_READ_HOLDING_REGISTERS), 0);
    EXPECT_EQ(slave.requests(MODBUS_FC_WRITE_MULTIPLE_REGISTERS), 2);
    slave.stop();
}

/// @brief it should return a critical error that names the channel and addresses of
/// a write the server rejects.
TEST(RegisterWriter, IllegalAddressNamesChannel) {
    testutil::SlaveConfig cfg;
    cfg.port = 1645;
    testutil::Slave slave(cfg);
    ASSERT_NIL(slave.start());
    auto dev = ASSERT_NIL_P(device::Manager().acquire({"127.0.0.1", 1645}));
    RegisterWriter writer({holding(1, 500, x::telem::UINT16_T, "setpoint")});

    x::telem::Frame fr(1);
    fr.emplace(1, x::telem::Series(static_cast<uint16_t>(1)));
    const auto err = writer.write(dev, fr);
    ASSERT_OCCURRED_AS(err, device::CRITICAL_ERROR);
    EXPECT_EQ(err.data, "setpoint (holding registers 500): Illegal data address");
    slave.stop();
}

/// @brief it should find the channel whose addresses overlap a new one.
TEST(FindOverlap, MultiRegisterChannels) {
    const std::vector chs{
        holding(1, 0, x::telem::FLOAT64_T, "a"),
        holding(2, 10, x::telem::UINT16_T, "b"),
    };
    EXPECT_EQ(find_overlap(chs, holding(3, 3, x::telem::UINT16_T, "c")), chs.data());
    EXPECT_EQ(find_overlap(chs, holding(3, 9, x::telem::INT32_T, "c")), &chs[1]);
    EXPECT_EQ(find_overlap(chs, holding(3, 4, x::telem::INT32_T, "c")), nullptr);
    EXPECT_EQ(find_overlap(chs, holding(3, 11, x::telem::UINT16_T, "c")), nullptr);
}
}
