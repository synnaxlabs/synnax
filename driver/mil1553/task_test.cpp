// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <atomic>
#include <chrono>
#include <optional>
#include <string>
#include <thread>
#include <vector>

#include "gtest/gtest.h"

#include "client/cpp/mil1553/json.gen.h"
#include "x/cpp/test/test.h"

#include "driver/bus/testutil/core.h"
#include "driver/bus/testutil/testutil.h"
#include "driver/mil1553/mil1553.h"

namespace driver::mil1553 {
using namespace bus::testutil;

namespace {
/// @returns a one-word message to terminal rt on subaddress sa with an 8-bit field
/// in the high byte of the word.
synnax::library::MessageEntry create_message(
    const std::uint8_t rt,
    const std::uint8_t sa,
    const std::string &direction,
    const std::optional<x::telem::TimeSpan> period = std::nullopt
) {
    auto m = binary_message(
        "rt" + std::to_string(rt) + "_sa" + std::to_string(sa),
        {binary_field("v", 0)},
        synnax::library::Mil1553Identifier{
            .rt = rt,
            .subaddress = sa,
            .direction = direction,
            .word_count = 1,
        }
    );
    m.period = period;
    return m;
}

synnax::library::MessageEntry create_transmit(
    const std::uint8_t rt,
    const std::uint8_t sa,
    const std::optional<x::telem::TimeSpan> period = std::nullopt
) {
    return create_message(rt, sa, synnax::library::DIRECTION_TRANSMIT, period);
}

synnax::library::MessageEntry
create_receive(const std::uint8_t rt, const std::uint8_t sa) {
    return create_message(rt, sa, synnax::library::DIRECTION_RECEIVE);
}

class MIL1553Task : public ::testing::Test {
protected:
    Core core;
    Factory factory{create_backends()};

    [[nodiscard]] std::string create_device(
        const std::string &role,
        const std::vector<std::uint8_t> &terminals = {}
    ) const {
        synnax::mil1553::Properties props;
        props.role = role;
        props.terminals = terminals;
        return this->core.create_device(MAKE, props.to_json());
    }

    std::unique_ptr<task::Task>
    configure(const std::string &type, const x::json::json &cfg) {
        auto [t, ok] = this->factory.configure_task(
            this->core.ctx,
            this->core.task(type, cfg),
            "configure"
        );
        EXPECT_TRUE(ok);
        return std::move(t);
    }

    /// @returns a read task over message on dev and the channel of its field.
    std::pair<std::unique_ptr<task::Task>, synnax::channel::Key> create_reader(
        const synnax::library::MessageEntry &message,
        const std::string &dev
    ) {
        synnax::mil1553::ReadConfig cfg;
        static_cast<::synnax::bus::ReadConfig &>(
            cfg
        ) = this->core.read_config(this->core.create_library({message}), dev);
        return {
            this->configure(READ_TASK_TYPE, cfg.to_json()),
            cfg.messages[0].fields[0].channel,
        };
    }

    /// @returns a write task that drives the field of message on dev from a virtual
    /// channel, and the channel.
    std::pair<std::unique_ptr<task::Task>, synnax::channel::Key> create_writer(
        const synnax::library::MessageEntry &message,
        const std::string &dev
    ) {
        const auto lib = this->core.create_library({message});
        const auto &m = std::get<synnax::library::MessageEntry>(lib.entries[0]);
        const auto cmd = create_virtual_channel(
            *this->core.client,
            x::telem::FLOAT64_T
        );
        synnax::mil1553::WriteConfig cfg;
        cfg.device = dev;
        cfg.library = lib.key;
        cfg.messages = {{
            .message = m.key,
            .fields = {{.field = key(m.fields[0]), .channel = cmd.key}},
        }};
        return {this->configure(WRITE_TASK_TYPE, cfg.to_json()), cmd.key};
    }

    /// @brief writes value to cmd until the streamer reads a frame.
    /// @returns the first value of data in the frame.
    double send_until_read(
        const synnax::channel::Key cmd,
        synnax::framer::Streamer &streamer,
        const synnax::channel::Key data,
        const double value
    ) {
        auto w = ASSERT_NIL_P(core.client->telem.open_writer({.channels = {cmd}}));
        std::atomic done = false;
        std::thread sender([&] {
            for (int i = 0; i < 100 && !done; i++) {
                ASSERT_NIL(w.write(x::telem::Frame(cmd, x::telem::Series(value))));
                std::this_thread::sleep_for(std::chrono::milliseconds(20));
            }
        });
        const auto fr = ASSERT_NIL_P(streamer.read());
        done = true;
        sender.join();
        EXPECT_FALSE(w.close());
        return fr.at<double>(data, 0);
    }

    [[nodiscard]] std::string error() const {
        EXPECT_EQ(this->core.ctx->statuses.size(), 1);
        if (this->core.ctx->statuses.empty()) return "";
        const auto &status = this->core.ctx->statuses[0];
        EXPECT_EQ(status.variant, synnax::status::VARIANT_ERROR);
        return status.message;
    }
};
}

TEST_F(MIL1553Task, BusControllerPollsTheWordsARemoteTerminalAnswersWith) {
    const auto bc = this->create_device(synnax::mil1553::ROLE_BUS_CONTROLLER);
    const auto rt = this->create_device(synnax::mil1553::ROLE_REMOTE_TERMINAL, {5});
    auto [reader, data] = this->create_reader(
        create_transmit(5, 1, 20 * x::telem::MILLISECOND),
        bc
    );
    ASSERT_NE(reader, nullptr);
    auto [writer, cmd] = this->create_writer(create_transmit(5, 1), rt);
    ASSERT_NE(writer, nullptr);
    exec(*writer, "start");
    exec(*reader, "start");
    auto streamer = core.stream(data);
    // The terminal answers busy with no data until the write task sets its words.
    EXPECT_EQ(this->send_until_read(cmd, streamer, data, 42), 42);
    exec(*reader, "stop");
    exec(*writer, "stop");
    ASSERT_NIL(streamer.close());
}

TEST_F(MIL1553Task, ReadAndWriteTasksShareTheChannelOfTheirDevice) {
    const auto bc = this->create_device(synnax::mil1553::ROLE_BUS_CONTROLLER);
    const auto rt = this->create_device(synnax::mil1553::ROLE_REMOTE_TERMINAL, {5});
    auto [rt_reader, rt_data] = this->create_reader(create_receive(5, 2), rt);
    auto [rt_writer, rt_cmd] = this->create_writer(create_transmit(5, 1), rt);
    auto [bc_reader, bc_data] = this->create_reader(
        create_transmit(5, 1, 20 * x::telem::MILLISECOND),
        bc
    );
    auto [bc_writer, bc_cmd] = this->create_writer(create_receive(5, 2), bc);
    ASSERT_TRUE(rt_reader && rt_writer && bc_reader && bc_writer);
    for (auto *t: {&rt_reader, &rt_writer, &bc_reader, &bc_writer})
        exec(**t, "start");
    auto rt_streamer = core.stream(rt_data);
    EXPECT_EQ(this->send_until_read(bc_cmd, rt_streamer, rt_data, 7), 7);
    auto bc_streamer = core.stream(bc_data);
    EXPECT_EQ(this->send_until_read(rt_cmd, bc_streamer, bc_data, 9), 9);
    for (auto *t: {&bc_writer, &bc_reader, &rt_writer, &rt_reader})
        exec(**t, "stop");
    ASSERT_NIL(rt_streamer.close());
    ASSERT_NIL(bc_streamer.close());
    EXPECT_TRUE(
        this->core.ctx->statuses.empty() ||
        this->core.ctx->statuses.back().variant != synnax::status::VARIANT_ERROR
    );
}

TEST_F(MIL1553Task, MonitorDecodesTheTransfersOfTheBus) {
    auto [reader, data] = this->create_reader(
        create_receive(3, 2),
        this->create_device(synnax::mil1553::ROLE_MONITOR)
    );
    auto [writer, cmd] = this->create_writer(
        create_receive(3, 2),
        this->create_device(synnax::mil1553::ROLE_BUS_CONTROLLER)
    );
    ASSERT_TRUE(reader && writer);
    exec(*reader, "start");
    exec(*writer, "start");
    auto streamer = core.stream(data);
    EXPECT_EQ(this->send_until_read(cmd, streamer, data, 12), 12);
    exec(*writer, "stop");
    exec(*reader, "stop");
    ASSERT_NIL(streamer.close());
}

TEST_F(MIL1553Task, RejectsAWriteTaskOnAMonitor) {
    auto [t, _] = this->create_writer(
        create_receive(3, 2),
        this->create_device(synnax::mil1553::ROLE_MONITOR)
    );
    EXPECT_EQ(t, nullptr);
    EXPECT_NE(this->error().find("cannot be sent by a monitor"), std::string::npos);
}

TEST_F(MIL1553Task, RejectsABusControllerMessageWithoutAPeriod) {
    auto [t, _] = this->create_reader(
        create_transmit(5, 1),
        this->create_device(synnax::mil1553::ROLE_BUS_CONTROLLER)
    );
    EXPECT_EQ(t, nullptr);
    EXPECT_NE(this->error().find("needs a period"), std::string::npos);
}

TEST_F(MIL1553Task, RejectsARemoteTerminalMessageForAnotherTerminal) {
    auto [t, _] = this->create_reader(
        create_receive(6, 2),
        this->create_device(synnax::mil1553::ROLE_REMOTE_TERMINAL, {5})
    );
    EXPECT_EQ(t, nullptr);
    EXPECT_NE(
        this->error().find("is for terminal 6, which the remote terminal does not own"),
        std::string::npos
    );
}

TEST_F(MIL1553Task, RejectsADeviceWithAnInvalidRole) {
    auto [t, _] = this->create_reader(
        create_receive(5, 2),
        this->create_device(synnax::mil1553::ROLE_MONITOR, {5})
    );
    EXPECT_EQ(t, nullptr);
    EXPECT_NE(
        this->error().find("only a remote terminal channel owns terminals"),
        std::string::npos
    );
}

TEST_F(MIL1553Task, ScansTheSimulatedChannel) {
    synnax::mil1553::ScanConfig cfg;
    auto t = this->configure(SCAN_TASK_TYPE, cfg.to_json());
    ASSERT_NE(t, nullptr);
    exec(*t, "start");
    const auto key = "mil1553_" + std::to_string(core.rack.key) + "_simulated_0_0";
    const auto dev = ASSERT_EVENTUALLY_NIL_P(core.client->devices.retrieve(key));
    exec(*t, "stop");
    EXPECT_EQ(dev.make, MAKE);
    EXPECT_EQ(dev.properties.at("backend"), "simulated");
}
}
