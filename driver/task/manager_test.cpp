// Copyright 2026 Synnax Labs, Inc.
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
#include "x/cpp/test/test.h"

#include "driver/task/task.h"

namespace driver::task {
class MockEchoTask final : public task::Task {
    const std::shared_ptr<task::Context> ctx;
    const synnax::task::Task sy_task;

public:
    MockEchoTask(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::task::Task &task
    ):
        ctx(ctx), sy_task(task) {
        synnax::task::Status status{
            .key = synnax::task::status_key(task),
            .variant = x::status::VARIANT_SUCCESS,
            .message = "configured",
            .details = {.task = task.key}
        };
        ctx->set_status(status);
    }

    std::string name() const override { return "echo"; }

    void exec(synnax::task::Command &cmd) override {
        synnax::task::Status status{
            .key = synnax::task::status_key(sy_task),
            .variant = x::status::VARIANT_SUCCESS,
            .details =
                {.task = sy_task.key, .cmd = cmd.key, .running = true, .data = cmd.args}
        };
        ctx->set_status(status);
    }

    void stop(bool) override {
        synnax::task::Status status{
            .key = synnax::task::status_key(sy_task),
            .variant = x::status::VARIANT_SUCCESS,
            .message = "stopped",
            .details = {.task = sy_task.key, .running = false}
        };
        ctx->set_status(status);
    }
};

class EchoTaskFactory final : public task::Factory {
public:
    std::pair<std::unique_ptr<task::Task>, bool> configure_task(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::task::Task &task
    ) override {
        if (task.type != "echo") return {nullptr, false};
        return {std::make_unique<MockEchoTask>(ctx, task), true};
    }
};

class BlockingTask final : public task::Task {
public:
    BlockingTask(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::task::Task &task,
        std::atomic<bool> &started,
        std::atomic<bool> &done,
        std::condition_variable &cv,
        std::mutex &mu
    ) {
        started = true;
        cv.notify_all();
        std::unique_lock lock(mu);
        cv.wait(lock, [&] { return done.load(); });
        synnax::task::Status status{
            .key = synnax::task::status_key(task),
            .variant = x::status::VARIANT_SUCCESS,
            .message = "configured",
            .details = {.task = task.key}
        };
        ctx->set_status(status);
    }

    std::string name() const override { return "blocking"; }
    void exec(synnax::task::Command &) override {}
    void stop(bool) override {}
};

class BlockingTaskFactory final : public task::Factory {
public:
    std::atomic<bool> started{false};
    std::atomic<bool> done{false};
    std::condition_variable cv;
    std::mutex mu;

    std::pair<std::unique_ptr<task::Task>, bool> configure_task(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::task::Task &task
    ) override {
        if (task.type == "blocking")
            return {
                std::make_unique<BlockingTask>(ctx, task, started, done, cv, mu),
                true
            };
        if (task.type == "echo")
            return {std::make_unique<MockEchoTask>(ctx, task), true};
        return {nullptr, false};
    }

    void release() {
        done = true;
        cv.notify_all();
    }
};

struct TrackingTaskState {
    std::atomic<int> exec_count{0};
    std::vector<std::string> cmd_order;
    std::mutex cmd_order_mu;
    std::atomic<bool> stopped{false};
    std::atomic<bool> stop_will_reconfigure{false};
};

class TrackingTask final : public task::Task {
public:
    synnax::task::Task sy_task;
    std::shared_ptr<TrackingTaskState> state;

    TrackingTask(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::task::Task &task,
        std::shared_ptr<TrackingTaskState> state
    ):
        sy_task(task), state(std::move(state)) {
        synnax::task::Status status{
            .key = synnax::task::status_key(task),
            .variant = x::status::VARIANT_SUCCESS,
            .message = "configured",
            .details = {.task = task.key}
        };
        ctx->set_status(status);
    }

    std::string name() const override { return "tracking"; }

    void exec(synnax::task::Command &cmd) override {
        state->exec_count++;
        std::lock_guard lock(state->cmd_order_mu);
        state->cmd_order.push_back(cmd.key);
    }

    void stop(bool will_reconfigure) override {
        state->stopped = true;
        state->stop_will_reconfigure = will_reconfigure;
    }

    std::vector<std::string> get_cmd_order() {
        std::lock_guard lock(state->cmd_order_mu);
        return state->cmd_order;
    }
};

class TrackingTaskFactory final : public task::Factory {
public:
    std::vector<std::shared_ptr<TrackingTaskState>> task_states;
    std::mutex mu;

    std::pair<std::unique_ptr<task::Task>, bool> configure_task(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::task::Task &task
    ) override {
        if (task.type == "tracking") {
            auto state = std::make_shared<TrackingTaskState>();
            auto t = std::make_unique<TrackingTask>(ctx, task, state);
            std::lock_guard lock(mu);
            task_states.push_back(state);
            return {std::move(t), true};
        }
        return {nullptr, false};
    }
};

class TimeoutTask final : public task::Task {
public:
    std::atomic<bool> &release;
    std::condition_variable &cv;
    std::mutex &mu;

    TimeoutTask(
        const std::shared_ptr<task::Context> &,
        const synnax::task::Task &,
        std::atomic<bool> &release,
        std::condition_variable &cv,
        std::mutex &mu
    ):
        release(release), cv(cv), mu(mu) {
        std::unique_lock lock(mu);
        cv.wait(lock, [&] { return this->release.load(); });
    }

    std::string name() const override { return "timeout"; }
    void exec(synnax::task::Command &) override {}
    void stop(bool) override {}
};

class TimeoutTaskFactory final : public task::Factory {
public:
    std::atomic<bool> release{false};
    std::condition_variable cv;
    std::mutex mu;

    std::pair<std::unique_ptr<task::Task>, bool> configure_task(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::task::Task &task
    ) override {
        if (task.type == "timeout")
            return {std::make_unique<TimeoutTask>(ctx, task, release, cv, mu), true};
        return {nullptr, false};
    }

    void release_all() {
        release = true;
        cv.notify_all();
    }
};

synnax::task::Status wait_for_task_status(
    synnax::framer::Streamer &streamer,
    const synnax::task::Task &task,
    const std::function<bool(const synnax::task::Status &)> &pred,
    const char *file,
    const int line,
    x::telem::TimeSpan timeout = 5 * x::telem::SECOND
) {
    synnax::task::Status result;
    x::test::eventually(
        [&]() {
            auto [frame, err] = streamer.read();
            if (err) return false;
            for (const auto &j: frame.series->at(0).json_values()) {
                auto parser = x::json::Parser(j);
                auto s = synnax::task::Status::parse(parser);
                if (s.details.task == task.key && pred(s)) {
                    result = s;
                    return true;
                }
            }
            return false;
        },
        [&]() { return "Timed out waiting for task status"; },
        file,
        line,
        std::chrono::duration_cast<std::chrono::milliseconds>(timeout.chrono())
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

#define EVENTUALLY(condition, failure_message, ...)                                    \
    ::x::test::eventually(                                                             \
        condition,                                                                     \
        failure_message,                                                               \
        __FILE__,                                                                      \
        __LINE__ __VA_OPT__(, ) __VA_ARGS__                                            \
    )

class TaskManagerTest : public testing::Test {
protected:
    std::shared_ptr<synnax::Synnax> client;
    std::unique_ptr<task::Manager> manager;
    synnax::rack::Rack rack;
    std::thread thread;
    synnax::framer::Streamer streamer;
    bool setup_succeeded = false;

    void SetUp() override {
        client = std::make_shared<synnax::Synnax>(new_test_client());
        rack = ASSERT_NIL_P(client->racks.create("test_rack"));
        auto ch = ASSERT_NIL_P(
            client->channels.retrieve(synnax::status::STATUS_SET_CHANNEL_NAME)
        );
        streamer = ASSERT_NIL_P(client->telem.open_streamer({.channels = {ch.key}}));
        setup_succeeded = true;
    }

    void start_manager(
        std::unique_ptr<task::Factory> factory,
        task::ManagerConfig cfg = {}
    ) {
        manager = std::make_unique<task::Manager>(
            rack,
            client,
            std::move(factory),
            cfg
        );
        std::promise<void> started;
        thread = std::thread([&] {
            ASSERT_NIL(manager->run([&] { started.set_value(); }));
        });
        ASSERT_EQ(
            started.get_future().wait_for((5 * x::telem::SECOND).chrono()),
            std::future_status::ready
        );
    }

    void TearDown() override {
        if (setup_succeeded) { ASSERT_NIL(streamer.close()); }
        if (manager) {
            manager->stop();
            if (thread.joinable()) thread.join();
        }
    }
};

TEST_F(TaskManagerTest, Configure) {
    start_manager(std::make_unique<EchoTaskFactory>());
    auto task = synnax::task::Task{.name = "t", .type = "echo"};
    ASSERT_NIL(rack.tasks.create(task));
    auto s = WAIT_FOR_TASK_STATUS(streamer, task, [](const synnax::task::Status &s) {
        return s.variant == x::status::VARIANT_SUCCESS && s.message == "configured";
    });
    ASSERT_EQ(s.details.task, task.key);
}

TEST_F(TaskManagerTest, Delete) {
    start_manager(std::make_unique<EchoTaskFactory>());
    auto task = synnax::task::Task{.name = "t", .type = "echo"};
    ASSERT_NIL(rack.tasks.create(task));
    WAIT_FOR_TASK_STATUS(streamer, task, [](const synnax::task::Status &s) {
        return s.message == "configured";
    });
    ASSERT_NIL(rack.tasks.del(task.key));
    auto s = WAIT_FOR_TASK_STATUS(streamer, task, [](const synnax::task::Status &s) {
        return s.message == "stopped";
    });
    ASSERT_EQ(s.details.task, task.key);
}

TEST_F(TaskManagerTest, Command) {
    start_manager(std::make_unique<EchoTaskFactory>());
    auto cmd_ch = ASSERT_NIL_P(client->channels.retrieve("sy_task_cmd"));
    auto writer = ASSERT_NIL_P(client->telem.open_writer(
        {.channels = {cmd_ch.key}, .start = x::telem::TimeStamp::now()}
    ));
    auto task = synnax::task::Task{.name = "t", .type = "echo"};
    ASSERT_NIL(rack.tasks.create(task));
    WAIT_FOR_TASK_STATUS(streamer, task, [](const synnax::task::Status &s) {
        return s.message == "configured";
    });

    auto cmd = synnax::task::Command{
        .task = task.key,
        .type = "test",
        .key = "cmd1",
        .args = json{{"msg", "hi"}}
    };
    ASSERT_NIL(
        writer.write(x::telem::Frame(cmd_ch.key, x::telem::Series(cmd.to_json())))
    );
    ASSERT_NIL(writer.close());

    auto s = WAIT_FOR_TASK_STATUS(streamer, task, [](const synnax::task::Status &s) {
        return s.details.cmd == "cmd1";
    });
    ASSERT_EQ((*s.details.data)["msg"], "hi");
}

TEST_F(TaskManagerTest, IgnoresForeignRack) {
    start_manager(std::make_unique<EchoTaskFactory>());
    auto other = ASSERT_NIL_P(client->racks.create("other"));
    auto task = synnax::task::Task{.name = "t", .type = "echo"};
    ASSERT_NIL(other.tasks.create(task));

    std::atomic<bool> received = false;
    std::thread reader([&] {
        auto [frame, err] = streamer.read();
        for (const auto &j: frame.series->at(0).json_values()) {
            auto parser = x::json::Parser(j);
            auto s = synnax::task::Status::parse(parser);
            if (s.variant != x::status::VARIANT_WARNING) received = true;
        }
    });
    streamer.close_send();
    reader.join();
    ASSERT_FALSE(received);
}

TEST_F(TaskManagerTest, StopOnShutdown) {
    start_manager(std::make_unique<EchoTaskFactory>());
    auto task = synnax::task::Task{.name = "t", .type = "echo"};
    ASSERT_NIL(rack.tasks.create(task));
    WAIT_FOR_TASK_STATUS(streamer, task, [](const synnax::task::Status &s) {
        return s.message == "configured";
    });
    manager->stop();
    thread.join();
    auto s = WAIT_FOR_TASK_STATUS(streamer, task, [](const synnax::task::Status &s) {
        return s.message == "stopped";
    });
    ASSERT_EQ(s.details.task, task.key);
}

TEST_F(TaskManagerTest, IgnoresSnapshot) {
    start_manager(std::make_unique<EchoTaskFactory>());
    auto task = synnax::task::Task{.name = "t", .type = "echo", .snapshot = true};
    ASSERT_NIL(rack.tasks.create(task));

    std::atomic<bool> received = false;
    std::thread reader([&] {
        auto [frame, err] = streamer.read();
        if (err) return;
        for (const auto &j: frame.series->at(0).json_values()) {
            auto parser = x::json::Parser(j);
            auto s = synnax::task::Status::parse(parser);
            if (s.variant != x::status::VARIANT_WARNING && s.details.task == task.key)
                received = true;
        }
    });
    std::this_thread::sleep_for((300 * x::telem::MILLISECOND).chrono());
    streamer.close_send();
    reader.join();
    ASSERT_FALSE(received);
}

TEST_F(TaskManagerTest, ParallelConfig) {
    auto factory = std::make_unique<BlockingTaskFactory>();
    auto *f = factory.get();
    start_manager(std::move(factory));

    auto blocking = synnax::task::Task{.name = "b", .type = "blocking"};
    ASSERT_NIL(rack.tasks.create(blocking));
    EVENTUALLY([&] { return f->started.load(); }, [] { return "not started"; });

    auto echo = synnax::task::Task{.name = "e", .type = "echo"};
    ASSERT_NIL(rack.tasks.create(echo));
    auto s = WAIT_FOR_TASK_STATUS(streamer, echo, [](const synnax::task::Status &s) {
        return s.message == "configured";
    });
    ASSERT_EQ(s.details.task, echo.key);

    f->release();
    WAIT_FOR_TASK_STATUS(streamer, blocking, [](const synnax::task::Status &s) {
        return s.message == "configured";
    });
}

TEST_F(TaskManagerTest, CommandForUnconfigured) {
    start_manager(std::make_unique<EchoTaskFactory>());
    auto cmd_ch = ASSERT_NIL_P(client->channels.retrieve("sy_task_cmd"));
    auto writer = ASSERT_NIL_P(client->telem.open_writer(
        {.channels = {cmd_ch.key}, .start = x::telem::TimeStamp::now()}
    ));

    auto fake_key = synnax::task::create_task_key(rack.key, 99999);
    auto cmd = synnax::task::Command{.task = fake_key, .type = "test", .args = json{}};
    ASSERT_NIL(
        writer.write(x::telem::Frame(cmd_ch.key, x::telem::Series(cmd.to_json())))
    );
    ASSERT_NIL(writer.close());
    std::this_thread::sleep_for((200 * x::telem::MILLISECOND).chrono());

    auto task = synnax::task::Task{.name = "t", .type = "echo"};
    ASSERT_NIL(rack.tasks.create(task));
    WAIT_FOR_TASK_STATUS(streamer, task, [](const synnax::task::Status &s) {
        return s.message == "configured";
    });
}

TEST_F(TaskManagerTest, RapidReconfigure) {
    start_manager(std::make_unique<EchoTaskFactory>());
    auto task = synnax::task::Task{.name = "t", .type = "echo"};
    ASSERT_NIL(rack.tasks.create(task));
    WAIT_FOR_TASK_STATUS(streamer, task, [](const synnax::task::Status &s) {
        return s.message == "configured";
    });

    for (int i = 0; i < 5; i++) {
        task.config = json{{"v", i}};
        ASSERT_NIL(rack.tasks.create(task));
    }
    std::this_thread::sleep_for((500 * x::telem::MILLISECOND).chrono());

    auto cmd_ch = ASSERT_NIL_P(client->channels.retrieve("sy_task_cmd"));
    auto writer = ASSERT_NIL_P(client->telem.open_writer(
        {.channels = {cmd_ch.key}, .start = x::telem::TimeStamp::now()}
    ));
    auto cmd = synnax::task::Command{
        .task = task.key,
        .type = "test",
        .key = "final",
        .args = json{}
    };
    ASSERT_NIL(
        writer.write(x::telem::Frame(cmd_ch.key, x::telem::Series(cmd.to_json())))
    );
    ASSERT_NIL(writer.close());
    WAIT_FOR_TASK_STATUS(streamer, task, [](const synnax::task::Status &s) {
        return s.details.cmd == "final";
    });
}

TEST_F(TaskManagerTest, Timeout) {
    auto factory = std::make_unique<TimeoutTaskFactory>();
    auto *f = factory.get();
    // 500ms timeout, 100ms poll for fast test
    start_manager(
        std::move(factory),
        {.op_timeout = 500 * x::telem::MILLISECOND,
         .poll_interval = 100 * x::telem::MILLISECOND}
    );

    auto task = synnax::task::Task{.name = "t", .type = "timeout"};
    ASSERT_NIL(rack.tasks.create(task));

    auto s = WAIT_FOR_TASK_STATUS(
        streamer,
        task,
        [](const synnax::task::Status &s) {
            return s.variant == x::status::VARIANT_ERROR &&
                   s.message == "operation timed out";
        },
        5 * x::telem::SECOND
    );
    ASSERT_EQ(s.details.task, task.key);

    f->release_all();
}

TEST_F(TaskManagerTest, CommandFIFO) {
    auto factory = std::make_unique<TrackingTaskFactory>();
    auto *f = factory.get();
    start_manager(std::move(factory));

    auto cmd_ch = ASSERT_NIL_P(client->channels.retrieve("sy_task_cmd"));
    auto writer = ASSERT_NIL_P(client->telem.open_writer(
        {.channels = {cmd_ch.key}, .start = x::telem::TimeStamp::now()}
    ));

    auto task = synnax::task::Task{.name = "t", .type = "tracking"};
    ASSERT_NIL(rack.tasks.create(task));
    EVENTUALLY(
        [&] {
            std::lock_guard lock(f->mu);
            return !f->task_states.empty();
        },
        [] { return "task not created"; }
    );

    std::vector<std::string> expected = {"c1", "c2", "c3", "c4", "c5"};
    for (const auto &k: expected) {
        auto cmd = synnax::task::Command{
            .task = task.key,
            .type = "test",
            .key = k,
            .args = json{}
        };
        ASSERT_NIL(
            writer.write(x::telem::Frame(cmd_ch.key, x::telem::Series(cmd.to_json())))
        );
    }
    ASSERT_NIL(writer.close());

    auto state = f->task_states[0];
    ASSERT_EVENTUALLY_GE(state->exec_count.load(), 5);
    std::lock_guard lock(state->cmd_order_mu);
    ASSERT_EQ(state->cmd_order, expected);
}

TEST_F(TaskManagerTest, ReconfigureStopsOld) {
    auto factory = std::make_unique<TrackingTaskFactory>();
    auto *f = factory.get();
    start_manager(std::move(factory));

    auto task = synnax::task::Task{.name = "t", .type = "tracking"};
    ASSERT_NIL(rack.tasks.create(task));

    std::shared_ptr<TrackingTaskState> first_state;
    EVENTUALLY(
        [&] {
            std::lock_guard lock(f->mu);
            if (f->task_states.empty()) return false;
            first_state = f->task_states[0];
            return true;
        },
        [] { return "first not created"; },
    );

    task.config = json{{"v", 2}};
    ASSERT_NIL(rack.tasks.create(task));

    ASSERT_EVENTUALLY_TRUE(first_state->stopped.load());
    ASSERT_TRUE(first_state->stop_will_reconfigure.load());

    EVENTUALLY(
        [&] {
            std::lock_guard lock(f->mu);
            return f->task_states.size() >= 2;
        },
        [] { return "second not created"; }
    );
}

class DestructorTrackingTask final : public task::Task {
public:
    synnax::task::Task sy_task;
    std::atomic<bool> *destroyed;
    std::atomic<bool> stopped{false};

    DestructorTrackingTask(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::task::Task &task,
        std::atomic<bool> *destroyed
    ):
        sy_task(task), destroyed(destroyed) {
        synnax::task::Status status{
            .key = synnax::task::status_key(task),
            .variant = x::status::VARIANT_SUCCESS,
            .message = "configured",
            .details = {.task = task.key}
        };
        ctx->set_status(status);
    }

    ~DestructorTrackingTask() override {
        if (destroyed != nullptr) *destroyed = true;
    }

    std::string name() const override { return "destructor_tracking"; }

    void exec(synnax::task::Command &) override {}

    void stop(bool) override { stopped = true; }
};

class DestructorTrackingFactory final : public task::Factory {
public:
    std::atomic<bool> first_destroyed{false};
    std::atomic<bool> second_destroyed{false};
    std::atomic<int> configure_count{0};

    std::pair<std::unique_ptr<task::Task>, bool> configure_task(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::task::Task &task
    ) override {
        if (task.type != "destructor_tracking") return {nullptr, false};
        int count = configure_count.fetch_add(1);
        std::atomic<bool> *destroyed = (count == 0) ? &first_destroyed
                                                    : &second_destroyed;
        return {std::make_unique<DestructorTrackingTask>(ctx, task, destroyed), true};
    }
};

TEST_F(TaskManagerTest, ReconfigureCallsDestructor) {
    auto factory = std::make_unique<DestructorTrackingFactory>();
    auto *f = factory.get();
    start_manager(std::move(factory));

    auto task = synnax::task::Task(rack.key, "t", "destructor_tracking", "");
    ASSERT_NIL(rack.tasks.create(task));

    WAIT_FOR_TASK_STATUS(streamer, task, [](auto &s) {
        return s.message == "configured";
    });

    ASSERT_EQ(f->configure_count.load(), 1);
    ASSERT_FALSE(f->first_destroyed.load());

    task.config = "{\"v\":2}";
    ASSERT_NIL(rack.tasks.create(task));

    ASSERT_EVENTUALLY_GE(f->configure_count.load(), 2);

    ASSERT_EVENTUALLY_TRUE(f->first_destroyed.load());

    ASSERT_FALSE(f->second_destroyed.load());
}

class ShutdownTest : public testing::Test {
protected:
    std::shared_ptr<synnax::Synnax> client;
    synnax::rack::Rack rack;

    void SetUp() override {
        client = std::make_shared<synnax::Synnax>(new_test_client());
        rack = ASSERT_NIL_P(client->racks.create("shutdown_rack"));
    }
};

TEST_F(ShutdownTest, DuringConfiguration) {
    auto factory = std::make_unique<BlockingTaskFactory>();
    auto *f = factory.get();
    auto manager = std::make_unique<task::Manager>(rack, client, std::move(factory));

    std::promise<void> started;
    std::thread thread([&] { manager->run([&] { started.set_value(); }); });
    started.get_future().wait_for((5 * x::telem::SECOND).chrono());

    auto task = synnax::task::Task{.name = "t", .type = "blocking"};
    ASSERT_NIL(rack.tasks.create(task));
    EVENTUALLY([&] { return f->started.load(); }, [] { return "not started"; });

    manager->stop();
    f->release();

    auto join = std::async(std::launch::async, [&] { thread.join(); });
    ASSERT_EQ(
        join.wait_for((5 * x::telem::SECOND).chrono()),
        std::future_status::ready
    );
}

TEST_F(ShutdownTest, WithPendingOps) {
    auto factory = std::make_unique<BlockingTaskFactory>();
    auto *f = factory.get();
    auto manager = std::make_unique<task::Manager>(rack, client, std::move(factory));

    std::promise<void> started;
    std::thread thread([&] { manager->run([&] { started.set_value(); }); });
    started.get_future().wait_for((5 * x::telem::SECOND).chrono());

    for (int i = 0; i < 3; i++) {
        auto task = synnax::task::Task{
            .name = "t" + std::to_string(i),
            .type = "blocking"
        };
        ASSERT_NIL(rack.tasks.create(task));
    }
    std::this_thread::sleep_for((50 * x::telem::MILLISECOND).chrono());

    manager->stop();
    f->release();

    auto join = std::async(std::launch::async, [&] { thread.join(); });
    ASSERT_EQ(
        join.wait_for((5 * x::telem::SECOND).chrono()),
        std::future_status::ready
    );
}

/// @brief Task that blocks forever on stop() - used to test shutdown timeout.
class BlockingStopTask final : public task::Task {
    std::atomic<bool> &stop_called;
    std::atomic<bool> &release;
    std::condition_variable &cv;
    std::mutex &mu;

public:
    BlockingStopTask(
        std::atomic<bool> &stop_called,
        std::atomic<bool> &release,
        std::condition_variable &cv,
        std::mutex &mu
    ):
        stop_called(stop_called), release(release), cv(cv), mu(mu) {}

    std::string name() const override { return "blocking_stop"; }
    void exec(synnax::task::Command &) override {}

    void stop(bool) override {
        stop_called = true;
        std::unique_lock lock(mu);
        cv.wait(lock, [&] { return release.load(); });
    }
};

class BlockingStopFactory final : public task::Factory {
public:
    std::atomic<bool> stop_called{false};
    std::atomic<bool> release{false};
    std::condition_variable cv;
    std::mutex mu;

    std::pair<std::unique_ptr<task::Task>, bool> configure_task(
        const std::shared_ptr<task::Context> &,
        const synnax::task::Task &task
    ) override {
        if (task.type == "blocking_stop")
            return {
                std::make_unique<BlockingStopTask>(stop_called, release, cv, mu),
                true
            };
        return {nullptr, false};
    }

    void release_all() {
        release = true;
        cv.notify_all();
    }
};

TEST_F(ShutdownTest, TimeoutDetachesStuckWorkers) {
    auto factory = std::make_unique<BlockingStopFactory>();
    auto *f = factory.get();
    // Very short shutdown timeout (500ms) for fast test
    auto manager = std::make_unique<task::Manager>(
        rack,
        client,
        std::move(factory),
        task::ManagerConfig{
            .op_timeout = 60 * x::telem::SECOND,
            .poll_interval = 1 * x::telem::SECOND,
            .shutdown_timeout = 500 * x::telem::MILLISECOND
        }
    );

    std::promise<void> started;
    std::thread thread([&] { manager->run([&] { started.set_value(); }); });
    started.get_future().wait_for((5 * x::telem::SECOND).chrono());

    auto task = synnax::task::Task{.name = "t", .type = "blocking_stop"};
    ASSERT_NIL(rack.tasks.create(task));
    std::this_thread::sleep_for((100 * x::telem::MILLISECOND).chrono());

    manager->stop();

    // Manager should shut down within ~1s even though stop() blocks forever
    auto join = std::async(std::launch::async, [&] { thread.join(); });
    ASSERT_EQ(
        join.wait_for((3 * x::telem::SECOND).chrono()),
        std::future_status::ready
    );

    // Release the blocking stop so the detached thread can exit cleanly
    f->release_all();
    std::this_thread::sleep_for((100 * x::telem::MILLISECOND).chrono());
}

/// @brief Task that takes a fixed time to stop - used to test parallel stopping.
class SlowStopTask final : public task::Task {
    x::telem::TimeSpan stop_duration;
    std::atomic<bool> &stopped;

public:
    SlowStopTask(x::telem::TimeSpan duration, std::atomic<bool> &stopped):
        stop_duration(duration), stopped(stopped) {}

    std::string name() const override { return "slow_stop"; }
    void exec(synnax::task::Command &) override {}

    void stop(bool) override {
        std::this_thread::sleep_for(stop_duration.chrono());
        stopped = true;
    }
};

class SlowStopFactory final : public task::Factory {
public:
    std::vector<std::atomic<bool> *> stopped_flags;
    std::mutex mu;
    x::telem::TimeSpan stop_duration;

    explicit SlowStopFactory(x::telem::TimeSpan duration): stop_duration(duration) {}

    std::pair<std::unique_ptr<task::Task>, bool> configure_task(
        const std::shared_ptr<task::Context> &,
        const synnax::task::Task &task
    ) override {
        if (task.type == "slow_stop") {
            auto flag = new std::atomic<bool>(false);
            std::lock_guard lock(mu);
            stopped_flags.push_back(flag);
            return {std::make_unique<SlowStopTask>(stop_duration, *flag), true};
        }
        return {nullptr, false};
    }

    ~SlowStopFactory() override {
        for (auto *f: stopped_flags)
            delete f;
    }
};

TEST_F(ShutdownTest, ParallelTaskStop) {
    // Each task takes 200ms to stop
    auto factory = std::make_unique<SlowStopFactory>(200 * x::telem::MILLISECOND);
    auto manager = std::make_unique<task::Manager>(rack, client, std::move(factory));

    std::promise<void> started;
    std::thread thread([&] { manager->run([&] { started.set_value(); }); });
    started.get_future().wait_for((5 * x::telem::SECOND).chrono());

    // Create 4 tasks that each take 200ms to stop
    for (int i = 0; i < 4; i++) {
        auto task = synnax::task::Task{
            .name = "t" + std::to_string(i),
            .type = "slow_stop"
        };
        ASSERT_NIL(rack.tasks.create(task));
    }
    std::this_thread::sleep_for((200 * x::telem::MILLISECOND).chrono());

    auto before = x::telem::TimeStamp::now();
    manager->stop();
    thread.join();
    auto elapsed = x::telem::TimeStamp::now() - before;

    // With parallel stopping, 4 tasks × 200ms should take ~200-400ms, not 800ms
    // Allow some overhead but it should definitely be under 700ms
    ASSERT_LT(elapsed.milliseconds(), 700);
}

/// @brief Factory where configure_task blocks forever, simulating a stuck hardware
/// call. This does NOT respond to breaker.stop() or cv.notify_all() - it only unblocks
/// when explicitly released. This tests that stop_workers() properly detaches stuck
/// workers.
class StuckWorkerFactory final : public task::Factory {
public:
    std::atomic<bool> configure_started{false};
    std::atomic<bool> release{false};
    std::condition_variable cv;
    std::mutex mu;

    std::pair<std::unique_ptr<task::Task>, bool> configure_task(
        const std::shared_ptr<task::Context> &,
        const synnax::task::Task &task
    ) override {
        if (task.type == "stuck_worker") {
            configure_started = true;
            std::unique_lock lock(mu);
            cv.wait(lock, [&] { return release.load(); });
            return {nullptr, true};
        }
        return {nullptr, false};
    }

    void release_all() {
        release = true;
        cv.notify_all();
    }
};

/// @brief Regression test for stop_workers() timeout logic.
/// Previously, stop_workers() had two bugs:
/// 1. The polling loop had an immediate `break`, so it only waited 50ms
/// 2. It then called join() which blocks forever if the worker is stuck
/// This test verifies that stuck workers are properly detached after shutdown_timeout.
TEST_F(ShutdownTest, StuckWorkerDetach) {
    auto factory = std::make_unique<StuckWorkerFactory>();
    auto *f = factory.get();
    auto manager = std::make_unique<task::Manager>(
        rack,
        client,
        std::move(factory),
        task::ManagerConfig{
            .op_timeout = 60 * x::telem::SECOND,
            .poll_interval = 1 * x::telem::SECOND,
            .shutdown_timeout = 500 * x::telem::MILLISECOND
        }
    );

    std::promise<void> started;
    std::thread thread([&] { manager->run([&] { started.set_value(); }); });
    started.get_future().wait_for((5 * x::telem::SECOND).chrono());

    auto task = synnax::task::Task{.name = "t", .type = "stuck_worker"};
    ASSERT_NIL(rack.tasks.create(task));

    EVENTUALLY(
        [&] { return f->configure_started.load(); },
        [] { return "configure not started"; }
    );

    auto before = x::telem::TimeStamp::now();
    manager->stop();
    thread.join();
    auto elapsed = x::telem::TimeStamp::now() - before;

    // Should shut down within ~1s (500ms timeout + overhead), not hang forever
    ASSERT_LT(elapsed.milliseconds(), 2000);

    f->release_all();
    std::this_thread::sleep_for((100 * x::telem::MILLISECOND).chrono());
}
}
