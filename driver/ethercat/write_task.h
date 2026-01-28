// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstring>
#include <memory>
#include <set>
#include <vector>

#include "client/cpp/synnax.h"
#include "x/cpp/xjson/xjson.h"

#include "driver/ethercat/channels.h"
#include "driver/ethercat/cyclic_engine.h"
#include "driver/task/common/write_task.h"

namespace ethercat {
/// Configuration for an EtherCAT write task.
struct WriteTaskConfig : common::BaseWriteTaskConfig {
    /// Output channels to write to.
    std::vector<channel::Output> channels;

    /// State channels for feedback.
    std::vector<synnax::Channel> state_channels;

    /// Index keys for state channels.
    std::set<synnax::ChannelKey> state_indexes;

    /// State update rate.
    telem::Rate state_rate;

    WriteTaskConfig(WriteTaskConfig &&other) noexcept:
        BaseWriteTaskConfig(std::move(other)),
        channels(std::move(other.channels)),
        state_channels(std::move(other.state_channels)),
        state_indexes(std::move(other.state_indexes)),
        state_rate(other.state_rate) {}

    WriteTaskConfig(const WriteTaskConfig &) = delete;
    const WriteTaskConfig &operator=(const WriteTaskConfig &) = delete;

    explicit WriteTaskConfig(
        const std::shared_ptr<synnax::Synnax> &client,
        xjson::Parser &cfg
    ):
        BaseWriteTaskConfig(cfg),
        state_rate(telem::Rate(cfg.field<float>("state_rate", 1.0f))) {
        cfg.iter("channels", [this](xjson::Parser &ch) {
            const auto type = ch.field<std::string>("type");
            if (type == "output") this->channels.emplace_back(ch);
        });

        channel::sort_by_position(this->channels);

        std::vector<synnax::ChannelKey> state_keys;
        for (const auto &ch: this->channels)
            if (ch.state_key != 0) state_keys.push_back(ch.state_key);

        if (!state_keys.empty()) {
            auto [state_chs, err] = client->channels.retrieve(state_keys);
            if (err) {
                cfg.field_err("channels", err.message());
                return;
            }
            this->state_channels = std::move(state_chs);
            for (const auto &ch: this->state_channels)
                if (ch.index != 0) this->state_indexes.insert(ch.index);
        }
    }

    /// Parses the configuration for the task from its JSON representation.
    /// @param client The Synnax client to use to retrieve channel information.
    /// @param task The task to parse.
    /// @returns A pair containing the parsed configuration and any error that occurred.
    static std::pair<WriteTaskConfig, xerrors::Error>
    parse(const std::shared_ptr<synnax::Synnax> &client, const synnax::Task &task) {
        auto parser = xjson::Parser(task.config);
        WriteTaskConfig cfg(client, parser);
        return {std::move(cfg), parser.error()};
    }

    /// Returns the command channel keys.
    [[nodiscard]] std::vector<synnax::ChannelKey> cmd_keys() const {
        std::vector<synnax::ChannelKey> keys;
        keys.reserve(this->channels.size());
        for (const auto &ch: this->channels)
            keys.push_back(ch.command_key);
        return keys;
    }
};

/// Implements common::Sink to write to EtherCAT slaves via the CyclicEngine.
class WriteTaskSink final : public common::Sink {
    WriteTaskConfig config_;
    std::shared_ptr<CyclicEngine> engine_;

public:
    /// Constructs a WriteTaskSink with the given engine and configuration.
    /// @param engine The CyclicEngine to use for cyclic PDO exchange.
    /// @param cfg The task configuration.
    explicit WriteTaskSink(std::shared_ptr<CyclicEngine> engine, WriteTaskConfig cfg):
        Sink(
            cfg.state_rate,
            cfg.state_indexes,
            cfg.state_channels,
            cfg.cmd_keys(),
            cfg.data_saving
        ),
        config_(std::move(cfg)),
        engine_(std::move(engine)) {}

    xerrors::Error start() override {
        // Register PDOs and store registration indices
        std::vector<size_t> registration_indices;
        registration_indices.reserve(config_.channels.size());

        for (const auto &ch : config_.channels) {
            auto [reg_index, err] = engine_->register_output_pdo(ch.to_pdo_entry(false));
            if (err) return err;
            registration_indices.push_back(reg_index);
        }

        // Activate the engine (this resolves actual offsets)
        if (auto err = engine_->add_task(); err) return err;

        // Now get the actual offsets
        for (size_t i = 0; i < config_.channels.size(); ++i) {
            config_.channels[i].buffer_offset =
                engine_->get_actual_output_offset(registration_indices[i]);
        }

        return xerrors::NIL;
    }

    xerrors::Error stop() override {
        engine_->remove_task();
        return xerrors::NIL;
    }

    xerrors::Error write(telem::Frame &frame) override {
        for (const auto &ch: config_.channels) {
            if (!frame.contains(ch.command_key)) continue;

            const telem::SampleValue value = frame.at(ch.command_key, 0);
            const void *data_ptr = telem::cast_to_void_ptr(value);
            const size_t byte_len = ch.byte_length();
            engine_->write_output(ch.buffer_offset, data_ptr, byte_len);
        }

        this->set_state(frame);
        return xerrors::NIL;
    }
};
}
