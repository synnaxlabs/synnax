// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <atomic>
#include <chrono>
#include <cstring>
#include <memory>
#include <set>
#include <thread>
#include <unordered_map>
#include <vector>

#include "client/cpp/synnax.h"
#include "x/cpp/xjson/xjson.h"

#include "driver/ethercat/channels.h"
#include "driver/ethercat/cyclic_engine.h"
#include "driver/ethercat/device.h"
#include "driver/ethercat/errors/errors.h"
#include "driver/task/common/read_task.h"

namespace ethercat {
/// Configuration for an EtherCAT read task.
struct ReadTaskConfig : common::BaseReadTaskConfig {
    /// The key of the network device in Synnax.
    std::string device_key;
    /// Interface name resolved from network device properties.
    std::string interface_name;
    /// Index keys of all data channels in the task.
    std::set<synnax::ChannelKey> indexes;
    /// Polymorphic input channels.
    std::vector<std::unique_ptr<channel::Input>> channels;
    /// Number of samples per channel to read on each read() call.
    size_t samples_per_chan;

    ReadTaskConfig(ReadTaskConfig &&other) noexcept:
        BaseReadTaskConfig(std::move(other)),
        device_key(std::move(other.device_key)),
        interface_name(std::move(other.interface_name)),
        indexes(std::move(other.indexes)),
        channels(std::move(other.channels)),
        samples_per_chan(other.samples_per_chan) {}

    ReadTaskConfig(const ReadTaskConfig &) = delete;
    const ReadTaskConfig &operator=(const ReadTaskConfig &) = delete;

    explicit ReadTaskConfig(
        const std::shared_ptr<synnax::Synnax> &client,
        xjson::Parser &cfg
    ):
        BaseReadTaskConfig(cfg),
        device_key(cfg.field<std::string>("device")),
        samples_per_chan(sample_rate / stream_rate) {
        auto [network_dev, net_err] = client->devices.retrieve(device_key);
        if (net_err) {
            cfg.field_err("device", net_err.message());
            return;
        }

        auto net_parser = xjson::Parser(network_dev.properties);
        device::NetworkProperties net_props(net_parser);
        if (net_parser.error()) {
            cfg.field_err("device", net_parser.error().message());
            return;
        }
        interface_name = net_props.interface;

        std::unordered_map<std::string, device::SlaveProperties> slave_cache;

        cfg.iter("channels", [&](xjson::Parser &ch) {
            auto slave_key = ch.field<std::string>("device");
            if (ch.error()) return;

            if (slave_cache.find(slave_key) == slave_cache.end()) {
                auto [slave_dev, slave_err] = client->devices.retrieve(slave_key);
                if (slave_err) {
                    ch.field_err("device", slave_err.message());
                    return;
                }
                auto props_parser = xjson::Parser(slave_dev.properties);
                slave_cache.emplace(slave_key, device::SlaveProperties(props_parser));
                if (props_parser.error()) {
                    ch.field_err("device", props_parser.error().message());
                    return;
                }
            }

            const auto &slave = slave_cache.at(slave_key);
            auto channel_ptr = channel::parse_input(ch, slave);
            if (channel_ptr && channel_ptr->enabled)
                channels.push_back(std::move(channel_ptr));
        });

        if (cfg.error()) return;

        if (channels.empty()) {
            cfg.field_err("channels", "task must have at least one enabled channel");
            return;
        }

        channel::sort_by_position(channels);

        std::vector<synnax::ChannelKey> keys;
        keys.reserve(channels.size());
        for (const auto &ch: channels)
            keys.push_back(ch->synnax_key);

        auto [synnax_channels, ch_err] = client->channels.retrieve(keys);
        if (ch_err) {
            cfg.field_err("channels", ch_err.message());
            return;
        }

        for (size_t i = 0; i < channels.size(); i++) {
            channels[i]->bind_remote_info(synnax_channels[i]);
            if (synnax_channels[i].index != 0) indexes.insert(synnax_channels[i].index);
        }
    }

    /// Parses the configuration for the task from its JSON representation.
    /// @param client The Synnax client to use to retrieve channel information.
    /// @param task The task to parse.
    /// @returns A pair containing the parsed configuration and any error that occurred.
    static std::pair<ReadTaskConfig, xerrors::Error>
    parse(const std::shared_ptr<synnax::Synnax> &client, const synnax::Task &task) {
        auto parser = xjson::Parser(task.config);
        ReadTaskConfig cfg(client, parser);
        return {std::move(cfg), parser.error()};
    }

    /// Returns all Synnax channels that the task will write to, excluding indexes.
    [[nodiscard]] std::vector<synnax::Channel> data_channels() const {
        std::vector<synnax::Channel> result;
        result.reserve(channels.size());
        for (const auto &ch: channels)
            result.push_back(ch->ch);
        return result;
    }

    /// Returns the configuration for opening a Synnax writer for the task.
    [[nodiscard]] synnax::WriterConfig writer_config() const {
        std::vector<synnax::ChannelKey> keys;
        keys.reserve(channels.size() + indexes.size());
        for (const auto &ch: channels)
            keys.push_back(ch->ch.key);
        for (const auto &idx: indexes)
            keys.push_back(idx);
        return synnax::WriterConfig{
            .channels = keys,
            .mode = common::data_saving_writer_mode(data_saving),
        };
    }
};

/// Implements common::Source to read from EtherCAT slaves via the CyclicEngine.
class ReadTaskSource final : public common::Source {
    ReadTaskConfig config;
    std::shared_ptr<CyclicEngine> engine;
    std::vector<uint8_t> input_buffer;
    std::atomic<bool> stopped;
    std::vector<PDOHandle> pdo_handles;

public:
    /// Constructs a ReadTaskSource with the given engine and configuration.
    /// @param engine The CyclicEngine to use for cyclic PDO exchange.
    /// @param cfg The task configuration.
    explicit ReadTaskSource(std::shared_ptr<CyclicEngine> engine, ReadTaskConfig cfg):
        config(std::move(cfg)), engine(std::move(engine)), stopped(false) {}

    xerrors::Error start() override {
        auto slaves = this->engine->slaves();
        std::unordered_map<uint32_t, uint16_t> serial_to_position;
        for (const auto &slave: slaves)
            serial_to_position[slave.serial] = slave.position;

        this->pdo_handles.clear();
        this->pdo_handles.reserve(this->config.channels.size());

        for (auto &ch: this->config.channels) {
            auto it = serial_to_position.find(ch->slave_serial);
            if (it == serial_to_position.end())
                return xerrors::Error(
                    SLAVE_STATE_ERROR,
                    "slave with serial " + std::to_string(ch->slave_serial) +
                        " not found on bus"
                );
            ch->slave_position = it->second;

            auto [handle, err] = this->engine->register_input_pdo(
                ch->to_pdo_entry(true)
            );
            if (err) return err;
            this->pdo_handles.push_back(handle);
        }

        if (auto err = this->engine->add_task(); err) return err;

        for (size_t i = 0; i < this->config.channels.size(); ++i)
            this->config.channels[i]->buffer_offset = this->engine
                                                          ->get_actual_input_offset(
                                                              this->pdo_handles[i].index
                                                          );

        return xerrors::NIL;
    }

    xerrors::Error stop() override {
        this->stopped = true;
        this->engine->remove_task();
        return xerrors::NIL;
    }

    common::ReadResult read(breaker::Breaker &breaker, telem::Frame &fr) override {
        common::ReadResult res;
        const size_t n_channels = this->config.channels.size();
        const size_t n_samples = this->config.samples_per_chan;
        const size_t total_count = n_channels + this->config.indexes.size();

        if (fr.size() != total_count) {
            fr.reserve(total_count);
            for (const auto &ch: this->config.channels)
                fr.emplace(ch->ch.key, telem::Series(ch->ch.data_type, n_samples));
            for (const auto &idx: this->config.indexes)
                fr.emplace(idx, telem::Series(telem::TIMESTAMP_T, n_samples));
        }

        for (auto &ser: *fr.series)
            ser.clear();

        for (size_t i = 0; i < n_samples; ++i) {
            const auto start = telem::TimeStamp::now();

            while (breaker.running()) {
                res.error = this->engine->wait_for_inputs(
                    this->input_buffer,
                    this->stopped
                );
                if (res.error.matches(ENGINE_RESTARTING)) {
                    std::this_thread::sleep_for(std::chrono::milliseconds(10));
                    continue;
                }
                break;
            }
            if (res.error) return res;

            const auto end = telem::TimeStamp::now();
            const auto midpoint = telem::TimeStamp::midpoint(start, end);

            size_t series_idx = 0;
            for (const auto &ch: this->config.channels) {
                auto &s = fr.series->at(series_idx++);
                if (ch->buffer_offset + ch->byte_length() <= this->input_buffer.size())
                    s.write_casted(
                        this->input_buffer.data() + ch->buffer_offset,
                        1,
                        ch->ch.data_type
                    );
            }

            for (size_t j = 0; j < this->config.indexes.size(); ++j)
                fr.series->at(series_idx++).write(midpoint);
        }

        return res;
    }

    [[nodiscard]] synnax::WriterConfig writer_config() const override {
        return this->config.writer_config();
    }

    [[nodiscard]] std::vector<synnax::Channel> channels() const override {
        return this->config.data_channels();
    }
};
}
