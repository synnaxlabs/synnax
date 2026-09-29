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
#include <string>
#include <thread>
#include <vector>

#include "gtest/gtest.h"

#include "client/cpp/arinc429/json.gen.h"
#include "x/cpp/test/test.h"

#include "driver/arinc429/arinc429.h"
#include "driver/bus/testutil/core.h"
#include "driver/bus/testutil/testutil.h"

namespace driver::arinc429 {
using namespace bus::testutil;

namespace {
/// @returns a message on label 0310 with SDI sdi and one 8-bit field in bits 11 to
/// 18.
synnax::library::MessageEntry create_message(const std::uint8_t sdi) {
    return binary_message(
        "altitude_" + std::to_string(sdi),
        {binary_field("v", 10)},
        synnax::library::Arinc429Identifier{
            .label = 0310,
            .sdi = sdi,
            .sdi_matched = true,
        }
    );
}

class ARINC429Task : public ::testing::Test {
protected:
    Core core;
    Factory factory{create_backends()};

    [[nodiscard]] std::string
    create_device(const std::string &speed = synnax::arinc429::SPEED_HIGH) const {
        synnax::arinc429::Properties props;
        props.speed = speed;
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

    /// @returns a write task that drives the v field of each message in lib from its
    /// own virtual channel, in order.
    std::pair<std::unique_ptr<task::Task>, std::vector<synnax::channel::Key>>
    create_writer(const synnax::library::Library &lib, const std::string &dev) {
        synnax::arinc429::WriteConfig cfg;
        cfg.device = dev;
        cfg.library = lib.key;
        std::vector<synnax::channel::Key> keys;
        for (const auto &e: lib.entries) {
            const auto &m = std::get<synnax::library::MessageEntry>(e);
            const auto cmd = create_virtual_channel(
                *this->core.client,
                x::telem::FLOAT64_T
            );
            keys.push_back(cmd.key);
            cfg.messages.push_back({
                .message = m.key,
                .fields = {{.field = key(m.fields[0]), .channel = cmd.key}},
            });
        }
        return {this->configure(WRITE_TASK_TYPE, cfg.to_json()), keys};
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

TEST_F(ARINC429Task, ReadsTheLabelsAWriteTaskSendsOnTheSameChannel) {
    const auto dev = this->create_device();
    const auto read_lib = core.create_library({create_message(1)});
    synnax::arinc429::ReadConfig read_cfg;
    static_cast<::synnax::bus::ReadConfig &>(
        read_cfg
    ) = core.read_config(read_lib, dev);
    const auto data = read_cfg.messages[0].fields[0].channel;
    auto reader = this->configure(READ_TASK_TYPE, read_cfg.to_json());
    ASSERT_NE(reader, nullptr);
    auto [writer, cmds] = this->create_writer(
        core.create_library({create_message(1), create_message(2)}),
        dev
    );
    ASSERT_NE(writer, nullptr);
    auto streamer = core.stream(data);
    exec(*reader, "start");
    exec(*writer, "start");
    auto w = ASSERT_NIL_P(core.client->telem.open_writer({.channels = cmds}));
    std::atomic done = false;
    std::thread sender([&] {
        for (int i = 0; i < 100 && !done; i++) {
            x::telem::Frame fr;
            fr.emplace(cmds[1], x::telem::Series(7.0));
            fr.emplace(cmds[0], x::telem::Series(42.0 + (i % 2)));
            ASSERT_NIL(w.write(fr));
            std::this_thread::sleep_for(std::chrono::milliseconds(20));
        }
    });
    const auto fr = ASSERT_NIL_P(streamer.read());
    done = true;
    sender.join();
    const auto v = fr.at<double>(data, 0);
    EXPECT_TRUE(v == 42 || v == 43) << v;
    ASSERT_NIL(w.close());
    exec(*writer, "stop");
    exec(*reader, "stop");
    ASSERT_NIL(streamer.close());
}

TEST_F(ARINC429Task, RejectsADeviceWithAnUnknownSpeed) {
    const auto lib = core.create_library({create_message(1)});
    synnax::arinc429::ReadConfig cfg;
    static_cast<::synnax::bus::ReadConfig &>(
        cfg
    ) = core.read_config(lib, this->create_device("medium"));
    EXPECT_EQ(this->configure(READ_TASK_TYPE, cfg.to_json()), nullptr);
    EXPECT_NE(this->error().find("unknown ARINC 429 speed medium"), std::string::npos);
}

TEST_F(ARINC429Task, RejectsAMessageWithoutALabel) {
    const auto lib = core.create_library(
        {binary_message("m", {binary_field("v", 10)})}
    );
    synnax::arinc429::ReadConfig cfg;
    static_cast<::synnax::bus::ReadConfig &>(
        cfg
    ) = core.read_config(lib, this->create_device());
    EXPECT_EQ(this->configure(READ_TASK_TYPE, cfg.to_json()), nullptr);
    EXPECT_NE(
        this->error().find("ARINC 429 messages need an identifier"),
        std::string::npos
    );
}

TEST_F(ARINC429Task, RejectsAWriteMessageWithAFieldOutsideTheData) {
    auto m = create_message(1);
    m.fields = {binary_field("v", 4)};
    auto [t, cmds] = this->create_writer(
        core.create_library({m}),
        this->create_device()
    );
    EXPECT_EQ(t, nullptr);
    EXPECT_NE(this->error().find("must lie in bits"), std::string::npos);
}

TEST_F(ARINC429Task, ScansTheSimulatedChannel) {
    synnax::arinc429::ScanConfig cfg;
    auto t = this->configure(SCAN_TASK_TYPE, cfg.to_json());
    ASSERT_NE(t, nullptr);
    exec(*t, "start");
    const auto key = "arinc429_" + std::to_string(core.rack.key) + "_simulated_0_0";
    const auto dev = ASSERT_EVENTUALLY_NIL_P(core.client->devices.retrieve(key));
    exec(*t, "stop");
    EXPECT_EQ(dev.make, MAKE);
    EXPECT_EQ(dev.properties.at("backend"), "simulated");
}
}
