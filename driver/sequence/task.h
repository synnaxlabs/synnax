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


class Task final : public task::Task {
    /// @brief cfg is the configuration for the task.
    TaskConfig cfg;
    /// @brief ctx is the task execution context for communicating with the Synnax cluster
    /// and updating the task state.
    std::shared_ptr<task::Context> ctx;
    /// @brief breaker is used to manage the lifecycle of the sequence.
    breaker::Breaker breaker;
    /// @brief thread is the thread that will execute the sequence.
    std::thread thread;
    std::vector<synnax::Channel> write_channels;
    std::vector<synnax::Channel> read_channels;

public:
    Task(
        const std::shared_ptr<task::Context> &ctx,
        TaskConfig cfg,
        const std::vector<synnax::Channel> &read_channels,
        const std::vector<synnax::Channel> &write_channels
    ): cfg(std::move(cfg)),
       ctx(ctx),
       read_channels(read_channels),
       write_channels(write_channels),
       breaker(breaker::Config{
           .name = "sequence",
           .base_interval = 1 * SECOND,
           .max_retries = 20,
           .scale = 1.2,
       }) {
    }


    void run() {
        std::unordered_map<synnax::ChannelKey, synnax::Channel> read_channel_map;
        for (const auto &ch: read_channels) read_channel_map[ch.key] = ch;

        std::unordered_map<std::string, synnax::Channel> write_channel_map;
        for (const auto &ch: write_channels) write_channel_map[ch.name] = ch;

        // Step 1 - instantiate the JSON source
        auto json_source = std::make_shared<JSONSource>(cfg.globals);

        // Step 2 - instantiate the channel source and streamer config.
        auto ch_source = std::make_shared<ChannelSource>(read_channel_map);
        synnax::StreamerConfig streamer_cfg{.channels = cfg.read,};

        auto breaker_config = breaker::Config{
            .name = "sequence",
            .base_interval = 1 * SECOND,
            .max_retries = 20,
            .scale = 1.2,
        };


        /// Step 3 - open the control pipeline;
        auto pipe = pipeline::Control(
            this->ctx->client,
            streamer_cfg,
            ch_source,
            breaker_config
        );

        /// Step 4 - open a synnax writer
        synnax::WriterConfig writer_cfg{
            .channels = cfg.write,
            .start = synnax::TimeStamp::now(),
        };

        auto [writer, err] = ctx->client->telem.openWriter(writer_cfg);
        if (err) {
            LOG(ERROR) << "[sequence] failed to open writer: " << err;
            return;
        }

        auto writer_ptr = std::make_unique<synnax::Writer>(std::move(writer));
        auto sink = std::make_shared<SynnaxSink>(std::move(writer_ptr));

        auto ops = std::make_shared<ChannelSetOperator>(sink, write_channel_map);


        auto [seq , seq_err] = sequence::Sequence::create(
            ops,
            ch_source,
            cfg.script
        );
        if (seq_err) {
            LOG(ERROR) << "[sequence] failed to create sequence: " << seq_err;
            return;
        }

        pipe.start();

        loop::Timer timer(this->cfg.rate);
        while (this->breaker.running()) {
            seq->next();
            timer.wait(breaker);
        }
        pipe.stop();
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
    }

    void stop(const std::string &key) {
        if (!this->breaker.running()) return;
        this->breaker.stop();
        this->breaker.reset();
        if (this->thread.joinable()) this->thread.join();
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


        return std::make_unique<Task>(
            ctx,
            cfg,
            read_channels,
            write_channels
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
