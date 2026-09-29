// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <chrono>
#include <string>
#include <thread>

#include "asio/buffer.hpp"
#include "asio/io_context.hpp"
#include "asio/ip/tcp.hpp"
#include "asio/read.hpp"
#include "asio/write.hpp"
#include "gtest/gtest.h"

#include "client/cpp/tcp/json.gen.h"
#include "x/cpp/test/test.h"

#include "driver/bus/testutil/core.h"
#include "driver/bus/testutil/testutil.h"
#include "driver/tcp/tcp.h"

namespace driver::tcp {
using namespace bus::testutil;

namespace {
class TCPFactory : public ::testing::Test {
protected:
    asio::io_context io;
    asio::ip::tcp::acceptor acceptor{
        io,
        asio::ip::tcp::endpoint(asio::ip::address_v4::loopback(), 0)
    };
    Core core;
    Factory factory;

    [[nodiscard]] std::string create_device() const {
        synnax::tcp::Properties p;
        p.host = "127.0.0.1";
        p.port = this->acceptor.local_endpoint().port();
        return this->core.create_device(MAKE, p.to_json());
    }
};
}

TEST_F(TCPFactory, ConfiguresAReadTaskThatStreamsDecodedValues) {
    const auto lib = core.create_library(
        {text_message("status", {delimited_field("v", 0)})}
    );
    synnax::tcp::ReadConfig cfg;
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
    auto peer = this->acceptor.accept();
    asio::write(peer, asio::buffer(std::string("7.5\n")));
    const auto fr = ASSERT_NIL_P(streamer.read());
    EXPECT_EQ(fr.at<double>(data, 0), 7.5);
    exec(*t, "stop");
    ASSERT_NIL(streamer.close());
}

TEST_F(TCPFactory, ConfiguresAWriteTask) {
    const auto lib = core.create_library(
        {text_message("set", {tagged_field("v", "VOLT ")})}
    );
    const auto &m = std::get<synnax::library::MessageEntry>(lib.entries[0]);
    const auto cmd = create_virtual_channel(*core.client, x::telem::FLOAT64_T);
    synnax::tcp::WriteConfig cfg;
    cfg.device = this->create_device();
    cfg.library = lib.key;
    cfg.messages = {{
        .message = m.key,
        .fields = {{.field = field_key(m, 0), .channel = cmd.key}},
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
    // The write task acknowledges its start before its streamer opens, and it connects
    // on its first send.
    this->acceptor.non_blocking(true);
    asio::ip::tcp::socket peer(this->io);
    std::error_code ec = asio::error::would_block;
    for (int i = 0; i < 50 && ec; i++) {
        ASSERT_NIL(writer.write(x::telem::Frame(cmd.key, x::telem::Series(4.0))));
        std::this_thread::sleep_for(std::chrono::milliseconds(20));
        this->acceptor.accept(peer, ec);
    }
    ASSERT_FALSE(ec) << ec.message();
    peer.non_blocking(false);
    std::string got(7, '\0');
    asio::read(peer, asio::buffer(got));
    EXPECT_EQ(got, "VOLT 4\n");
    ASSERT_NIL(writer.close());
    exec(*t, "stop");
}

TEST_F(TCPFactory, SharesOneConnectionBetweenTheTasksOfADevice) {
    const auto lib = core.create_library(
        {text_message("status", {delimited_field("v", 0)}),
         text_message("set", {tagged_field("v", "VOLT ")})}
    );
    const auto dev = this->create_device();
    synnax::tcp::ReadConfig read_cfg;
    static_cast<::synnax::bus::ReadConfig &>(read_cfg) = core.read_config(lib, dev);
    read_cfg.messages.resize(1);
    const auto data = read_cfg.messages[0].fields[0].channel;
    const auto &set = std::get<synnax::library::MessageEntry>(lib.entries[1]);
    const auto cmd = create_virtual_channel(*core.client, x::telem::FLOAT64_T);
    synnax::tcp::WriteConfig write_cfg;
    write_cfg.device = dev;
    write_cfg.library = lib.key;
    write_cfg.messages = {{
        .message = set.key,
        .fields = {{.field = field_key(set, 0), .channel = cmd.key}},
    }};
    auto [reader, read_ok] = this->factory.configure_task(
        core.ctx,
        core.task(READ_TASK_TYPE, read_cfg.to_json()),
        ""
    );
    auto [writer, write_ok] = this->factory.configure_task(
        core.ctx,
        core.task(WRITE_TASK_TYPE, write_cfg.to_json()),
        ""
    );
    ASSERT_NE(reader, nullptr);
    ASSERT_NE(writer, nullptr);
    auto streamer = core.stream(data);
    exec(*reader, "start");
    exec(*writer, "start");
    auto peer = this->acceptor.accept();
    asio::write(peer, asio::buffer(std::string("7.5\n")));
    EXPECT_EQ(ASSERT_NIL_P(streamer.read()).at<double>(data, 0), 7.5);
    auto cmd_writer = ASSERT_NIL_P(
        core.client->telem.open_writer({.channels = {cmd.key}})
    );
    // The write task acknowledges its start before its streamer opens.
    for (int i = 0; i < 50 && peer.available() == 0; i++) {
        ASSERT_NIL(cmd_writer.write(x::telem::Frame(cmd.key, x::telem::Series(4.0))));
        std::this_thread::sleep_for(std::chrono::milliseconds(20));
    }
    std::string got(7, '\0');
    asio::read(peer, asio::buffer(got));
    EXPECT_EQ(got, "VOLT 4\n");
    ASSERT_NIL(cmd_writer.close());
    exec(*writer, "stop");
    exec(*reader, "stop");
    ASSERT_NIL(streamer.close());
    this->acceptor.non_blocking(true);
    std::error_code ec;
    this->acceptor.accept(ec);
    EXPECT_EQ(ec, asio::error::would_block);
}

TEST_F(TCPFactory, ConfiguresTheScanTask) {
    auto [t, ok] = this->factory.configure_task(
        core.ctx,
        core.task(SCAN_TASK_TYPE, x::json::json::object()),
        ""
    );
    ASSERT_TRUE(ok);
    ASSERT_NE(t, nullptr);
    t->stop(false);
}
}
