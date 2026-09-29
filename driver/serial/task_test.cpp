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

#include "gtest/gtest.h"

#include "client/cpp/serial/json.gen.h"
#include "x/cpp/test/test.h"

#include "driver/bus/read.h"
#include "driver/bus/task.h"
#include "driver/bus/testutil/testutil.h"
#include "driver/bus/write.h"
#include "driver/pipeline/mock/pipeline.h"
#include "driver/serial/port.h"
#include "driver/serial/pty.h"
#include "driver/serial/scan_task.h"

namespace driver::serial {
using namespace bus::testutil;

namespace {
/// @brief runs bus tasks against a port whose device the test plays on a pty.
class SerialTask : public PtyTest {
protected:
    std::shared_ptr<bus::Connections>
        connections = std::make_shared<bus::Connections>();
    std::shared_ptr<task::MockContext> ctx = std::make_shared<task::MockContext>(
        nullptr
    );
    synnax::task::Task task{.key = x::uuid::create(), .name = "serial"};
    std::shared_ptr<std::vector<x::telem::Frame>>
        writes = std::make_shared<std::vector<x::telem::Frame>>();

    std::unique_ptr<common::ReadTask> read_task(bus::ReadConfig cfg) {
        return std::make_unique<common::ReadTask>(
            this->task,
            this->ctx,
            x::breaker::default_config(this->task.name),
            std::make_unique<bus::Source>(
                std::move(cfg),
                bus::acquirer<Port>(this->connections, "dev", this->props())
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
                    bus::acquirer<Port>(this->connections, "dev", this->props())
                )
            ),
            nullptr,
            pipeline::mock::simple_streamer_factory({7}, commands)
        );
    }
};
}

TEST_F(SerialTask, ReadsTaggedTextLinesIntoChannels) {
    const auto m = text_message(
        "env",
        {tagged_field("t", "T="), tagged_field("p", "P=")}
    );
    auto t = this->read_task(read_config({m}, ::synnax::bus::DelimiterFraming{}));
    t->start("start");
    this->peer_write("T=23.5,P=101.3\nT=24");
    this->peer_write(".5,P=99\n");
    ASSERT_EVENTUALLY_EQ(values(*this->writes, 1).size(), 2);
    t->stop("stop", true);
    EXPECT_EQ(values(*this->writes, 1), (std::vector<double>{23.5, 24.5}));
    EXPECT_EQ(values(*this->writes, 2), (std::vector<double>{101.3, 99}));
    EXPECT_EQ(values(*this->writes, 100).size(), 2);
}

TEST_F(SerialTask, ReadsSyncFramesWithAChecksum) {
    const ::synnax::bus::Framing framing = ::synnax::bus::SyncFraming{
        .sync = "AA55",
        .length_offset = 2,
        .length_adjustment = 2,
        .checksum = ::synnax::bus::CHECKSUM_CRC_16_MODBUS,
    };
    const auto m = binary_message("status", {binary_field("v", 24, 16)});
    auto t = this->read_task(read_config({m}, framing));
    auto [framer, err] = codec::framing::create(framing);
    ASSERT_NIL(err);
    std::vector<std::uint8_t> wire;
    ASSERT_NIL(
        framer->encode(std::vector<std::uint8_t>{0xAA, 0x55, 0, 0x34, 0x12}, wire)
    );
    t->start("start");
    this->peer_write("junk");
    this->peer_write(text(wire));
    ASSERT_EVENTUALLY_EQ(values(*this->writes, 1).size(), 1);
    t->stop("stop", true);
    EXPECT_EQ(values(*this->writes, 1)[0], 0x1234);
}

TEST_F(SerialTask, PollsAnInstrumentAndDecodesItsReplies) {
    auto volt = text_message("volt", {delimited_field("v", 0)});
    volt.query = "MEAS:VOLT?";
    auto t = this->read_task(read_config(
        {volt},
        ::synnax::bus::DelimiterFraming{},
        nullptr,
        {.rate = x::telem::Rate(20), .timeout = x::telem::SECOND}
    ));
    t->start("start");
    for (int i = 0; i < 3; i++) {
        ASSERT_EQ(this->peer_read(11), "MEAS:VOLT?\n");
        this->peer_write(std::to_string(10 + i) + "\n");
    }
    ASSERT_EVENTUALLY_EQ(values(*this->writes, 1).size(), 3);
    t->stop("stop", true);
    EXPECT_EQ(values(*this->writes, 1), (std::vector<double>{10, 11, 12}));
}

TEST_F(SerialTask, WritesAFramedCommand) {
    const auto m = text_message("set", {tagged_field("v", "VOLT ")});
    auto commands = std::make_shared<std::vector<x::telem::Frame>>();
    commands->emplace_back(7, x::telem::Series(12.5, x::telem::FLOAT64_T));
    auto t = this->write_task(
        write_config(m, {{7, 0}}, ::synnax::bus::DelimiterFraming{}),
        commands
    );
    t->start("start");
    EXPECT_EQ(this->peer_read(10), "VOLT 12.5\n");
    t->stop("stop", true);
}

TEST_F(SerialTask, ReadsAndWritesOnePortAtOnce) {
    const ::synnax::bus::Framing newline = ::synnax::bus::DelimiterFraming{};
    auto reader = this->read_task(
        read_config({text_message("env", {tagged_field("t", "T=")})}, newline)
    );
    auto commands = std::make_shared<std::vector<x::telem::Frame>>();
    commands->emplace_back(7, x::telem::Series(5.0, x::telem::FLOAT64_T));
    auto writer = this->write_task(
        write_config(
            text_message("set", {tagged_field("v", "VOLT ")}),
            {{7, 0}},
            newline
        ),
        commands
    );
    reader->start("start");
    writer->start("start");
    this->peer_write("T=0\n");
    EXPECT_EQ(this->peer_read(7), "VOLT 5\n");
    for (int i = 1; i < 5; i++)
        this->peer_write("T=" + std::to_string(i) + "\n");
    ASSERT_EVENTUALLY_EQ(values(*this->writes, 1).size(), 5);
    writer->stop("stop", true);
    reader->stop("stop", true);
    EXPECT_EQ(values(*this->writes, 1), (std::vector<double>{0, 1, 2, 3, 4}));
}

TEST(Scanner, ReportsEachPortAsADevice) {
    synnax::task::Task task{.rack = 7, .name = "scan"};
    Scanner scanner(task, [] {
        return std::pair{
            std::vector<Info>{
                {.path = "/dev/ttyUSB0", .name = "ttyUSB0"},
                {.path = "/dev/ttyUSB1", .name = "ttyUSB1"},
            },
            x::errors::NIL,
        };
    });
    const std::unordered_map<std::string, synnax::device::Device> tracked = {
        {"mine",
         synnax::device::Device{
             .key = "mine",
             .rack = 7,
             .make = MAKE,
             .name = "Power supply",
             .properties = {{"port", "/dev/ttyUSB1"}, {"baud_rate", 115200}},
         }},
    };
    const auto devs = ASSERT_NIL_P(scanner.scan({.devices = &tracked}));
    ASSERT_EQ(devs.size(), 2);
    EXPECT_EQ(devs[0].key, "mine");
    EXPECT_EQ(devs[0].properties.at("baud_rate"), 115200);
    EXPECT_EQ(devs[0].status->variant, synnax::status::VARIANT_SUCCESS);
    EXPECT_EQ(devs[1].key, "serial_7__dev_ttyUSB0");
    EXPECT_EQ(devs[1].make, "Serial");
    EXPECT_EQ(devs[1].name, "ttyUSB0");
    EXPECT_EQ(devs[1].properties.at("port"), "/dev/ttyUSB0");
    EXPECT_EQ(devs[1].status->message, "Port available");
}

TEST(Scanner, ReturnsTheListingError) {
    Scanner scanner(synnax::task::Task{}, [] {
        return std::pair{std::vector<Info>{}, x::errors::Error(SCAN_ERROR, "nope")};
    });
    ASSERT_OCCURRED_AS_P(scanner.scan({}), SCAN_ERROR);
}
}
