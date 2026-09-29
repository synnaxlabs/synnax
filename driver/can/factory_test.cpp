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
#include <memory>
#include <string>
#include <thread>
#include <vector>

#include "gtest/gtest.h"

#include "client/cpp/can/json.gen.h"
#include "x/cpp/test/test.h"

#include "driver/bus/testutil/core.h"
#include "driver/bus/testutil/testutil.h"
#include "driver/can/factory.h"
#include "driver/can/loopback/loopback.h"

namespace driver::can {
using namespace bus::testutil;

namespace {
class CANFactory : public ::testing::Test {
protected:
    std::shared_ptr<loopback::Backend> backend = std::make_shared<loopback::Backend>(
        std::vector<std::string>{"can0"}
    );
    Core core;
    Factory factory{Backends{{loopback::BACKEND, backend}}};
    std::unique_ptr<Bus> peer;

    void SetUp() override { this->peer = ASSERT_NIL_P(this->backend->open(props())); }

    [[nodiscard]] static synnax::can::Properties props() {
        synnax::can::Properties p;
        p.backend = loopback::BACKEND;
        p.channel = "can0";
        return p;
    }

    /// @returns the configured task, or nullptr with the error in core.ctx->statuses.
    std::unique_ptr<task::Task>
    configure(const std::string &type, const x::json::json &cfg) {
        auto [t, ok] = this->factory
                           .configure_task(core.ctx, core.task(type, cfg), "configure");
        EXPECT_TRUE(ok);
        return std::move(t);
    }

    /// @returns the message of the last status the factory reported.
    [[nodiscard]] std::string error() const {
        EXPECT_FALSE(core.ctx->statuses.empty());
        if (core.ctx->statuses.empty()) return "";
        const auto &status = core.ctx->statuses.back();
        EXPECT_EQ(status.variant, synnax::status::VARIANT_ERROR);
        return status.message;
    }

    /// @returns a write config that drives the first field of the library's message
    /// on device from a new virtual channel.
    synnax::can::WriteConfig
    write_config(const synnax::library::Library &lib, const std::string &device) {
        const auto &m = std::get<synnax::library::MessageEntry>(lib.entries[0]);
        const auto cmd = create_virtual_channel(*core.client, x::telem::FLOAT64_T);
        synnax::can::WriteConfig cfg;
        cfg.device = device;
        cfg.library = lib.key;
        cfg.messages = {{
            .message = m.key,
            .fields = {{.field = field_key(m, 0), .channel = cmd.key}},
        }};
        return cfg;
    }
};

synnax::library::MessageEntry
can_message(const std::uint32_t id, const std::uint16_t length = 1) {
    auto m = binary_message(
        "m",
        {binary_field("v", 0)},
        synnax::library::CanIdentifier{.id = id}
    );
    binary(m).length = length;
    return m;
}
}

TEST_F(CANFactory, ConfiguresAReadTaskThatStreamsDecodedValues) {
    const auto lib = core.create_library({can_message(0x10)});
    synnax::can::ReadConfig cfg;
    static_cast<::synnax::bus::ReadConfig &>(
        cfg
    ) = core.read_config(lib, core.create_device(MAKE, props().to_json()));
    const auto data = cfg.messages[0].fields[0].channel;
    auto t = this->configure(READ_TASK_TYPE, cfg.to_json());
    ASSERT_NE(t, nullptr);
    auto streamer = core.stream(data);
    exec(*t, "start");
    Frame frame{.id = 0x10, .length = 1};
    frame.data[0] = 42;
    for (int i = 0; i < 20; i++)
        ASSERT_NIL(this->peer->send(frame));
    const auto fr = ASSERT_NIL_P(streamer.read());
    EXPECT_EQ(fr.at<double>(data, 0), 42);
    exec(*t, "stop");
    ASSERT_NIL(streamer.close());
}

TEST_F(CANFactory, ConfiguresAWriteTask) {
    const auto lib = core.create_library({can_message(0x20)});
    const auto cfg = this->write_config(
        lib,
        core.create_device(MAKE, props().to_json())
    );
    const auto cmd = cfg.messages[0].fields[0].channel;
    auto t = this->configure(WRITE_TASK_TYPE, cfg.to_json());
    ASSERT_NE(t, nullptr);
    exec(*t, "start");
    auto writer = ASSERT_NIL_P(core.client->telem.open_writer({.channels = {cmd}}));
    Frame frame;
    bool got = false;
    // The write task acknowledges its start before its streamer opens.
    for (int i = 0; i < 50 && !got; i++) {
        ASSERT_NIL(writer.write(x::telem::Frame(cmd, x::telem::Series(9.0))));
        got = ASSERT_NIL_P(this->peer->receive(frame, 20 * x::telem::MILLISECOND));
    }
    ASSERT_TRUE(got);
    EXPECT_EQ(frame.id, 0x20);
    EXPECT_EQ(frame.length, 1);
    EXPECT_EQ(frame.data[0], 9);
    ASSERT_NIL(writer.close());
    exec(*t, "stop");
}

TEST_F(CANFactory, RejectsAWriteTaskOnAListenOnlyDevice) {
    const auto lib = core.create_library({can_message(0x20)});
    auto p = props();
    p.listen_only = true;
    const auto cfg = this->write_config(lib, core.create_device(MAKE, p.to_json()));
    EXPECT_EQ(this->configure(WRITE_TASK_TYPE, cfg.to_json()), nullptr);
    EXPECT_NE(this->error().find("listen only"), std::string::npos);
}

TEST_F(CANFactory, RejectsAnFDMessageOnAClassicBus) {
    auto m = can_message(0x20, 12);
    std::get<synnax::library::CanIdentifier>(*binary(m).identifier).fd = true;
    const auto lib = core.create_library({m});
    const auto cfg = this->write_config(
        lib,
        core.create_device(MAKE, props().to_json())
    );
    EXPECT_EQ(this->configure(WRITE_TASK_TYPE, cfg.to_json()), nullptr);
    EXPECT_NE(
        this->error().find("a CAN FD frame cannot be sent on a classic CAN bus"),
        std::string::npos
    );
}

TEST_F(CANFactory, RejectsALengthWithNoDataLengthCode) {
    auto p = props();
    p.fd = true;
    auto m = can_message(0x20, 9);
    std::get<synnax::library::CanIdentifier>(*binary(m).identifier).fd = true;
    const auto lib = core.create_library({m});
    const auto cfg = this->write_config(lib, core.create_device(MAKE, p.to_json()));
    EXPECT_EQ(this->configure(WRITE_TASK_TYPE, cfg.to_json()), nullptr);
    EXPECT_NE(
        this->error().find("length 9 is not a valid CAN FD length"),
        std::string::npos
    );
}

TEST_F(CANFactory, RejectsAMessageWithoutACANIdentifier) {
    const auto lib = core.create_library({binary_message("m", {binary_field("v", 0)})});
    const auto cfg = this->write_config(
        lib,
        core.create_device(MAKE, props().to_json())
    );
    EXPECT_EQ(this->configure(WRITE_TASK_TYPE, cfg.to_json()), nullptr);
    EXPECT_NE(
        this->error().find("needs a CAN identifier to send with"),
        std::string::npos
    );
}

TEST_F(CANFactory, ConfiguresTheScanTask) {
    auto t = this->configure(SCAN_TASK_TYPE, x::json::json::object());
    ASSERT_NE(t, nullptr);
    t->stop(false);
}

TEST_F(CANFactory, CreatesTheScanTaskOnARackWithNone) {
    auto tasks = this->factory.configure_initial_tasks(core.ctx, core.rack);
    ASSERT_EQ(tasks.size(), 1);
    EXPECT_EQ(tasks[0].first.name, "CAN Scanner");
    EXPECT_EQ(tasks[0].first.type, SCAN_TASK_TYPE);
    tasks[0].second->stop(false);
}
}
