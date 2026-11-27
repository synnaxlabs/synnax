// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <map>
#include <memory>
#include <optional>
#include <string>
#include <utility>
#include <vector>

#include "gtest/gtest.h"

#include "client/cpp/channel/types.gen.h"
#include "client/cpp/icd/types.gen.h"
#include "client/cpp/serial/types.gen.h"
#include "x/cpp/breaker/breaker.h"
#include "x/cpp/defer/defer.h"
#include "x/cpp/telem/frame.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/testutil/testutil.h"
#include "x/cpp/uuid/uuid.h"

#include "driver/common/common.h"
#include "driver/common/read_task.h"
#include "driver/common/write_task.h"
#include "driver/icd/receiver.h"
#include "driver/icd/sender.h"
#include "driver/pipeline/testutil/pipeline.h"
#include "driver/serial/pty.h"
#include "driver/serial/read_task.h"
#include "driver/serial/serial.h"
#include "driver/serial/write_task.h"
#include "driver/task/task.h"

namespace driver::serial {
namespace {
/// @brief Command is an ICD of lines such as "4.5" whose only item takes the commands
/// of float64 channel 2 up to 100, or 7 before the first command.
struct Command {
    synnax::icd::ICD ic;
    synnax::icd::WriteTarget target;
    std::map<synnax::channel::Key, synnax::channel::Channel> channels;

    explicit Command(std::optional<x::telem::TimeSpan> period = std::nullopt) {
        synnax::icd::DelimitedTextField v;
        v.key = x::uuid::create();
        v.name = "v";
        v.default_ = 7.0;
        v.max = 100.0;
        synnax::icd::Message m;
        m.key = x::uuid::create();
        m.name = "setpoint";
        m.payload = synnax::icd::TextPayload{.fields = {v}};
        this->ic.key = x::uuid::create();
        this->ic.name = "valve";
        this->ic.framing = synnax::icd::DelimitedFraming{};
        this->ic.messages = {m};
        synnax::icd::ChannelWriteBinding b;
        b.field = v.key;
        b.channel = 2;
        this->target = {
            .icd = this->ic.key,
            .message = {.message = m.key, .bindings = {b}},
            .period = period,
        };
        synnax::channel::Channel ch;
        ch.key = 2;
        ch.data_type = x::telem::FLOAT64_T;
        ch.is_virtual = true;
        this->channels = {{2, ch}};
    }
};

/// @brief SerialWriteTask runs write tasks on a pseudo-terminal that plays the device.
class SerialWriteTask : public PtyTest {
protected:
    x::breaker::Breaker cancel;
    std::shared_ptr<task::MockContext> ctx = std::make_shared<task::MockContext>(
        nullptr
    );
    std::shared_ptr<Ports> ports = std::make_shared<Ports>();
    std::unique_ptr<common::WriteTask> task;

    /// @brief creates the task of a target that sends Command on the port of props,
    /// and gets commands from commands.
    void create(
        const synnax::serial::Properties &props,
        const Command &cmd,
        std::vector<x::telem::Frame> commands = {}
    ) {
        auto sender = ASSERT_NIL_P(
            icd::Sender::create({cmd.target}, {cmd.ic}, cmd.channels, false)
        );
        synnax::task::Task t;
        t.key = x::uuid::create();
        t.name = "serial write";
        this->task = std::make_unique<common::WriteTask>(
            t,
            this->ctx,
            common::unlimited_retry_config(t.name),
            std::make_unique<Sink>(
                this->ports,
                Device{.key = "device", .name = "device", .props = props},
                std::move(sender),
                std::vector<synnax::channel::Key>{2}
            ),
            std::make_shared<pipeline::testutil::WriterFactory>(),
            pipeline::testutil::simple_streamer_factory(
                {2},
                std::make_shared<std::vector<x::telem::Frame>>(std::move(commands))
            )
        );
    }

    void TearDown() override {
        if (this->task != nullptr) this->task->stop("", false);
        PtyTest::TearDown();
    }

    /// @returns the last status of the task, or an empty status before the first.
    [[nodiscard]] synnax::task::Status last_status() const {
        const auto statuses = this->ctx->snapshot();
        return statuses.empty() ? synnax::task::Status{} : statuses.back();
    }

    /// @brief reads from the port until what it read ends with suffix, or two seconds
    /// pass.
    [[nodiscard]] std::string read_until(const std::string &suffix) const {
        std::string got;
        const x::telem::Stopwatch sw;
        while (!got.ends_with(suffix) && sw.elapsed() < 2 * x::telem::SECOND)
            got += this->peer_read(suffix.size());
        return got;
    }
};
}

/// @brief it should send a message for a command on a target with no period.
TEST_F(SerialWriteTask, SendsCommandedMessage) {
    std::vector<x::telem::Frame> commands;
    commands.emplace_back(2, x::telem::Series(4.5));
    this->create(this->props(), Command(), std::move(commands));
    ASSERT_TRUE(this->task->start("start", this->cancel));
    ASSERT_EQ(this->peer_read(4), "4.5\n");
}

/// @brief it should skip a command that its field refuses with a warning, and send the
/// next good command.
TEST_F(SerialWriteTask, WarnsOnRefusedCommand) {
    std::vector<x::telem::Frame> commands;
    commands.emplace_back(2, x::telem::Series(1000.0));
    commands.emplace_back(2, x::telem::Series(4.5));
    this->create(this->props(), Command(), std::move(commands));
    ASSERT_TRUE(this->task->start("start", this->cancel));
    ASSERT_EQ(this->peer_read(4), "4.5\n");
    const auto statuses = this->ctx->snapshot();
    const auto refused = std::ranges::find_if(statuses, [](const auto &status) {
        return status.message == "Skipped invalid command";
    });
    ASSERT_NE(refused, statuses.end());
    ASSERT_EQ(refused->variant, synnax::status::VARIANT_WARNING);
    ASSERT_EQ(
        refused->description,
        "message setpoint: field v: 1000 is above the max 100"
    );
}

/// @brief it should send the message of a target on its period, with the default of a
/// field before its first command.
TEST_F(SerialWriteTask, SendsMessageOnPeriod) {
    this->create(this->props(), Command(100 * x::telem::MILLISECOND));
    ASSERT_TRUE(this->task->start("start", this->cancel));
    ASSERT_EQ(this->peer_read(2), "7\n");
    const x::telem::Stopwatch sw;
    ASSERT_EQ(this->peer_read(4), "7\n7\n");
    ASSERT_GE(sw.elapsed(), 150 * x::telem::MILLISECOND);
}

/// @brief it should send the last command of a target with a period on each period.
TEST_F(SerialWriteTask, SendsLastCommandOnPeriod) {
    std::vector<x::telem::Frame> commands;
    commands.emplace_back(2, x::telem::Series(3.0));
    this->create(
        this->props(),
        Command(50 * x::telem::MILLISECOND),
        std::move(commands)
    );
    ASSERT_TRUE(this->task->start("start", this->cancel));
    const auto got = this->read_until("3\n3\n");
    ASSERT_TRUE(got.ends_with("3\n3\n")) << got;
}

/// @brief it should share the port of a device with a read task on the device.
TEST_F(SerialWriteTask, SharesPortWithReadTask) {
    std::vector<x::telem::Frame> commands;
    commands.emplace_back(2, x::telem::Series(4.5));
    this->create(this->props(), Command(), std::move(commands));
    synnax::icd::DelimitedTextField v;
    v.key = x::uuid::create();
    v.name = "v";
    synnax::icd::Message m;
    m.key = x::uuid::create();
    m.name = "reading";
    m.payload = synnax::icd::TextPayload{.fields = {v}};
    synnax::icd::ICD ic;
    ic.key = x::uuid::create();
    ic.name = "sensor";
    ic.framing = synnax::icd::DelimitedFraming{};
    ic.messages = {m};
    synnax::channel::Channel ch;
    ch.key = 3;
    ch.data_type = x::telem::FLOAT64_T;
    ch.is_virtual = true;
    auto receiver = ASSERT_NIL_P(
        icd::Receiver::create(
            {{.icd = ic.key,
              .messages =
                  {{.message = m.key, .bindings = {{.field = v.key, .channel = 3}}}}}},
            {ic},
            {{3, ch}},
            false
        )
    );
    synnax::task::Task t;
    t.key = x::uuid::create();
    t.name = "serial read";
    const auto writes = std::make_shared<pipeline::testutil::WriterFactory>();
    common::ReadTask reader(
        t,
        this->ctx,
        common::unlimited_retry_config(t.name),
        std::make_unique<Source>(
            this->ports,
            Device{.key = "device", .name = "device", .props = this->props()},
            ASSERT_NIL_P(this->ports->claim_read("device", t)),
            std::move(receiver),
            std::vector{ch},
            true
        ),
        writes
    );
    ASSERT_TRUE(reader.start("start", this->cancel));
    x::defer::defer stop_reader([&reader] { reader.stop("stop", true); });
    ASSERT_TRUE(this->task->start("start", this->cancel));
    ASSERT_EQ(this->peer_read(4), "4.5\n");
    this->peer_write("1.5\n");
    ASSERT_EVENTUALLY_GE(writes->writes->size(), 1);
    ASSERT_EQ(writes->writes->at(0).at<double>(3, 0), 1.5);
}

/// @brief it should start a target with a period on a missing port, and warn while it
/// tries to open the port.
TEST_F(SerialWriteTask, WarnsWhilePortIsMissing) {
    auto props = this->props();
    props.port = "/dev/synnax-missing-port";
    this->create(props, Command(100 * x::telem::MILLISECOND));
    ASSERT_TRUE(this->task->start("start", this->cancel));
    ASSERT_EVENTUALLY_EQ(this->last_status().message, "Failed to send message");
    const auto status = this->last_status();
    ASSERT_EQ(status.variant, synnax::status::VARIANT_WARNING);
    ASSERT_TRUE(status.details.running);
    ASSERT_TRUE(
        status.description.starts_with("failed to open serial port " + props.port)
    ) << status.description;
}

/// @brief it should fail to configure a task with no device and no target, and name
/// both.
TEST(SerialWriteConfigure, NamesMissingDeviceAndTargets) {
    x::breaker::Breaker cancel;
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    Factory factory;
    cancel.start();
    const auto [tsk, handled] = factory.configure_task(
        ctx,
        synnax::task::Task{.name = "t", .type = WRITE_TASK_TYPE},
        "start-cmd",
        cancel
    );
    cancel.stop();
    ASSERT_TRUE(handled);
    ASSERT_EQ(tsk, nullptr);
    const auto statuses = ctx->snapshot();
    ASSERT_EQ(statuses.size(), 1);
    ASSERT_EQ(statuses[0].variant, synnax::status::VARIANT_ERROR);
    ASSERT_EQ(statuses[0].message, "Failed to configure task");
    ASSERT_EQ(
        statuses[0].description,
        "device: this field is required\ntargets: at least one target is required"
    );
}
}
