// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <memory>
#include <string>
#include <vector>

#include "asio/buffer.hpp"
#include "asio/io_context.hpp"
#include "asio/ip/udp.hpp"
#include "gtest/gtest.h"

#include "client/cpp/udp/json.gen.h"
#include "x/cpp/test/test.h"

#include "driver/bus/read.h"
#include "driver/bus/task.h"
#include "driver/bus/testutil/testutil.h"
#include "driver/bus/write.h"
#include "driver/pipeline/mock/pipeline.h"
#include "driver/udp/socket.h"

namespace driver::udp {
using namespace bus::testutil;

namespace {
/// @brief runs bus tasks against a device the test plays with a localhost socket.
class UDPTask : public ::testing::Test {
protected:
    asio::io_context io;
    asio::ip::udp::socket device{
        io,
        asio::ip::udp::endpoint(asio::ip::address_v4::loopback(), 0)
    };
    std::uint16_t task_port = 0;
    std::shared_ptr<bus::Connections>
        connections = std::make_shared<bus::Connections>();
    std::shared_ptr<task::MockContext> ctx = std::make_shared<task::MockContext>(
        nullptr
    );
    synnax::task::Task task{.key = x::uuid::create(), .name = "udp"};
    std::shared_ptr<std::vector<x::telem::Frame>>
        writes = std::make_shared<std::vector<x::telem::Frame>>();

    void SetUp() override {
        asio::ip::udp::socket probe(
            this->io,
            asio::ip::udp::endpoint(asio::ip::address_v4::loopback(), 0)
        );
        this->task_port = probe.local_endpoint().port();
    }

    [[nodiscard]] synnax::udp::Properties props() const {
        synnax::udp::Properties p;
        p.port = this->task_port;
        p.remote_host = "127.0.0.1";
        p.remote_port = this->device.local_endpoint().port();
        return p;
    }

    void send(const std::vector<std::uint8_t> &datagram) {
        this->device.send_to(
            asio::buffer(datagram),
            asio::ip::udp::endpoint(asio::ip::address_v4::loopback(), this->task_port)
        );
    }

    std::vector<std::uint8_t> receive() {
        std::vector<std::uint8_t> buf(1024);
        asio::ip::udp::endpoint from;
        const auto n = this->device.receive_from(asio::buffer(buf), from);
        buf.resize(n);
        return buf;
    }

    std::unique_ptr<common::ReadTask> read_task(bus::ReadConfig cfg) {
        return std::make_unique<common::ReadTask>(
            this->task,
            this->ctx,
            x::breaker::default_config(this->task.name),
            std::make_unique<bus::Source>(
                std::move(cfg),
                bus::acquirer<Socket>(this->connections, "dev", this->props())
            ),
            std::make_shared<pipeline::mock::WriterFactory>(this->writes)
        );
    }

    std::unique_ptr<common::WriteTask> write_task(
        bus::WriteConfig cfg,
        const std::shared_ptr<std::vector<x::telem::Frame>> &commands
    ) {
        return std::make_unique<common::WriteTask>(
            this->task,
            this->ctx,
            x::breaker::default_config(this->task.name),
            std::make_unique<bus::Sink>(
                std::move(cfg),
                std::make_unique<bus::ConnectionTransmitter>(
                    bus::acquirer<Socket>(this->connections, "dev", this->props())
                ),
                this->ctx,
                this->task
            ),
            nullptr,
            pipeline::mock::simple_streamer_factory({7}, commands)
        );
    }
};
}

TEST_F(UDPTask, DecodesEachDatagramAsOneMessage) {
    auto kind = binary_field("kind", 0);
    const auto a = binary_message(
        "a",
        {kind, binary_field("v", 8, 16)},
        synnax::library::FieldIdentifier{.field = kind.key, .value = 1}
    );
    auto kind_b = binary_field("kind", 0);
    const auto b = binary_message(
        "b",
        {kind_b, binary_field("v", 8)},
        synnax::library::FieldIdentifier{.field = kind_b.key, .value = 2}
    );
    auto t = this->read_task(read_config({a, b}, std::nullopt));
    t->start("start");
    ASSERT_EVENTUALLY_TRUE([&] {
        this->send({0x01, 0x34, 0x12});
        this->send({0x02, 0x07});
        return !values(*this->writes, 4).empty();
    }());
    t->stop("stop", true);
    EXPECT_EQ(values(*this->writes, 2)[0], 0x1234);
    EXPECT_EQ(values(*this->writes, 1)[0], 1);
    EXPECT_EQ(values(*this->writes, 4)[0], 7);
}

TEST_F(UDPTask, PollsTheRemoteAndDecodesItsReply) {
    auto m = binary_message("status", {binary_field("v", 0)});
    m.query = R"(\x10\x20)";
    auto t = this->read_task(read_config(
        {m},
        std::nullopt,
        nullptr,
        {.rate = x::telem::Rate(20), .timeout = x::telem::SECOND}
    ));
    t->start("start");
    EXPECT_EQ(this->receive(), (std::vector<std::uint8_t>{0x10, 0x20}));
    this->send({0x2A});
    ASSERT_EVENTUALLY_EQ(values(*this->writes, 1).size(), 1);
    t->stop("stop", true);
    EXPECT_EQ(values(*this->writes, 1)[0], 42);
}

TEST_F(UDPTask, SendsEachCommandAsOneDatagram) {
    const auto m = binary_message("cmd", {binary_field("a", 0), binary_field("b", 8)});
    auto commands = std::make_shared<std::vector<x::telem::Frame>>();
    commands->emplace_back(7, x::telem::Series(5.0, x::telem::FLOAT64_T));
    auto t = this->write_task(write_config(m, {{7, 1}}, std::nullopt), commands);
    t->start("start");
    EXPECT_EQ(this->receive(), (std::vector<std::uint8_t>{0x00, 0x05}));
    t->stop("stop", true);
}

TEST_F(UDPTask, ReadsAndWritesOneSocketAtOnce) {
    auto reader = this->read_task(
        read_config({binary_message("status", {binary_field("v", 0)})}, std::nullopt)
    );
    auto commands = std::make_shared<std::vector<x::telem::Frame>>();
    for (int i = 1; i <= 5; i++)
        commands->emplace_back(7, x::telem::Series(double(i), x::telem::FLOAT64_T));
    auto writer = this->write_task(
        write_config(
            binary_message("cmd", {binary_field("v", 0)}),
            {{7, 0}},
            std::nullopt
        ),
        commands
    );
    reader->start("start");
    writer->start("start");
    for (std::uint8_t i = 1; i <= 5; i++) {
        std::vector<std::uint8_t> buf(16);
        asio::ip::udp::endpoint from;
        buf.resize(this->device.receive_from(asio::buffer(buf), from));
        EXPECT_EQ(buf, std::vector<std::uint8_t>{i});
        EXPECT_EQ(from.port(), this->task_port);
    }
    ASSERT_EVENTUALLY_TRUE([&] {
        this->send({42});
        return !values(*this->writes, 1).empty();
    }());
    writer->stop("stop", true);
    reader->stop("stop", true);
    EXPECT_EQ(values(*this->writes, 1)[0], 42);
}
}
