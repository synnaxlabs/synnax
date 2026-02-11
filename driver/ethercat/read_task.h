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

#include "client/cpp/synnax.h"
#include "x/cpp/json/json.h"

#include "driver/common/read_task.h"
#include "driver/common/sample_clock.h"
#include "driver/ethercat/channel/channel.h"
#include "driver/ethercat/engine/engine.h"
#include "driver/ethercat/topology/topology.h"

namespace driver::ethercat {
/// @brief configuration for EtherCAT read tasks.
struct ReadTaskConfig : common::BaseReadTaskConfig {
    /// @brief network interface name for the EtherCAT master.
    /// Dynamically populated from device properties.
    std::string interface_name;
    /// @brief index channel keys for timestamp generation.
    /// Dynamically populated by querying the core.
    std::set<synnax::channel::Key> indexes;
    /// @brief configured input channels.
    std::vector<std::unique_ptr<channel::Input>> channels;
    /// @brief number of samples per channel per batch.
    size_t samples_per_chan;
    /// @brief cached device properties for topology validation.
    std::unordered_map<std::string, slave::Properties> device_cache;

    ReadTaskConfig(ReadTaskConfig &&other) noexcept:
        BaseReadTaskConfig(std::move(other)),
        interface_name(std::move(other.interface_name)),
        indexes(std::move(other.indexes)),
        channels(std::move(other.channels)),
        samples_per_chan(other.samples_per_chan),
        device_cache(std::move(other.device_cache)) {}

    ReadTaskConfig(const ReadTaskConfig &) = delete;
    const ReadTaskConfig &operator=(const ReadTaskConfig &) = delete;

    explicit ReadTaskConfig(
        const std::shared_ptr<synnax::Synnax> &client,
        x::json::Parser &cfg
    ):
        BaseReadTaskConfig(cfg), samples_per_chan(sample_rate / stream_rate) {
        const auto sample_rate_int = static_cast<size_t>(sample_rate.hz());
        const auto stream_rate_int = static_cast<size_t>(stream_rate.hz());
        if (sample_rate_int % stream_rate_int != 0) {
            cfg.field_err(
                "stream_rate",
                "sample_rate must be divisible by stream_rate"
            );
            return;
        }

        std::unordered_map<std::string, slave::Properties> slave_cache;
        std::string first_network;

        cfg.iter("channels", [&](x::json::Parser &ch) {
            auto slave_key = ch.field<std::string>("device");
            if (ch.error()) return;
            if (!slave_cache.contains(slave_key)) {
                auto [slave_dev, slave_err] = client->devices.retrieve(slave_key);
                if (slave_err) {
                    ch.field_err("device", slave_err.message());
                    return;
                }
                auto props_parser = x::json::Parser(slave_dev.properties);
                slave_cache.emplace(slave_key, slave::Properties::parse(props_parser));
                if (props_parser.error()) {
                    ch.field_err("device", props_parser.error().message());
                    return;
                }
                const auto &slave = slave_cache.at(slave_key);
                if (first_network.empty())
                    first_network = slave.network;
                else if (slave.network != first_network) {
                    ch.field_err("device", "all slaves must be on the same network");
                    return;
                }
            }
            const auto &slave = slave_cache.at(slave_key);
            auto channel_ptr = channel::parse_input(ch, slave);
            if (channel_ptr && channel_ptr->enabled)
                this->channels.push_back(std::move(channel_ptr));
        });

        if (cfg.error()) return;

        if (this->channels.empty()) {
            cfg.field_err("channels", "task must have at least one enabled channel");
            return;
        }

        this->interface_name = first_network;
        this->device_cache = std::move(slave_cache);

        channel::sort_by_position(this->channels);

        std::vector<synnax::channel::Key> keys;
        keys.reserve(this->channels.size());
        for (const auto &ch: this->channels)
            keys.push_back(ch->synnax_key);

        auto [synnax_channels, ch_err] = client->channels.retrieve(keys);
        if (ch_err) {
            cfg.field_err("channels", ch_err.message());
            return;
        }

        for (size_t i = 0; i < this->channels.size(); i++) {
            this->channels[i]->bind_remote_info(synnax_channels[i]);
            if (synnax_channels[i].index != 0)
                this->indexes.insert(synnax_channels[i].index);
        }
    }

    static std::pair<ReadTaskConfig, x::errors::Error> parse(
        const std::shared_ptr<synnax::Synnax> &client,
        const synnax::task::Task &task
    ) {
        auto parser = x::json::Parser(task.config);
        ReadTaskConfig cfg(client, parser);
        return {std::move(cfg), parser.error()};
    }

    [[nodiscard]] std::vector<synnax::channel::Channel> data_channels() const {
        std::vector<synnax::channel::Channel> result;
        result.reserve(channels.size());
        for (const auto &ch: channels)
            result.push_back(ch->ch);
        return result;
    }

    [[nodiscard]] synnax::framer::WriterConfig writer_config() const {
        std::vector<synnax::channel::Key> keys;
        keys.reserve(channels.size() + indexes.size());
        for (const auto &ch: channels)
            keys.push_back(ch->ch.key);
        for (const auto &idx: indexes)
            keys.push_back(idx);
        return synnax::framer::WriterConfig{
            .channels = keys,
            .mode = common::data_saving_writer_mode(data_saving),
        };
    }
};

/// @brief source implementation for EtherCAT read tasks.
class ReadTaskSource final : public common::Source {
    ReadTaskConfig cfg;
    std::shared_ptr<engine::Engine> engine;
    std::unique_ptr<engine::Engine::Reader> reader;

public:
    explicit ReadTaskSource(std::shared_ptr<engine::Engine> eng, ReadTaskConfig cfg):
        cfg(std::move(cfg)), engine(std::move(eng)) {}

    x::errors::Error start() override {
        if (auto err = this->engine->ensure_initialized(); err) return err;
        if (auto err = topology::validate(
                slave::discovered_properties(this->engine->slaves()),
                this->cfg.device_cache
            );
            err)
            return err;

        std::vector<pdo::Entry> entries;
        entries.reserve(this->cfg.channels.size());
        for (const auto &ch: this->cfg.channels)
            entries.push_back(*ch);

        auto [rdr, err] = this->engine->open_reader(
            std::move(entries),
            this->cfg.sample_rate
        );
        if (err) return err;
        this->reader = std::move(rdr);
        return x::errors::NIL;
    }

    x::errors::Error stop() override {
        this->reader.reset();
        return x::errors::NIL;
    }

    common::ReadResult
    read(x::breaker::Breaker &breaker, x::telem::Frame &fr) override {
        common::ReadResult res;
        const size_t n_channels = this->cfg.channels.size();
        const size_t n_samples = this->cfg.samples_per_chan;
        common::initialize_frame(fr, this->cfg.channels, this->cfg.indexes, n_samples);
        for (auto &ser: *fr.series)
            ser.clear();

        const auto engine_rate = this->engine->cycle_rate();
        const size_t decimation = static_cast<size_t>(
            engine_rate / this->cfg.sample_rate
        );
        const size_t epochs_per_batch = n_samples * decimation;

        const auto start = x::telem::TimeStamp::now();
        for (size_t epoch = 0; epoch < epochs_per_batch; ++epoch) {
            if (epoch % decimation == 0) {
                if (res.error = this->reader->read(breaker, fr); res.error) return res;
            } else if (res.error = this->reader->wait(breaker); res.error)
                return res;
            if (!breaker.running()) {
                fr.clear();
                return res;
            }
        }
        const auto end = x::telem::TimeStamp::now();
        common::generate_index_data(
            fr,
            this->cfg.indexes,
            start,
            end,
            n_samples,
            n_channels
        );
        return res;
    }

    [[nodiscard]] synnax::framer::WriterConfig writer_config() const override {
        return cfg.writer_config();
    }

    [[nodiscard]] std::vector<synnax::channel::Channel> channels() const override {
        return cfg.data_channels();
    }
};
}
