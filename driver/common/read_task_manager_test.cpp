// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <future>
#include <thread>

#include "gtest/gtest.h"

#include "client/cpp/testutil/testutil.h"
#include "x/cpp/test/test.h"

#include "driver/common/read_task.h"
#include "driver/errors/errors.h"
#include "driver/task/task.h"

namespace driver::common {
namespace {
/// @brief a source whose every read fails, as a read of a node that does not exist.
class FailingSource final : public Source {
    synnax::framer::WriterConfig writer_config() const override { return {}; }

    std::vector<synnax::channel::Channel> channels() const override { return {}; }

    ReadResult read(x::breaker::Breaker &, x::telem::Frame &) override {
        return {.error = x::errors::Error(errors::CRITICAL_HARDWARE_ERROR, "bad node")};
    }
};

class FailingReadFactory final : public task::Factory {
public:
    std::pair<std::unique_ptr<task::Task>, bool> configure_task(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::task::Task &task,
        const std::string &
    ) override {
        if (task.type != SYNTHETIC_TASK_TYPE) return {nullptr, false};
        return {
            std::make_unique<ReadTask>(
                task,
                ctx,
                x::breaker::default_config(task.name),
                std::make_unique<FailingSource>()
            ),
            true
        };
    }
};
}

/// @brief it should report the error of a read task whose first read fails.
TEST(ReadTaskManagerTest, FailedFirstReadReportsAnError) {
    const auto client = std::make_shared<synnax::Synnax>(new_test_client());
    const auto rack = ASSERT_NIL_P(client->racks.create("read_task_rack"));
    const auto ch = ASSERT_NIL_P(
        client->channels.retrieve(synnax::status::STATUS_SET_CHANNEL_NAME)
    );
    auto streamer = ASSERT_NIL_P(client->telem.open_streamer({.channels = {ch.key}}));
    task::Manager manager(rack, client, std::make_unique<FailingReadFactory>());
    std::promise<void> started;
    std::thread run([&] { ASSERT_NIL(manager.run([&] { started.set_value(); })); });
    ASSERT_EQ(
        started.get_future().wait_for(std::chrono::seconds(5)),
        std::future_status::ready
    );
    auto task = synnax::task::Task{
        .rack = rack.key,
        .name = "failing read",
        .type = SYNTHETIC_TASK_TYPE,
    };
    ASSERT_NIL(rack.tasks.create(task));
    auto cmd_ch = ASSERT_NIL_P(client->channels.retrieve(synnax::task::CMD_CHANNEL));
    auto writer = ASSERT_NIL_P(client->telem.open_writer(
        {.channels = {cmd_ch.key}, .start = x::telem::TimeStamp::now()}
    ));
    const synnax::task::Command cmd{
        .task = task.key,
        .type = synnax::task::START_CMD_TYPE,
        .key = "start-cmd",
    };
    ASSERT_NIL(
        writer.write(x::telem::Frame(cmd_ch.key, x::telem::Series(cmd.to_json())))
    );
    ASSERT_NIL(writer.close());
    std::promise<bool> errored;
    auto read = errored.get_future();
    std::thread reader([&] {
        while (true) {
            auto [frame, err] = streamer.read();
            if (err) {
                errored.set_value(false);
                return;
            }
            for (const auto &j: frame.series->at(0).json_values()) {
                auto parser = x::json::Parser(j);
                const auto s = synnax::task::Status::parse(parser);
                if (s.details.task == task.key &&
                    s.variant == synnax::status::VARIANT_ERROR) {
                    errored.set_value(true);
                    return;
                }
            }
        }
    });
    // read has no deadline, so closing the stream ends a read that waits for a status
    // that never comes.
    if (read.wait_for(std::chrono::seconds(5)) != std::future_status::ready)
        streamer.close_send();
    reader.join();
    EXPECT_TRUE(read.get()) << "no error status";
    manager.stop();
    run.join();
    ASSERT_NIL(streamer.close());
}
}
