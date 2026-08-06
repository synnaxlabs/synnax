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
#include "driver/ethercat/topology/topology.h"

namespace driver::ethercat {
/// @brief configuration for EtherCAT write tasks.
struct WriteTaskConfig : ::synnax::ethercat::WriteConfig {
    /// @brief network interface name for the EtherCAT master.
    /// Dynamically populated from device properties.
    std::string interface_name;
    /// @brief configured output channels.
    std::vector<std::unique_ptr<channel::Output>> outputs;
    /// @brief state feedback channels.
    /// Dynamically populated by querying the core.
    std::vector<synnax::channel::Channel> state_channels;
    /// @brief index channel keys for state timestamps.
    /// Dynamically populated by querying the core.
    std::set<synnax::channel::Key> state_indexes;
    /// @brief cached device properties for topology validation.
    std::unordered_map<std::string, slave::Properties> device_cache;

    WriteTaskConfig(WriteTaskConfig &&other) noexcept:
        ::synnax::ethercat::WriteConfig(std::move(other)),
        interface_name(std::move(other.interface_name)),
        outputs(std::move(other.outputs)),
        state_channels(std::move(other.state_channels)),
        state_indexes(std::move(other.state_indexes)),
        device_cache(std::move(other.device_cache)) {}

    WriteTaskConfig(const WriteTaskConfig &) = delete;
    const WriteTaskConfig &operator=(const WriteTaskConfig &) = delete;

    explicit WriteTaskConfig(
        const std::shared_ptr<synnax::Synnax> &client,
        x::json::Parser &cfg
    ):
        ::synnax::ethercat::WriteConfig(::synnax::ethercat::WriteConfig::parse(cfg)) {
        std::unordered_map<std::string, slave::Properties> slave_cache;
        std::string first_network;

        for (std::size_t i = 0; i < this->channels.size(); ++i) {
            const auto path = "channels." + std::to_string(i) + ".";
            const auto &slave_key = channel::base(this->channels[i]).device;
            if (!slave_cache.contains(slave_key)) {
                auto [slave_dev, slave_err] = client->devices.retrieve(slave_key);
                if (slave_err) {
                    cfg.field_err(path + "device", slave_err.message());
                    continue;
                }
                auto props_parser = x::json::Parser(slave_dev.properties);
                auto props = slave::Properties::parse(props_parser);
                if (props_parser.error()) {
                    cfg.field_err(path + "device", props_parser.error().message());
                    continue;
                }
                slave_cache.emplace(slave_key, std::move(props));
                const auto &network = slave_cache.at(slave_key).network;
                if (first_network.empty())
                    first_network = network;
                else if (network != first_network) {
                    cfg.field_err(
                        path + "device",
                        "all slaves must be on the same network"
                    );
                    continue;
                }
            }

            const auto &slave = slave_cache.at(slave_key);
            auto output = channel::make_output(this->channels[i], slave, cfg, path);
            if (output->enabled) this->outputs.push_back(std::move(output));
        }

        if (cfg.error()) return;

        this->interface_name = first_network;
        this->device_cache = std::move(slave_cache);

        channel::sort_by_position(this->outputs);
        std::vector<synnax::channel::Key> state_keys;
        for (const auto &ch: this->outputs)
            if (ch->state_key != 0) state_keys.push_back(ch->state_key);

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

    static std::pair<WriteTaskConfig, x::errors::Error> parse(
        const std::shared_ptr<synnax::Synnax> &client,
        const synnax::task::Task &task
    ) {
        auto parser = x::json::Parser(task.config);
        WriteTaskConfig cfg(client, parser);
        return {std::move(cfg), parser.error()};
    }

    [[nodiscard]] std::vector<synnax::channel::Key> cmd_keys() const {
        std::vector<synnax::channel::Key> keys;
        keys.reserve(this->outputs.size());
        for (const auto &ch: this->outputs)
            keys.push_back(ch->command_key);
        return keys;
    }
};

/// @brief sink implementation for EtherCAT write tasks.
class WriteTaskSink final : public common::Sink {
    WriteTaskConfig cfg;
    std::shared_ptr<engine::Engine> engine;
    std::unique_ptr<engine::Engine::Writer> writer;

public:
    explicit WriteTaskSink(std::shared_ptr<engine::Engine> eng, WriteTaskConfig cfg):
        Sink(
            cfg.state_rate,
            cfg.state_indexes,
            cfg.state_channels,
            cfg.cmd_keys(),
            !cfg.data_saving_disabled
        ),
        cfg(std::move(cfg)),
        engine(std::move(eng)) {}

    x::errors::Error start() override {
        if (auto err = this->engine->ensure_initialized(); err) return err;
        if (auto err = topology::validate(
                slave::discovered_properties(this->engine->slaves()),
                this->cfg.device_cache
            ))
            return err;

        std::vector<pdo::Entry> entries;
        entries.reserve(this->cfg.outputs.size());
        for (const auto &ch: this->cfg.outputs)
            entries.push_back(*ch);
        auto [wtr, err] = this->engine->open_writer(
            std::move(entries),
            this->cfg.execution_rate
        );
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
            tx.write(i, frame.at(ch->command_key, -1));
        }
        this->set_state(frame);
        return x::errors::NIL;
    }
};
}
