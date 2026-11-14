// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <memory>
#include <set>
#include <vector>

#include "client/cpp/ethercat/json.gen.h"
#include "client/cpp/synnax.h"
#include "x/cpp/json/json.h"

#include "driver/common/write_task.h"
#include "driver/ethercat/channel/channel.h"
#include "driver/ethercat/engine/engine.h"
#include "driver/ethercat/slaves.h"

namespace driver::ethercat {
/// @brief configuration for EtherCAT write tasks.
struct WriteTaskConfig : ::synnax::ethercat::WriteConfig {
    /// @brief the slaves that the channels name.
    Slaves slaves;
    /// @brief configured output channels.
    std::vector<std::unique_ptr<channel::Output>> outputs;

    WriteTaskConfig(WriteTaskConfig &&other) noexcept:
        ::synnax::ethercat::WriteConfig(std::move(other)),
        slaves(std::move(other.slaves)),
        outputs(std::move(other.outputs)) {}

    WriteTaskConfig(const WriteTaskConfig &) = delete;
    const WriteTaskConfig &operator=(const WriteTaskConfig &) = delete;

    explicit WriteTaskConfig(
        const std::shared_ptr<synnax::Synnax> &client,
        x::json::Parser &cfg
    ):
        ::synnax::ethercat::WriteConfig(::synnax::ethercat::WriteConfig::parse(cfg)) {
        if (cfg.error()) return;
        channel::Claims claims;
        for (std::size_t i = 0; i < this->channels.size(); ++i) {
            const auto &base = ::synnax::ethercat::base(this->channels[i]);
            if (base.disabled) continue;
            const auto path = "channels." + std::to_string(i) + ".";
            const auto *slave = this->slaves.resolve(*client, base.device, cfg, path);
            if (slave == nullptr) continue;
            auto output = channel::make_output(this->channels[i], *slave, cfg, path);
            if (output == nullptr) continue;
            claims.claim(output->entry, *slave, cfg, path);
            this->outputs.push_back(std::move(output));
        }

        if (cfg.error()) return;

        if (this->outputs.empty()) {
            cfg.field_err("channels", "task must have at least one enabled channel");
            return;
        }

        channel::sort_by_position(this->outputs);
        std::vector<synnax::channel::Key> state_keys;
        for (const auto &ch: this->outputs)
            if (ch->state_key != 0) state_keys.push_back(ch->state_key);

        if (!state_keys.empty()) {
            auto [state_chs, err] = client->channels.retrieve(state_keys);
            if (err) {
                cfg.field_err("channels", err.data);
                return;
            }
            std::unordered_map<synnax::channel::Key, synnax::channel::Channel> by_key;
            for (auto &ch: state_chs)
                by_key.emplace(ch.key, std::move(ch));
            for (const auto &out: this->outputs)
                if (out->state_key != 0) out->state = by_key.at(out->state_key);
        }
    }

    static std::pair<WriteTaskConfig, x::errors::Error> parse(
        const std::shared_ptr<synnax::Synnax> &client,
        const synnax::task::Task &task
    ) {
        auto parser = x::json::Parser(task.config);
        WriteTaskConfig cfg(client, parser);
        return {std::move(cfg), parser.error()};
    }

    /// @brief returns the command and state channel of each output.
    [[nodiscard]] std::vector<common::Sink::Output> output_channels() const {
        std::vector<common::Sink::Output> outs;
        outs.reserve(this->outputs.size());
        for (const auto &out: this->outputs)
            outs.push_back({out->command_key, out->state});
        return outs;
    }
};

/// @brief sink implementation for EtherCAT write tasks.
class WriteTaskSink final : public common::Sink {
    WriteTaskConfig cfg;
    std::shared_ptr<engine::Engine> engine;
    std::unique_ptr<engine::Engine::Writer> writer;

public:
    explicit WriteTaskSink(std::shared_ptr<engine::Engine> eng, WriteTaskConfig cfg):
        Sink(cfg.state_rate, cfg.output_channels(), cfg.data_saving_disabled),
        cfg(std::move(cfg)),
        engine(std::move(eng)) {}

    x::errors::Error start(const std::string &, x::breaker::Breaker &cancel) override {
        const auto [slaves, discover_err] = this->engine->discover();
        if (discover_err) return discover_err;
        if (auto err = this->cfg.slaves.validate(slaves)) return err;

        std::vector<engine::Engine::Writer::Entry> entries;
        entries.reserve(this->cfg.outputs.size());
        for (const auto &ch: this->cfg.outputs)
            entries.push_back({.entry = ch->entry, .stop_value = ch->stop_value});
        auto [wtr, err] = this->engine
                              ->open_writer(entries, this->cfg.execution_rate, cancel);
        if (err) return err;
        this->writer = std::move(wtr);
        return x::errors::NIL;
    }

    x::errors::Error stop() override {
        this->writer.reset();
        return x::errors::NIL;
    }

    x::errors::Error write(x::telem::Frame &frame) override {
        const auto tx = this->writer->open_tx();
        for (size_t i = 0; i < this->cfg.outputs.size(); ++i) {
            const auto &ch = this->cfg.outputs[i];
            if (!frame.contains(ch->command_key)) continue;
            if (auto err = tx.write(i, frame.at(ch->command_key, -1)); err) return err;
        }
        this->set_state(frame);
        return x::errors::NIL;
    }
};
}
