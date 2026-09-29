// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "client/cpp/serial/json.gen.h"
#include "x/cpp/test/test.h"

#include "driver/bus/testutil/core.h"
#include "driver/bus/testutil/testutil.h"
#include "driver/serial/pty.h"
#include "driver/serial/serial.h"

namespace driver::serial {
using namespace bus::testutil;

namespace {
class SerialFactory : public PtyTest {
protected:
    Core core;
    Factory factory;
};
}

TEST_F(SerialFactory, ConfiguresAReadTaskThatStreamsDecodedValues) {
    const auto lib = core.create_library(
        {text_message("env", {tagged_field("t", "T=")})}
    );
    const auto dev = core.create_device(MAKE, this->props().to_json());
    synnax::serial::ReadConfig cfg;
    static_cast<::synnax::bus::ReadConfig &>(cfg) = core.read_config(lib, dev);
    const auto data = cfg.messages[0].fields[0].channel;
    const auto task = core.task(READ_TASK_TYPE, cfg.to_json());
    auto [t, ok] = this->factory.configure_task(core.ctx, task, "");
    ASSERT_TRUE(ok);
    ASSERT_NE(t, nullptr);
    auto streamer = core.stream(data);
    exec(*t, "start");
    this->peer_write("T=21.5\n");
    const auto fr = ASSERT_NIL_P(streamer.read());
    EXPECT_EQ(fr.at<double>(data, 0), 21.5);
    exec(*t, "stop");
    ASSERT_NIL(streamer.close());
}

TEST_F(SerialFactory, RejectsAMessageThatIsNotInTheLibrary) {
    const auto lib = core.create_library(
        {text_message("env", {tagged_field("t", "T=")})}
    );
    const auto dev = core.create_device(MAKE, this->props().to_json());
    synnax::serial::ReadConfig cfg;
    static_cast<::synnax::bus::ReadConfig &>(cfg) = core.read_config(lib, dev);
    cfg.messages[0].message = x::uuid::create();
    auto [t, ok] = this->factory.configure_task(
        core.ctx,
        core.task(READ_TASK_TYPE, cfg.to_json()),
        "configure"
    );
    ASSERT_TRUE(ok);
    EXPECT_EQ(t, nullptr);
    ASSERT_EQ(core.ctx->statuses.size(), 1);
    const auto &status = core.ctx->statuses[0];
    EXPECT_EQ(status.variant, synnax::status::VARIANT_ERROR);
    EXPECT_EQ(status.details.cmd, "configure");
    EXPECT_NE(status.message.find("is not in library"), std::string::npos);
}

TEST_F(SerialFactory, ConfiguresAWriteTask) {
    const auto lib = core.create_library(
        {text_message("set", {tagged_field("v", "VOLT ")})}
    );
    const auto dev = core.create_device(MAKE, this->props().to_json());
    const auto &m = std::get<synnax::library::MessageEntry>(lib.entries[0]);
    const auto cmd = create_virtual_channel(*core.client, x::telem::FLOAT64_T);
    synnax::serial::WriteConfig cfg;
    cfg.device = dev;
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
    std::string got;
    for (int i = 0; i < 50 && got.empty(); i++) {
        ASSERT_NIL(writer.write(x::telem::Frame(cmd.key, x::telem::Series(4.0))));
        got = this->peer_read(7);
    }
    EXPECT_EQ(got.substr(0, 7), "VOLT 4\n");
    ASSERT_NIL(writer.close());
    exec(*t, "stop");
}

TEST_F(SerialFactory, ConfiguresTheScanTask) {
    const auto task = core.task(SCAN_TASK_TYPE, x::json::json::object());
    auto [t, ok] = this->factory.configure_task(core.ctx, task, "");
    ASSERT_TRUE(ok);
    ASSERT_NE(t, nullptr);
    t->stop(false);
}
}
