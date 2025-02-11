// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// external
#include "gtest/gtest.h"
#include "nlohmann/json.hpp"

/// module
#include "x/cpp/breaker/breaker.h"

/// internal
#include "driver/task/task.h"
#include "driver/testutil/testutil.h"


using json = nlohmann::json;

class MockEchoTask final : public task::Task {
    const std::shared_ptr<task::Context> ctx;
    const synnax::Task task;
public:

    explicit MockEchoTask(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    ) : ctx(ctx), task(task) {
            ctx->set_state({
            .task = task.key,
            .variant = "success",
            .details = json{
                {"message", "task configured successfully"}
            }
        });
    }

    std::string name() override { return "echo"; }

    void exec(task::Command &cmd) override {
        ctx->set_state({
            .task = task.key,
            .key = cmd.key,
            .variant = "success",
            .details = cmd.args,
        });
    }

    void stop() override {
        ctx->set_state({
            .task = task.key,
            .variant = "success",
            .details = json{
                {"message", "task stopped successfully"}
            }
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

    void SetUp() override {
        client = std::make_shared<synnax::Synnax>(new_test_client());
        auto [r, err] = client->hardware.create_rack("test_rack");
        ASSERT_FALSE(err) << err.message();
        rack = r;
    }

    void TearDown() override {
        if (task_manager != nullptr) {
            task_manager->stop();
        }
    }
};

/// @brief should start and stop a task manager.
TEST_F(TaskManagerTestFixture, basicStartStop) {
    std::unique_ptr<EchoTaskFactory> factory = std::make_unique<EchoTaskFactory>();
    task_manager = std::make_unique<task::Manager>(
        client,
        std::move(factory),
        breaker::default_config("name")
    );
    auto err = task_manager->start();
    ASSERT_FALSE(err) << err.message();
    err = task_manager->stop();
    ASSERT_FALSE(err) << err.message();
}

/// @brief it should correctly configure an echo task.
TEST_F(TaskManagerTestFixture, testEchoTask) {
    auto factory = std::make_unique<EchoTaskFactory>();
    task_manager = std::make_unique<task::Manager>(
        rack.key,
        client->auth->cluster_info.cluster_key,
        [](synnax::RackKey, std::string) { return xerrors::NIL; },
        client,
        std::move(factory),
        breaker::default_config("")
    );
    auto start_err = task_manager->start();
    ASSERT_FALSE(start_err) << start_err;

    auto [sy_task_state, ch_err] = client->channels.retrieve("sy_task_state");
    ASSERT_FALSE(ch_err) << ch_err;

    auto [streamer, s_err] = client->telem.open_streamer(synnax::StreamerConfig{
        .channels = {sy_task_state.key}
    });
    ASSERT_FALSE(s_err) << s_err;

    std::this_thread::sleep_for(std::chrono::milliseconds(5));

    auto echo_task = synnax::Task(
        rack.key,
        "echo_task",
        "echo",
        ""
    );
    auto t_err = rack.tasks.create(echo_task);
    ASSERT_FALSE(t_err) << t_err;

    auto [f, r_err] = streamer.read();
    ASSERT_FALSE(r_err) << r_err;
    ASSERT_EQ(f.size(), 1);
    std::string state_str;
    f.at(sy_task_state.key, 0, state_str);
    auto parser = config::Parser(state_str);
    auto state = task::State::parse(parser) ;
    ASSERT_EQ(state.task, echo_task.key);
    ASSERT_EQ(state.variant, "success");
    ASSERT_EQ(state.details["message"], "task configured successfully");
    const auto close_err = streamer.close();
    ASSERT_FALSE(close_err) << close_err;
}

TEST_F(TaskManagerTestFixture, testEchoTaskDelete) {
    auto factory = std::make_unique<EchoTaskFactory>();
    task_manager = std::make_unique<task::Manager>(
        rack.key,
        client->auth->cluster_info.cluster_key,
        [](synnax::RackKey, std::string) { return xerrors::NIL; },
        client,
        std::move(factory),
        breaker::default_config("")
    );
    auto start_err = task_manager->start();
    ASSERT_FALSE(start_err) << start_err;

    auto [sy_task_state, ch_err] = client->channels.retrieve("sy_task_state");
    ASSERT_FALSE(ch_err) << ch_err;

    auto [streamer, s_err] = client->telem.open_streamer(synnax::StreamerConfig{
        .channels = {sy_task_state.key}
    });
    ASSERT_FALSE(s_err) << s_err;

    std::this_thread::sleep_for(std::chrono::milliseconds(5));

    auto echo_task = synnax::Task(
        rack.key,
        "echo_task",
        "echo",
        ""
    );
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
    std::string state_str;
    f2.at(sy_task_state.key, 0, state_str);
    auto parser = config::Parser(state_str);
    auto state = task::State::parse(parser);
    ASSERT_EQ(state.task, echo_task.key);
    ASSERT_EQ(state.variant, "success");
    ASSERT_EQ(state.details["message"], "task stopped successfully");
    auto close_err = streamer.close();
    ASSERT_FALSE(close_err) << close_err;
}

TEST_F(TaskManagerTestFixture, testEchoTaskCommand) {
    auto factory = std::make_unique<EchoTaskFactory>();
    task_manager = std::make_unique<task::Manager>(
        rack.key,
        client->auth->cluster_info.cluster_key,
        [](synnax::RackKey, std::string) { return xerrors::NIL; },
        client,
        std::move(factory),
        breaker::default_config("")
    );
    auto start_err = task_manager->start();
    ASSERT_FALSE(start_err) << start_err;

    auto [sy_task_state, ch_err] = client->channels.retrieve("sy_task_state");
    ASSERT_FALSE(ch_err) << ch_err;

    auto [streamer, s_err] = client->telem.open_streamer(synnax::StreamerConfig{
        .channels = {sy_task_state.key}
    });
    ASSERT_FALSE(s_err) << s_err;
    auto [sy_task_cmd, c_err] = client->channels.retrieve("sy_task_cmd");
    auto [writer, w_err] = client->telem.open_writer(synnax::WriterConfig{
        .start = telem::TimeStamp::now(),
        .channels = {sy_task_cmd.key}
    });
    ASSERT_FALSE(w_err) << w_err;


    std::this_thread::sleep_for(std::chrono::milliseconds(100));

    auto echo_task = synnax::Task(
        rack.key,
        "echo_task",
        "echo",
        ""
    );
    auto t_err = rack.tasks.create(echo_task);
    ASSERT_FALSE(t_err) << t_err;

    // Wait for task to be configured
    auto [f1, r_err1] = streamer.read();
    ASSERT_FALSE(r_err1) << r_err1;

    // Create and send a command
    auto cmd = task::Command(echo_task.key, "test_command", json{
        {"message", "hello world"}
    });
    auto ok = writer.write(synnax::Frame(sy_task_cmd.key, telem::Series(cmd.to_json())));
    ASSERT_TRUE(ok);
    auto w_close_err = writer.close();
    ASSERT_FALSE(w_close_err) << w_close_err;

    // Read the command execution state
    auto [f2, r_err2] = streamer.read();
    ASSERT_FALSE(r_err2) << r_err2;
    ASSERT_EQ(f2.size(), 1);
    std::string state_str;
    f2.at(sy_task_state.key, 0, state_str);
    auto parser = config::Parser(state_str);
    auto [task, key, variant, details] = task::State::parse(parser);
    ASSERT_EQ(task, echo_task.key);
    ASSERT_EQ(key, cmd.key);
    ASSERT_EQ(variant, "success");
    ASSERT_EQ(details["message"], "hello world");
    auto close_err = streamer.close();
    ASSERT_FALSE(close_err) << close_err;
}