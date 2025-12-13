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
        synnax::TaskStatus status{
            .key = task.status_key(),
            .variant = status::variant::SUCCESS,
            .message = "task configured successfully",
            .details = synnax::TaskStatusDetails{
                .task = task.key,
            }
        };
        ctx->set_status(status);
    }

    std::string name() const override { return "echo"; }

    void exec(task::Command &cmd) override {
        synnax::TaskStatus status{
            .key = task.status_key(),
            .variant = status::variant::SUCCESS,
            .details = synnax::TaskStatusDetails{
                .task = task.key,
                .cmd = cmd.key,
                .running = true,
                .data = cmd.args,
            },
        };
        ctx->set_status(status);
    }

    void stop(bool will_reconfigure) override {
        synnax::TaskStatus status{
            .key = task.status_key(),
            .variant = status::variant::SUCCESS,
            .message = "task stopped successfully",
            .details = synnax::TaskStatusDetails{
                .task = task.key,
                .running = false,
            },
        };
        ctx->set_status(status);
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
    synnax::Channel status_chan;
    synnax::Streamer status_streamer;

    void SetUp() override {
        client = std::make_shared<synnax::Synnax>(new_test_client());
        this->rack = ASSERT_NIL_P(client->racks.create("test_rack"));
        this->status_chan = ASSERT_NIL_P(
            client->channels.retrieve(synnax::STATUS_SET_CHANNEL_NAME)
        );
        this->status_streamer = ASSERT_NIL_P(client->telem.open_streamer(
            synnax::StreamerConfig{.channels = {this->status_chan.key}}
        ));

        auto factory = std::make_unique<EchoTaskFactory>();
        task_manager = std::make_unique<task::Manager>(
            rack,
            client,
            std::move(factory)
        );

        std::promise<void> started_promise;
        const auto started_future = started_promise.get_future();
        task_thread = std::thread([&] {
            ASSERT_NIL(task_manager->run([&started_promise]() {
                started_promise.set_value();
            }));
        });
        const auto status = started_future.wait_for(std::chrono::seconds(5));
        ASSERT_EQ(status, std::future_status::ready)
            << "Manager failed to start within timeout";
    }

    void TearDown() override {
        ASSERT_NIL(this->status_streamer.close());
        if (task_manager != nullptr) {
            task_manager->stop();
            if (task_thread.joinable()) task_thread.join();
            task_manager.reset();
        }
    }
};

/// @brief Helper to wait for a task status matching specific criteria.
synnax::TaskStatus wait_for_task_status(
    synnax::Streamer &streamer,
    const synnax::Task &task,
    const std::function<bool(const synnax::TaskStatus &)> &predicate,
    const char *file,
    const int line,
    const std::chrono::milliseconds timeout = std::chrono::seconds(5)
) {
    synnax::TaskStatus result;
    xtest::eventually(
        [&]() {
            auto [frame, err] = streamer.read();
            if (err) return false;
            auto json_values = frame.series->at(0).json_values();
            for (const auto &j: json_values) {
                auto p = xjson::Parser(j);
                auto s = synnax::TaskStatus::parse(p);
                if (s.details.task == task.key && predicate(s)) {
                    result = s;
                    return true;
                }
            }
            return false;
        },
        [&]() { return "Timed out waiting for task status"; },
        file,
        line,
        timeout
    );
    return result;
}

#define WAIT_FOR_TASK_STATUS(streamer, task, predicate, ...)                           \
    wait_for_task_status(                                                              \
        streamer,                                                                      \
        task,                                                                          \
        predicate,                                                                     \
        __FILE__,                                                                      \
        __LINE__ __VA_OPT__(, ) __VA_ARGS__                                            \
    )

/// @brief it should correctly configure an echo task.
TEST_F(TaskManagerTestFixture, testEchoTask) {
    auto echo_task = synnax::Task(rack.key, "echo_task", "echo", "");
    ASSERT_NIL(rack.tasks.create(echo_task));

    // Wait for the successful configuration status
    auto status = WAIT_FOR_TASK_STATUS(
        this->status_streamer,
        echo_task,
        [](const synnax::TaskStatus &s) {
            return s.variant == status::variant::SUCCESS &&
                   s.message == "task configured successfully";
        }
    );
    ASSERT_EQ(status.details.task, echo_task.key);
    ASSERT_EQ(status.variant, status::variant::SUCCESS);
    ASSERT_EQ(status.message, "task configured successfully");
}

/// @brief it should stop and remove the task.
TEST_F(TaskManagerTestFixture, testEchoTaskDelete) {
    auto echo_task = synnax::Task(rack.key, "echo_task", "echo", "");
    ASSERT_NIL(rack.tasks.create(echo_task));

    // Wait for task to be configured
    WAIT_FOR_TASK_STATUS(
        this->status_streamer,
        echo_task,
        [](const synnax::TaskStatus &s) {
            return s.variant == status::variant::SUCCESS &&
                   s.message == "task configured successfully";
        }
    );

    // Delete the task
    ASSERT_NIL(rack.tasks.del(echo_task.key));

    // Wait for the stop status
    auto state = WAIT_FOR_TASK_STATUS(
        this->status_streamer,
        echo_task,
        [](const synnax::TaskStatus &s) {
            return s.variant == status::variant::SUCCESS &&
                   s.message == "task stopped successfully";
        }
    );
    ASSERT_EQ(state.details.task, echo_task.key);
    ASSERT_EQ(state.variant, status::variant::SUCCESS);
    ASSERT_EQ(state.message, "task stopped successfully");
}

/// @brief it should execute an echo command on the task.
TEST_F(TaskManagerTestFixture, testEchoTaskCommand) {
    auto sy_task_cmd = ASSERT_NIL_P(client->channels.retrieve("sy_task_cmd"));
    auto writer = ASSERT_NIL_P(client->telem.open_writer(
        synnax::WriterConfig{
            .channels = {sy_task_cmd.key},
            .start = telem::TimeStamp::now(),
        }
    ));
    auto echo_task = synnax::Task(rack.key, "echo_task", "echo", "");
    ASSERT_NIL(rack.tasks.create(echo_task));

    // Wait for task to be configured
    WAIT_FOR_TASK_STATUS(
        this->status_streamer,
        echo_task,
        [](const synnax::TaskStatus &s) {
            return s.variant == status::variant::SUCCESS &&
                   s.message == "task configured successfully";
        }
    );

    // Create and send a command
    auto cmd = task::Command(
        echo_task.key,
        "test_command",
        json{{"message", "hello world"}}
    );
    cmd.key = "my_command";
    ASSERT_NIL(
        writer.write(synnax::Frame(sy_task_cmd.key, telem::Series(cmd.to_json())))
    );
    ASSERT_NIL(writer.close());

    // Wait for command execution status
    auto status = WAIT_FOR_TASK_STATUS(
        this->status_streamer,
        echo_task,
        [&cmd](const synnax::TaskStatus &s) { return s.details.cmd == cmd.key; }
    );
    ASSERT_EQ(status.details.task, echo_task.key);
    ASSERT_EQ(status.key, echo_task.status_key());
    ASSERT_EQ(status.details.cmd, cmd.key);
    ASSERT_EQ(status.variant, status::variant::SUCCESS);
    ASSERT_EQ(status.details.data["message"], "hello world");
}

/// @brief should ignore tasks for a different rack.
TEST_F(TaskManagerTestFixture, testIgnoreDifferentRackTask) {
    // Create a different rack
    auto other_rack = ASSERT_NIL_P(client->racks.create("other_rack"));

    // Create a task for the other rack
    auto echo_task = synnax::Task(other_rack.key, "echo_task", "echo", "");
    ASSERT_NIL(other_rack.tasks.create(echo_task));

    // Set up variables for thread communication
    std::atomic received_state = false;

    // Start reading thread
    std::thread reader([&] {
        auto [frame, err] = this->status_streamer.read();
        auto json_vs = frame.series->at(0).json_values();
        for (const auto &j: json_vs) {
            auto parser = xjson::Parser(j);
            auto status = synnax::TaskStatus::parse(parser);
            if (status.variant != status::variant::WARNING) received_state = true;
        }
    });

    // Signal thread to stop and wait for it
    this->status_streamer.close_send();
    reader.join();

    // Verify no state changes were received
    ASSERT_FALSE(received_state)
        << "Received unexpected state change for different rack's task";
}

/// @brief it should stop all tasks when the manager is shut down.
TEST_F(TaskManagerTestFixture, testStopTaskOnShutdown) {
    auto echo_task = synnax::Task(rack.key, "echo_task", "echo", "");
    ASSERT_NIL(rack.tasks.create(echo_task));

    // Wait for task to be configured
    WAIT_FOR_TASK_STATUS(
        this->status_streamer,
        echo_task,
        [](const synnax::TaskStatus &s) {
            return s.variant == status::variant::SUCCESS &&
                   s.message == "task configured successfully";
        }
    );

    // Stop the task manager
    task_manager->stop();
    task_thread.join();

    // Wait for the stop status
    auto state = WAIT_FOR_TASK_STATUS(
        this->status_streamer,
        echo_task,
        [](const synnax::TaskStatus &s) {
            return s.variant == status::variant::SUCCESS &&
                   s.message == "task stopped successfully";
        }
    );
    ASSERT_EQ(state.details.task, echo_task.key);
    ASSERT_EQ(state.variant, status::variant::SUCCESS);
    ASSERT_EQ(state.message, "task stopped successfully");
}

/// @brief it should ignore snapshot tasks during configuration.
TEST_F(TaskManagerTestFixture, testIgnoresSnapshot) {
    auto snapshot_task = synnax::Task(rack.key, "snapshot_task", "echo", "");
    snapshot_task.snapshot = true;
    ASSERT_NIL(rack.tasks.create(snapshot_task));
    std::atomic received_state = false;
    std::thread reader([&] {
        auto [frame, err] = this->status_streamer.read();
        if (err) return;
        auto json_vs = frame.series->at(0).json_values();
        for (const auto &j: json_vs) {
            auto parser = xjson::Parser(j);
            auto status = synnax::TaskStatus::parse(parser);
            if (status.variant != status::variant::WARNING &&
                status.details.task == snapshot_task.key) {
                received_state = true;
                break;
            }
        }
    });
    std::this_thread::sleep_for(std::chrono::milliseconds(500));
    this->status_streamer.close_send();
    reader.join();
    ASSERT_FALSE(received_state)
        << "Received unexpected state change for snapshot task";
}
