#pragma once

/// external.
#include <utility>

#include "nlohmann/json.hpp"

/// internal.
#include "json_source.h"
#include "driver/sequence/channel_set_operator.h"
#include "driver/sequence/channel_source.h"
#include "driver/loop/loop.h"
#include "driver/pipeline/acquisition.h"
#include "driver/pipeline/control.h"
#include "driver/sequence/sequence.h"
#include "driver/task/task.h"

using json = nlohmann::json;

namespace sequence {
const std::string INTEGRATION_NAME = "sequence";

/// @brief TaskConfig is the configuration for creating a sequence task.
struct TaskConfig {
    /// @brief rate is the rate at which the script loop will execute.
    synnax::Rate rate;
    /// @brief script is the lua scrip that will be executed ihn the fixed rate loop.
    std::string script;
    /// @brief read is the list of channels that the task will need to read from in
    /// real-time.
    std::vector<synnax::ChannelKey> read;
    /// @brief write_to is the channels that the task will need write access to for
    /// control.
    std::vector<synnax::ChannelKey> write;

    /// @brief globals is a JSON object whose keys are global variables that will be
    /// available within the Lua script.
    json globals{};

    explicit TaskConfig(config::Parser &parser):
        // this comment keeps the formatter happy
        rate(synnax::Rate(parser.required<float>("rate"))),
        script(parser.required<std::string>("script")),
        read(parser.required_vector<synnax::ChannelKey>("read")),
        write(parser.required_vector<synnax::ChannelKey>("write")),
        globals(parser.optional<json>("globals", json::object())) {
    };
};


/// @brief an implementation of a driver task used for configuring and running
/// automated sequences.
class Task final : public task::Task {
    /// @brief cfg is the configuration for the task.
    const TaskConfig cfg;
    /// @brief task is the task configuration.
    const synnax::Task task;
    /// @brief the list of channels that the task will write to.
    breaker::Breaker breaker;
    /// @brief thread is the thread that will execute the sequence.
    std::thread thread;
    pipeline::Control pipe;
    /// @brief ctx is the task execution context for communicating with the Synnax cluster
    /// and updating the task state.
    std::shared_ptr<task::Context> ctx;
    std::unique_ptr<sequence::Sequence> seq;
    std::shared_ptr<SynnaxSink> sink;

public:
    Task(
        const std::shared_ptr<task::Context> &ctx,
        synnax::Task task,
        TaskConfig cfg,
        std::unique_ptr<sequence::Sequence> seq,
        synnax::StreamerConfig streamer_config,
        std::shared_ptr<ChannelSource> ch_source,
        breaker::Config breaker_config,
        std::shared_ptr<SynnaxSink> sink
    ): cfg(std::move(cfg)),
       ctx(ctx),
       task(std::move(task)),
       pipe(
           ctx->client,
           streamer_config,
           ch_source,
           breaker_config
       ),
       breaker(breaker::Config{
           .name = "sequence",
           .base_interval = 1 * SECOND,
           .max_retries = 20,
           .scale = 1.2,
       }),
       seq(std::move(seq)),
       sink(sink) {
    }


    void run() {
        this->pipe.start();
        loop::Timer timer(this->cfg.rate);
        while (this->breaker.running()) {
            if (const auto next_err = this->seq->next()) {
                ctx->set_state({
                    .task = task.key,
                    .variant = "error",
                    .details = {
                        {"running", false},
                        {"message", next_err.message()}
                    }
                });
            }
            timer.wait(breaker);
        }
        this->pipe.stop();
        if (const auto sink_close_err = this->sink->close()) {
            ctx->set_state({
                .task = task.key,
                .variant = "error",
                .details = {
                    {"running", false},
                    {"message", sink_close_err.message()}
                }
            });
        }
    }

    void stop() override { this->stop(""); };

    void exec(task::Command &cmd) override {
        if (cmd.type == "start") this->start(cmd.key);
        else if (cmd.type == "stop") this->stop(cmd.key);
    }

    void start(const std::string &key) {
        if (this->breaker.running()) return;
        this->breaker.reset();
        this->breaker.start();
        this->thread = std::thread([this] { this->run(); });
        ctx->set_state({
            .task = this->task.key,
            .key = key,
            .variant = "success",
            .details = json{
                {"running", true},
                {"message", "Sequence started successfully"}
            }
        });
    }

    void stop(const std::string &key) {
        if (!this->breaker.running()) return;
        this->breaker.stop();
        this->breaker.reset();
        if (this->thread.joinable()) this->thread.join();
        ctx->set_state({
            .task = this->task.key,
            .key = key,
            .variant = "success",
            .details = json{
                {"running", false},
                {"message", "Sequence stopped"}
            }
        });
    }

    static std::unique_ptr<task::Task> configure(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    ) {
        LOG(INFO) << "[sequence] configuring task " << task.name;
        auto parser = config::Parser(task.config);
        TaskConfig cfg(parser);
        if (!parser.ok()) {
            LOG(ERROR) << "[sequence] failed to parse task configuration: " << parser.
                    error();
            ctx->set_state({
                .task = task.key,
                .variant = "error",
                .details = parser.error_json()
            });
            return nullptr;
        }

        // Step 1 - fetch read and write channels.
        auto [read_channels, r_err] = ctx->client->channels.retrieve(cfg.read);
        if (r_err) {
            LOG(ERROR) << "[sequence] failed to retrieve read channels: " << r_err;
            return nullptr;
        }


        auto [write_channels, w_err] = ctx->client->channels.retrieve(cfg.write);
        if (w_err) {
            LOG(ERROR) << "[sequence] failed to retrieve write channels: " << w_err;
            return nullptr;
        }

        auto json_source = std::make_shared<JSONSource>(cfg.globals);
        auto ch_source = std::make_shared<ChannelSource>(read_channels);
        synnax::StreamerConfig streamer_cfg{.channels = cfg.read,};
        auto breaker_config = breaker::Config{
            .name = "sequence",
            .base_interval = 1 * SECOND,
            .max_retries = 20,
            .scale = 1.2,
        };

        synnax::ControlSubject subject{
            .name = task.name,
            .key = std::to_string(task.key)
        };

        synnax::WriterConfig writer_cfg{
            .channels = cfg.write,
            .start = synnax::TimeStamp::now(),
            .authorities = {200},
            .subject = subject,
        };

        auto sink = std::make_shared<SynnaxSink>(ctx->client, writer_cfg);
        auto ops = std::make_shared<ChannelSetOperator>(sink, write_channels);
        auto [seq, seq_err] = sequence::Sequence::create(
            ops,
            ch_source,
            cfg.script
        );
        if (seq_err) {
            ctx->set_state({
                .task = task.key,
                .variant = "error",
                .details = {
                    {"running", false},
                    {"message", seq_err.message()}
                }
            });
            return nullptr;
        }

        ctx->set_state({
            .task = task.key,
            .variant = "success",
            .details = json{
                {"running", false},
                {"message", "Sequence configured successfully"}
            }
        });

        return std::make_unique<Task>(
            ctx,
            task,
            cfg,
            std::move(seq),
            streamer_cfg,
            ch_source,
            breaker_config,
            sink
        );
    }
};

class Factory final : public task::Factory {
public:
    Factory() = default;

    std::pair<std::unique_ptr<task::Task>, bool> configure_task(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    ) override {
        if (task.type != "sequence") return {nullptr, false};
        return {sequence::Task::configure(ctx, task), true};
    }

    std::vector<std::pair<synnax::Task, std::unique_ptr<task::Task> > >
    configure_initial_tasks(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Rack &rack
    ) override {
        return {};
    }
};
}
