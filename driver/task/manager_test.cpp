// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"
#include "nlohmann/json.hpp"

#include "client/cpp/testutil/testutil.h"
#include "x/cpp/breaker/breaker.h"
#include "x/cpp/status/status.h"
#include "x/cpp/xtest/xtest.h"

#include "driver/task/task.h"

using json = nlohmann::json;

class MockEchoTask final : public task::Task {
    const std::shared_ptr<task::Context> ctx;
    const synnax::Task task;

public:
    explicit MockEchoTask(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    ):
        ctx(ctx), task(task) {
        ctx->set_status(
            {.variant = status::variant::SUCCESS,
             .message = "task configured successfully",
             .details = synnax::TaskStatusDetails{
                 .task = task.key,
             }}
        );
    }

    std::string name() const override { return "echo"; }

    void exec(task::Command &cmd) override {
        ctx->set_status({
            .key = cmd.key,
            .variant = status::variant::SUCCESS,
            .details = synnax::TaskStatusDetails{
                .task = task.key,
                .running = true,
                .data = cmd.args,
            },
        });
    }

    void stop(bool will_reconfigure) override {
        ctx->set_status({
            .variant = status::variant::SUCCESS,
            .message = "task stopped successfully",
            .details = synnax::TaskStatusDetails{
                .task = task.key,
                .running = false,
            },
        });
    }
};

class EchoTaskFactory final : public task::Factory {
public:
    bool configured = false;

    std::pair<std::unique_ptr<task::Task>, bool> configure_task(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    ) override {
        if (task.type != "echo") return {nullptr, false};
        return {std::make_unique<MockEchoTask>(ctx, task), true};
    }
};

class TaskManagerTestFixture : public testing::Test {
protected:
    std::shared_ptr<synnax::Synnax> client;
    std::unique_ptr<task::Manager> task_manager;
    synnax::Rack rack;
    std::thread task_thread;

    void SetUp() override {
        client = std::make_shared<synnax::Synnax>(new_test_client());
        auto [r, err] = client->hardware.create_rack("test_rack");
        ASSERT_FALSE(err) << err.message();
        rack = r;

        auto factory = std::make_unique<EchoTaskFactory>();
        task_manager = std::make_unique<task::Manager>(
            rack,
            client,
            std::move(factory)
        );

        std::promise<void> started_promise;
        const auto started_future = started_promise.get_future();
        task_thread = std::thread([&] {
            const auto t_err = task_manager->run([&started_promise]() {
                started_promise.set_value();
            });
            ASSERT_FALSE(t_err) << t_err.message();
        });
        const auto status = started_future.wait_for(std::chrono::seconds(5));
        ASSERT_EQ(status, std::future_status::ready)
            << "Manager failed to start within timeout";
    }

    void TearDown() override {
        if (task_manager != nullptr) {
            task_manager->stop();
            if (task_thread.joinable()) task_thread.join();
            task_manager.reset();
        }
    }
};

/// @brief it should correctly configure an echo task.
TEST_F(TaskManagerTestFixture, testEchoTask) {
    auto [sy_task_status, ch_err] = client->channels.retrieve(
        synnax::TASK_STATUS_CHANNEL_NAME
    );
    ASSERT_FALSE(ch_err) << ch_err;

    auto [streamer, s_err] = client->telem.open_streamer(
        synnax::StreamerConfig{.channels = {sy_task_status.key}}
    );
    ASSERT_FALSE(s_err) << s_err;

    auto echo_task = synnax::Task(rack.key, "echo_task", "echo", "");
    auto t_err = rack.tasks.create(echo_task);
    ASSERT_FALSE(t_err) << t_err;

    auto [f, r_err] = streamer.read();
    ASSERT_FALSE(r_err) << r_err;
    ASSERT_EQ(f.size(), 1);
    auto state_str = f.at<std::string>(sy_task_status.key, 0);
    auto parser = xjson::Parser(state_str);
    auto status = synnax::TaskStatus::parse(parser);
    ASSERT_EQ(status.details.task, echo_task.key);
    ASSERT_EQ(status.variant, status::variant::SUCCESS);
    ASSERT_EQ(status.message, "task configured successfully");
    const auto close_err = streamer.close();
    ASSERT_FALSE(close_err) << close_err;
}

/// @brief it should stop and remove the task.
TEST_F(TaskManagerTestFixture, testEchoTaskDelete) {
    auto [sy_task_status, ch_err] = client->channels.retrieve(
        synnax::TASK_STATUS_CHANNEL_NAME
    );
    ASSERT_FALSE(ch_err) << ch_err;

    auto [streamer, s_err] = client->telem.open_streamer(
        synnax::StreamerConfig{.channels = {sy_task_status.key}}
    );
    ASSERT_FALSE(s_err) << s_err;

    auto echo_task = synnax::Task(rack.key, "echo_task", "echo", "");
    auto t_err = rack.tasks.create(echo_task);
    ASSERT_FALSE(t_err) << t_err;

    // Wait for task to be configured
    auto [f1, r_err1] = streamer.read();
    ASSERT_FALSE(r_err1) << r_err1;

    // Delete the task
    auto d_err = rack.tasks.del(echo_task.key);
    ASSERT_FALSE(d_err) << d_err;

    // Read the stop state
    auto [f2, r_err2] = streamer.read();
    ASSERT_FALSE(r_err2) << r_err2;
    ASSERT_EQ(f2.size(), 1);
    auto state_str = f2.at<std::string>(sy_task_status.key, 0);
    auto parser = xjson::Parser(state_str);
    auto state = synnax::TaskStatus::parse(parser);
    ASSERT_EQ(state.details.task, echo_task.key);
    ASSERT_EQ(state.variant, status::variant::SUCCESS);
    ASSERT_EQ(state.message, "task stopped successfully");
    auto close_err = streamer.close();
    ASSERT_FALSE(close_err) << close_err;
}

/// @brief it should execute an echo command on the task.
TEST_F(TaskManagerTestFixture, testEchoTaskCommand) {
    auto [sy_task_status, ch_err] = client->channels.retrieve(
        synnax::TASK_STATUS_CHANNEL_NAME
    );
    ASSERT_FALSE(ch_err) << ch_err;
    auto [streamer, s_err] = client->telem.open_streamer(
        synnax::StreamerConfig{.channels = {sy_task_status.key}}
    );
    ASSERT_FALSE(s_err) << s_err;
    auto [sy_task_cmd, c_err] = client->channels.retrieve("sy_task_cmd");
    auto [writer, w_err] = client->telem.open_writer(
        synnax::WriterConfig{
            .channels = {sy_task_cmd.key},
            .start = telem::TimeStamp::now(),
        }
    );
    ASSERT_FALSE(w_err) << w_err;
    auto echo_task = synnax::Task(rack.key, "echo_task", "echo", "");
    auto t_err = rack.tasks.create(echo_task);
    ASSERT_FALSE(t_err) << t_err;

    // Wait for task to be configured
    auto [f1, r_err1] = streamer.read();
    ASSERT_FALSE(r_err1) << r_err1;

    // Create and send a command
    auto cmd = task::Command(
        echo_task.key,
        "test_command",
        json{{"message", "hello world"}}
    );
    ASSERT_NIL(
        writer.write(telem::Frame(sy_task_cmd.key, telem::Series(cmd.to_json())))
    );
    auto w_close_err = writer.close();
    ASSERT_FALSE(w_close_err) << w_close_err;

    // Read the command execution state
    auto [f2, r_err2] = streamer.read();
    ASSERT_FALSE(r_err2) << r_err2;
    ASSERT_EQ(f2.size(), 1);
    auto state_str = f2.at<std::string>(sy_task_status.key, 0);
    auto parser = xjson::Parser(state_str);
    auto status = synnax::TaskStatus::parse(parser);
    ASSERT_EQ(status.details.task, echo_task.key);
    ASSERT_EQ(status.key, cmd.key);
    ASSERT_EQ(status.variant, status::variant::SUCCESS);
    ASSERT_EQ(status.details.data["message"], "hello world");
    auto close_err = streamer.close();
    ASSERT_FALSE(close_err) << close_err;
}

/// @brief should ignore tasks for a different rack.
TEST_F(TaskManagerTestFixture, testIgnoreDifferentRackTask) {
    auto [sy_task_status, ch_err] = client->channels.retrieve(
        synnax::TASK_STATUS_CHANNEL_NAME
    );
    ASSERT_FALSE(ch_err) << ch_err;

    auto [streamer, s_err] = client->telem.open_streamer(
        synnax::StreamerConfig{.channels = {sy_task_status.key}}
    );
    ASSERT_FALSE(s_err) << s_err;

    // Create a different rack
    auto [other_rack, r_err] = client->hardware.create_rack("other_rack");
    ASSERT_FALSE(r_err) << r_err;

    // Create a task for the other rack
    auto echo_task = synnax::Task(other_rack.key, "echo_task", "echo", "");
    auto t_err = other_rack.tasks.create(echo_task);
    ASSERT_FALSE(t_err) << t_err;

    // Set up variables for thread communication
    std::atomic received_state = false;

    // Start reading thread
    std::thread reader([&] {
        auto [frame, err] = streamer.read();
        if (!err) received_state = true;
    });

    // Signal thread to stop and wait for it
    streamer.close_send();
    reader.join();

    // Verify no state changes were received
    ASSERT_FALSE(received_state)
        << "Received unexpected state change for different rack's task";

    const auto close_err = streamer.close();
    ASSERT_FALSE(close_err) << close_err;
}

/// @brief it should stop all tasks when the manager is shut down.
TEST_F(TaskManagerTestFixture, testStopTaskOnShutdown) {
    auto [sy_task_status, ch_err] = client->channels.retrieve(
        synnax::TASK_STATUS_CHANNEL_NAME
    );
    ASSERT_FALSE(ch_err) << ch_err;

    auto [streamer, s_err] = client->telem.open_streamer(
        synnax::StreamerConfig{.channels = {sy_task_status.key}}
    );
    ASSERT_FALSE(s_err) << s_err;

    auto echo_task = synnax::Task(rack.key, "echo_task", "echo", "");
    auto t_err = rack.tasks.create(echo_task);
    ASSERT_FALSE(t_err) << t_err;

    // Wait for task to be configured
    auto [f1, r_err1] = streamer.read();
    ASSERT_FALSE(r_err1) << r_err1;

    // Stop the task manager
    task_manager->stop();
    task_thread.join();

    // Verify that the task was stopped
    auto [f2, r_err2] = streamer.read();
    ASSERT_FALSE(r_err2) << r_err2;
    ASSERT_EQ(f2.size(), 1);

    auto state_str = f2.at<std::string>(sy_task_status.key, 0);
    auto parser = xjson::Parser(state_str);
    auto state = synnax::TaskStatus::parse(parser);

    ASSERT_EQ(state.details.task, echo_task.key);
    ASSERT_EQ(state.variant, status::variant::SUCCESS);
    ASSERT_EQ(state.message, "task stopped successfully");

    const auto close_err = streamer.close();
    ASSERT_FALSE(close_err) << close_err;
}

/// @brief it should ignore snapshot tasks during configuration.
TEST_F(TaskManagerTestFixture, testIgnoresSnapshot) {
    auto [sy_task_status, ch_err] = client->channels.retrieve(
        synnax::TASK_STATUS_CHANNEL_NAME
    );
    ASSERT_FALSE(ch_err) << ch_err;
    auto [streamer, s_err] = client->telem.open_streamer(
        synnax::StreamerConfig{.channels = {sy_task_status.key}}
    );
    ASSERT_FALSE(s_err) << s_err;
    auto snapshot_task = synnax::Task(rack.key, "snapshot_task", "echo", "");
    snapshot_task.snapshot = true;
    auto t_err = rack.tasks.create(snapshot_task);
    ASSERT_FALSE(t_err) << t_err;
    std::atomic received_state = false;
    std::thread reader([&] {
        auto [frame, err] = streamer.read();
        if (err) return;
        auto json_vs = frame.series->at(0).json_values();
        for (const auto &j: json_vs) {
            auto parser = xjson::Parser(j);
            auto status = synnax::TaskStatus::parse(parser);
            if (status.details.task == snapshot_task.key) {
                received_state = true;
                break;
            }
        }
    });
    std::this_thread::sleep_for(std::chrono::milliseconds(500));
    streamer.close_send();
    reader.join();
    ASSERT_FALSE(received_state)
        << "Received unexpected state change for snapshot task";
    const auto close_err = streamer.close();
    ASSERT_FALSE(close_err) << close_err;
}
