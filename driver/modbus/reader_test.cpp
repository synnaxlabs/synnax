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

#include "driver/modbus/read_task.h"
#include "driver/modbus/testutil/slave.h"

namespace driver::modbus {
namespace {
channel::InputRegister holding(
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
    const auto cfg = ::synnax::modbus::HoldingRegisterReadChannel::parse(p);
    channel::InputRegister ch(cfg, device::ConnectionConfig{});
    ch.ch = synnax::channel::Channel{.key = key, .name = name, .data_type = dt};
    return ch;
}

channel::InputDiscrete coil(
    const synnax::channel::Key key,
    const std::uint16_t address,
    const std::string &name
) {
    auto p = x::json::Parser(
        x::json::json{{"type", "coil"}, {"address", address}, {"channel", key}}
    );
    channel::InputDiscrete ch(::synnax::modbus::CoilReadChannel::parse(p));
    ch.ch = synnax::channel::Channel{
        .key = key,
        .name = name,
        .data_type = x::telem::UINT8_T,
    };
    return ch;
}

/// @brief reads once into a frame with one series per reader channel.
std::pair<x::telem::Frame, x::errors::Error>
read_once(Reader &reader, const std::shared_ptr<device::Device> &dev) {
    const auto chs = reader.sy_channels();
    x::telem::Frame fr(chs.size());
    for (const auto &ch: chs)
        fr.emplace(ch.key, x::telem::Series(ch.data_type, 1));
    std::size_t offset = 0;
    auto err = reader.read(dev, fr, offset);
    return {std::move(fr), err};
}
}

/// @brief it should read a multi-register value and a register inside it, sizing
/// the request by the value that ends last.
TEST(RegisterReader, OverlappingChannels) {
    testutil::SlaveConfig cfg;
    cfg.port = 1630;
    cfg.holding_registers[10] = 3.14159;
    testutil::Slave slave(cfg);
    ASSERT_NIL(slave.start());
    auto dev = ASSERT_NIL_P(device::Manager().acquire({"127.0.0.1", 1630}));

    RegisterReader reader(
        device::HoldingRegister,
        {holding(1, 10, x::telem::FLOAT64_T, "value"),
         holding(2, 11, x::telem::UINT16_T, "word")}
    );
    const auto [fr, err] = read_once(reader, dev);
    ASSERT_NIL(err);
    EXPECT_DOUBLE_EQ(fr.at<double>(1, 0), 3.14159);
    EXPECT_EQ(fr.at<uint16_t>(2, 0), slave.get_holding_register(11));
    EXPECT_EQ(slave.requests(MODBUS_FC_READ_HOLDING_REGISTERS), 1);
    slave.stop();
}

/// @brief it should split a span wider than one request into several requests.
TEST(RegisterReader, SplitsWideSpan) {
    testutil::SlaveConfig cfg;
    cfg.port = 1631;
    cfg.holding_registers[0] = static_cast<uint16_t>(11);
    cfg.holding_registers[200] = static_cast<uint16_t>(22);
    testutil::Slave slave(cfg);
    ASSERT_NIL(slave.start());
    auto dev = ASSERT_NIL_P(device::Manager().acquire({"127.0.0.1", 1631}));

    RegisterReader reader(
        device::HoldingRegister,
        {holding(1, 200, x::telem::UINT16_T, "high"),
         holding(2, 0, x::telem::UINT16_T, "low")}
    );
    const auto [fr, err] = read_once(reader, dev);
    ASSERT_NIL(err);
    EXPECT_EQ(fr.at<uint16_t>(1, 0), 22);
    EXPECT_EQ(fr.at<uint16_t>(2, 0), 11);
    EXPECT_EQ(slave.requests(MODBUS_FC_READ_HOLDING_REGISTERS), 2);
    slave.stop();
}

/// @brief it should return a critical error that names the channels and addresses
/// of a request the server rejects.
TEST(RegisterReader, IllegalAddressNamesChannels) {
    testutil::SlaveConfig cfg;
    cfg.port = 1632;
    testutil::Slave slave(cfg);
    ASSERT_NIL(slave.start());
    auto dev = ASSERT_NIL_P(device::Manager().acquire({"127.0.0.1", 1632}));

    RegisterReader reader(
        device::HoldingRegister,
        {holding(1, 500, x::telem::FLOAT32_T, "pressure")}
    );
    const auto [fr, err] = read_once(reader, dev);
    ASSERT_OCCURRED_AS(err, device::CRITICAL_ERROR);
    EXPECT_EQ(err.data, "pressure (holding registers 500-501): Illegal data address");
    slave.stop();
}

/// @brief it should cast a register value into the data type of its Synnax channel.
TEST(RegisterReader, CastsToChannelDataType) {
    testutil::SlaveConfig cfg;
    cfg.port = 1634;
    cfg.holding_registers[4] = static_cast<uint16_t>(1234);
    testutil::Slave slave(cfg);
    ASSERT_NIL(slave.start());
    auto dev = ASSERT_NIL_P(device::Manager().acquire({"127.0.0.1", 1634}));

    auto level = holding(1, 4, x::telem::UINT16_T, "level");
    level.ch.data_type = x::telem::FLOAT32_T;
    auto low = holding(2, 4, x::telem::UINT16_T, "low");
    low.ch.data_type = x::telem::UINT8_T;
    RegisterReader reader(device::HoldingRegister, {level, low});
    const auto [fr, err] = read_once(reader, dev);
    ASSERT_NIL(err);
    EXPECT_FLOAT_EQ(fr.at<float>(1, 0), 1234.0F);
    EXPECT_EQ(fr.at<uint8_t>(2, 0), static_cast<uint8_t>(1234));
    EXPECT_EQ(fr.series->at(1).byte_size(), 1U);
    slave.stop();
}

/// @brief it should cast a coil into the data type of its Synnax channel.
TEST(BitReader, CastsToChannelDataType) {
    testutil::SlaveConfig cfg;
    cfg.port = 1635;
    cfg.coils[2] = 1;
    testutil::Slave slave(cfg);
    ASSERT_NIL(slave.start());
    auto dev = ASSERT_NIL_P(device::Manager().acquire({"127.0.0.1", 1635}));

    auto state = coil(1, 2, "state");
    state.ch.data_type = x::telem::FLOAT64_T;
    BitReader reader(device::Coil, {state});
    const auto [fr, err] = read_once(reader, dev);
    ASSERT_NIL(err);
    EXPECT_DOUBLE_EQ(fr.at<double>(1, 0), 1.0);
    slave.stop();
}

/// @brief it should return an unreachable error without channel names when the
/// server drops the connection, then read again on a new connection.
TEST(BitReader, ReconnectsAfterDroppedConnection) {
    testutil::SlaveConfig cfg;
    cfg.port = 1633;
    cfg.coils[3] = 1;
    testutil::Slave slave(cfg);
    ASSERT_NIL(slave.start());
    auto dev = ASSERT_NIL_P(device::Manager().acquire({"127.0.0.1", 1633}));

    BitReader reader(device::Coil, {coil(1, 3, "valve")});
    // The slave drops only accepted connections, so read once to get accepted first.
    ASSERT_NIL(read_once(reader, dev).second);
    slave.drop_connections();
    const auto [dropped, drop_err] = read_once(reader, dev);
    ASSERT_OCCURRED_AS(drop_err, device::UNREACHABLE);
    EXPECT_EQ(drop_err.data.find("valve"), std::string::npos);

    const auto [fr, err] = read_once(reader, dev);
    ASSERT_NIL(err);
    EXPECT_EQ(fr.at<uint8_t>(1, 0), 1);
    slave.stop();
}
}
