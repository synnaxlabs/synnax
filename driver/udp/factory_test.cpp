// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <chrono>
#include <cstdint>
#include <string>
#include <thread>
#include <vector>

#include "asio/buffer.hpp"
#include "asio/io_context.hpp"
#include "asio/ip/udp.hpp"
#include "gtest/gtest.h"

#include "client/cpp/udp/json.gen.h"
#include "x/cpp/test/test.h"

#include "driver/bus/testutil/core.h"
#include "driver/bus/testutil/testutil.h"
#include "driver/udp/udp.h"

namespace driver::udp {
using namespace bus::testutil;

namespace {
class UDPFactory : public ::testing::Test {
protected:
    asio::io_context io;
    asio::ip::udp::socket device{
        io,
        asio::ip::udp::endpoint(asio::ip::address_v4::loopback(), 0)
    };
    std::uint16_t task_port = 0;
    Core core;
    Factory factory;

    void SetUp() override {
        asio::ip::udp::socket probe(
            this->io,
            asio::ip::udp::endpoint(asio::ip::address_v4::loopback(), 0)
        );
        this->task_port = probe.local_endpoint().port();
    }

    [[nodiscard]] std::string create_device() const {
        synnax::udp::Properties p;
        p.port = this->task_port;
        p.remote_host = "127.0.0.1";
        p.remote_port = this->device.local_endpoint().port();
        return this->core.create_device(MAKE, p.to_json());
    }
};
}

TEST_F(UDPFactory, ConfiguresAReadTaskThatStreamsDecodedValues) {
    const auto lib = core.create_library(
        {binary_message("status", {binary_field("v", 0, 16)})}
    );
    synnax::udp::ReadConfig cfg;
    static_cast<::synnax::bus::ReadConfig &>(
        cfg
    ) = core.read_config(lib, this->create_device());
    const auto data = cfg.messages[0].fields[0].channel;
    auto [t, ok] = this->factory.configure_task(
        core.ctx,
        core.task(READ_TASK_TYPE, cfg.to_json()),
        ""
    );
    ASSERT_TRUE(ok);
    ASSERT_NE(t, nullptr);
    auto streamer = core.stream(data);
    exec(*t, "start");
    const std::vector<std::uint8_t> datagram{0x34, 0x12};
    const asio::ip::udp::endpoint to(asio::ip::address_v4::loopback(), this->task_port);
    for (int i = 0; i < 20; i++)
        this->device.send_to(asio::buffer(datagram), to);
    const auto fr = ASSERT_NIL_P(streamer.read());
    EXPECT_EQ(fr.at<double>(data, 0), 0x1234);
    exec(*t, "stop");
    ASSERT_NIL(streamer.close());
}

TEST_F(UDPFactory, ConfiguresAWriteTask) {
    const auto lib = core.create_library(
        {binary_message("cmd", {binary_field("v", 0)})}
    );
    const auto &m = std::get<synnax::library::MessageEntry>(lib.entries[0]);
    const auto cmd = create_virtual_channel(*core.client, x::telem::FLOAT64_T);
    synnax::udp::WriteConfig cfg;
    cfg.device = this->create_device();
    cfg.library = lib.key;
    cfg.messages = {{
        .message = m.key,
        .fields = {{.field = key(m.fields[0]), .channel = cmd.key}},
    }};
    auto [t, ok] = this->factory.configure_task(
        core.ctx,
        core.task(WRITE_TASK_TYPE, cfg.to_json()),
        ""
    );
    ASSERT_TRUE(ok);
    ASSERT_NE(t, nullptr);
    exec(*t, "start");
    auto writer = ASSERT_NIL_P(core.client->telem.open_writer({.channels = {cmd.key}}));
    // The write task acknowledges its start before its streamer opens.
    for (int i = 0; i < 50 && this->device.available() == 0; i++) {
        ASSERT_NIL(writer.write(x::telem::Frame(cmd.key, x::telem::Series(9.0))));
        std::this_thread::sleep_for(std::chrono::milliseconds(20));
    }
    std::vector<std::uint8_t> buf(16);
    asio::ip::udp::endpoint from;
    const auto n = this->device.receive_from(asio::buffer(buf), from);
    buf.resize(n);
    EXPECT_EQ(buf, (std::vector<std::uint8_t>{9}));
    ASSERT_NIL(writer.close());
    exec(*t, "stop");
}
}
