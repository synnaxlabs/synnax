// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <chrono>
#include <istream>
#include <memory>
#include <string>
#include <thread>
#include <vector>

#include "asio/buffer.hpp"
#include "asio/io_context.hpp"
#include "asio/ip/tcp.hpp"
#include "asio/read.hpp"
#include "asio/read_until.hpp"
#include "asio/streambuf.hpp"
#include "asio/write.hpp"
#include "gtest/gtest.h"

#include "x/cpp/test/test.h"

#include "driver/bus/read.h"
#include "driver/bus/task.h"
#include "driver/bus/testutil/testutil.h"
#include "driver/bus/write.h"
#include "driver/pipeline/mock/pipeline.h"
#include "driver/tcp/client.h"
#include "driver/tcp/scan_task.h"

namespace driver::tcp {
using namespace bus::testutil;

namespace {
/// @brief runs bus tasks against a device the test plays on a localhost listener.
class TCPTask : public ::testing::Test {
protected:
    asio::io_context io;
    asio::ip::tcp::acceptor acceptor{
        io,
        asio::ip::tcp::endpoint(asio::ip::address_v4::loopback(), 0)
    };
    std::shared_ptr<bus::Connections>
        connections = std::make_shared<bus::Connections>();
    std::shared_ptr<task::MockContext> ctx = std::make_shared<task::MockContext>(
        nullptr
    );
    synnax::task::Task task{.key = x::uuid::create(), .name = "tcp"};
    std::shared_ptr<std::vector<x::telem::Frame>>
        writes = std::make_shared<std::vector<x::telem::Frame>>();
    const x::breaker::Config breaker{
        .name = "tcp",
        .base_interval = 20 * x::telem::MILLISECOND,
    };

    [[nodiscard]] synnax::tcp::Properties props() const {
        synnax::tcp::Properties p;
        p.host = "127.0.0.1";
        p.port = this->acceptor.local_endpoint().port();
        return p;
    }

    std::unique_ptr<common::ReadTask> read_task(bus::ReadConfig cfg) {
        return std::make_unique<common::ReadTask>(
            this->task,
            this->ctx,
            this->breaker,
            std::make_unique<bus::Source>(
                std::move(cfg),
                bus::acquirer<Client>(this->connections, "dev", this->props())
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
            this->breaker,
            std::make_unique<bus::Sink>(
                std::move(cfg),
                std::make_unique<bus::ConnectionTransmitter>(
                    bus::acquirer<Client>(this->connections, "dev", this->props())
                ),
                this->ctx,
                this->task
            ),
            nullptr,
            pipeline::mock::simple_streamer_factory({7}, commands)
        );
    }

    static void send(asio::ip::tcp::socket &peer, const std::string &data) {
        asio::write(peer, asio::buffer(data));
    }

    static std::string receive(asio::ip::tcp::socket &peer, const std::size_t n) {
        std::string out(n, '\0');
        asio::read(peer, asio::buffer(out));
        return out;
    }

    bool warned() {
        for (const auto &s: this->ctx->statuses)
            if (s.variant == synnax::status::VARIANT_WARNING) return true;
        return false;
    }
};

const ::synnax::bus::Framing NEWLINE = ::synnax::bus::DelimiterFraming{};
}

TEST_F(TCPTask, ReadsAStreamAndReconnectsAfterTheDeviceDrops) {
    const auto m = text_message("status", {delimited_field("v", 0)});
    auto t = this->read_task(read_config({m}, NEWLINE));
    t->start("start");
    {
        auto peer = this->acceptor.accept();
        send(peer, "1\n2");
        ASSERT_EVENTUALLY_EQ(values(*this->writes, 1).size(), 1);
    }
    auto peer = this->acceptor.accept();
    send(peer, "3\n");
    ASSERT_EVENTUALLY_EQ(values(*this->writes, 1).size(), 2);
    t->stop("stop", true);
    EXPECT_EQ(values(*this->writes, 1), (std::vector<double>{1, 3}));
    EXPECT_TRUE(this->warned());
}

TEST_F(TCPTask, PollsAnInstrumentWithSeveralQueries) {
    auto volt = text_message("volt", {delimited_field("v", 0)});
    volt.query = "MEAS:VOLT?";
    auto curr = text_message("curr", {delimited_field("i", 0)});
    curr.query = "MEAS:CURR?";
    auto t = this->read_task(read_config(
        {volt, curr},
        NEWLINE,
        nullptr,
        {.rate = x::telem::Rate(20), .timeout = x::telem::SECOND}
    ));
    t->start("start");
    auto peer = this->acceptor.accept();
    for (int i = 0; i < 2; i++) {
        ASSERT_EQ(receive(peer, 11), "MEAS:VOLT?\n");
        send(peer, "12.5\n");
        ASSERT_EQ(receive(peer, 11), "MEAS:CURR?\n");
        send(peer, "0.5\n");
    }
    ASSERT_EVENTUALLY_EQ(values(*this->writes, 2).size(), 2);
    t->stop("stop", true);
    EXPECT_EQ(values(*this->writes, 1), (std::vector<double>{12.5, 12.5}));
    EXPECT_EQ(values(*this->writes, 2), (std::vector<double>{0.5, 0.5}));
}

TEST_F(TCPTask, WritesCommands) {
    const auto m = text_message("set", {tagged_field("v", "VOLT ")});
    auto commands = std::make_shared<std::vector<x::telem::Frame>>();
    commands->emplace_back(7, x::telem::Series(12.5, x::telem::FLOAT64_T));
    auto t = this->write_task(write_config(m, {{7, 0}}, NEWLINE), commands);
    t->start("start");
    auto peer = this->acceptor.accept();
    EXPECT_EQ(receive(peer, 10), "VOLT 12.5\n");
    t->stop("stop", true);
}

TEST_F(TCPTask, InterleavesCommandsWithPollsOnOneConnection) {
    auto volt = text_message("volt", {delimited_field("v", 0)});
    volt.query = "MEAS:VOLT?";
    auto reader = this->read_task(read_config(
        {volt},
        NEWLINE,
        nullptr,
        {.rate = x::telem::Rate(500), .timeout = x::telem::SECOND}
    ));
    auto commands = std::make_shared<std::vector<x::telem::Frame>>();
    for (int i = 1; i <= 20; i++)
        commands->emplace_back(7, x::telem::Series(double(i), x::telem::FLOAT64_T));
    auto writer = this->write_task(
        write_config(
            text_message("set", {tagged_field("v", "VOLT ")}),
            {{7, 0}},
            NEWLINE
        ),
        commands
    );
    reader->start("start");
    writer->start("start");
    auto peer = this->acceptor.accept();
    // Plays a SCPI instrument, which drops a reply when a command arrives before the
    // reply goes out.
    asio::streambuf buf;
    std::istream lines(&buf);
    int volts = 0;
    int interrupted = 0;
    int answered = 0;
    while (volts < 20 || answered < 1) {
        asio::read_until(peer, buf, '\n');
        std::string line;
        std::getline(lines, line);
        if (line != "MEAS:VOLT?") {
            volts = std::stoi(line.substr(5));
            answered = 0;
            continue;
        }
        std::this_thread::sleep_for(std::chrono::milliseconds(2));
        if (peer.available() > 0 || buf.size() > 0) {
            interrupted++;
            continue;
        }
        send(peer, std::to_string(volts) + "\n");
        answered++;
    }
    ASSERT_EVENTUALLY_TRUE([&] {
        const auto v = values(*this->writes, 1);
        return !v.empty() && v.back() == 20;
    }());
    writer->stop("stop", true);
    reader->stop("stop", true);
    EXPECT_EQ(interrupted, 0);
    this->acceptor.non_blocking(true);
    std::error_code ec;
    this->acceptor.accept(ec);
    EXPECT_EQ(ec, asio::error::would_block);
}

TEST(Scanner, ReportsWhetherEachDeviceAcceptsAConnection) {
    asio::io_context io;
    asio::ip::tcp::acceptor open(
        io,
        asio::ip::tcp::endpoint(asio::ip::address_v4::loopback(), 0)
    );
    std::uint16_t closed_port;
    {
        asio::ip::tcp::acceptor closed(
            io,
            asio::ip::tcp::endpoint(asio::ip::address_v4::loopback(), 0)
        );
        closed_port = closed.local_endpoint().port();
    }
    const auto device = [](const std::string &key, const std::uint16_t port) {
        return synnax::device::Device{
            .key = key,
            .make = MAKE,
            .name = key,
            .properties = {{"host", "127.0.0.1"}, {"port", port}},
        };
    };
    const std::unordered_map<std::string, synnax::device::Device> tracked = {
        {"up", device("up", open.local_endpoint().port())},
        {"down", device("down", closed_port)},
    };
    Scanner scanner(synnax::task::Task{.rack = 1}, Config{});
    const auto devs = ASSERT_NIL_P(scanner.scan({.devices = &tracked}));
    ASSERT_EQ(devs.size(), 2);
    for (const auto &dev: devs)
        if (dev.key == "up") {
            EXPECT_EQ(dev.status->variant, synnax::status::VARIANT_SUCCESS);
            EXPECT_EQ(dev.status->message, "Device connected");
        } else {
            EXPECT_EQ(dev.status->variant, synnax::status::VARIANT_WARNING);
            EXPECT_EQ(dev.status->message, "Failed to reach device");
        }
}
}
