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
#include <memory>
#include <optional>
#include <set>
#include <thread>
#include <vector>

#include "client/cpp/synnax.h"
#include "x/cpp/xjson/xjson.h"

#include "driver/ethercat/channel/channel.h"
#include "driver/ethercat/device/device.h"
#include "driver/ethercat/errors/errors.h"
#include "driver/ethercat/loop/loop.h"
#include "driver/task/common/read_task.h"

namespace ethercat {
struct ReadTaskConfig : common::BaseReadTaskConfig {
    std::string device_key;
    std::string interface_name;
    std::set<synnax::ChannelKey> indexes;
    std::vector<std::unique_ptr<channel::Input>> channels;
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

    static std::pair<ReadTaskConfig, xerrors::Error>
    parse(const std::shared_ptr<synnax::Synnax> &client, const synnax::Task &task) {
        auto parser = xjson::Parser(task.config);
        ReadTaskConfig cfg(client, parser);
        return {std::move(cfg), parser.error()};
    }

    [[nodiscard]] std::vector<synnax::Channel> data_channels() const {
        std::vector<synnax::Channel> result;
        result.reserve(channels.size());
        for (const auto &ch: channels)
            result.push_back(ch->ch);
        return result;
    }

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

class ReadTaskSource final : public common::Source {
    ReadTaskConfig config;
    std::shared_ptr<Loop> loop;
    std::optional<Loop::Reader> reader;
    std::vector<uint8_t> input_buffer;
    std::atomic<bool> stopped;

public:
    explicit ReadTaskSource(std::shared_ptr<Loop> loop, ReadTaskConfig cfg):
        config(std::move(cfg)), loop(std::move(loop)), stopped(false) {}

    xerrors::Error start() override {
        std::vector<PDOEntry> entries;
        entries.reserve(this->config.channels.size());
        for (const auto &ch: this->config.channels)
            entries.push_back(ch->to_pdo_entry(true));

        auto [rdr, err] = this->loop->open_reader(std::move(entries));
        if (err) return err;

        this->reader.emplace(std::move(rdr));
        return xerrors::NIL;
    }

    xerrors::Error stop() override {
        stopped = true;
        reader.reset();
        return xerrors::NIL;
    }

    common::ReadResult read(breaker::Breaker &breaker, telem::Frame &fr) override {
        common::ReadResult res;
        const size_t n_channels = config.channels.size();
        const size_t n_samples = config.samples_per_chan;
        const size_t total_count = n_channels + config.indexes.size();

        if (fr.size() != total_count) {
            fr.reserve(total_count);
            for (const auto &ch: config.channels)
                fr.emplace(ch->ch.key, telem::Series(ch->ch.data_type, n_samples));
            for (const auto &idx: config.indexes)
                fr.emplace(idx, telem::Series(telem::TIMESTAMP_T, n_samples));
        }

        for (auto &ser: *fr.series)
            ser.clear();

        for (size_t i = 0; i < n_samples; ++i) {
            const auto start = telem::TimeStamp::now();

            while (breaker.running()) {
                res.error = reader->read(input_buffer, stopped);
                if (res.error.matches(ENGINE_RESTARTING)) {
                    std::this_thread::sleep_for(std::chrono::milliseconds(10));
                    continue;
                }
                break;
            }
            if (res.error) return res;

            const auto end = telem::TimeStamp::now();
            const auto midpoint = telem::TimeStamp::midpoint(start, end);

            // PDO data is laid out contiguously in registration order
            size_t series_idx = 0;
            size_t offset = 0;
            for (const auto &ch: config.channels) {
                auto &s = fr.series->at(series_idx++);
                const size_t len = ch->byte_length();
                if (offset + len <= input_buffer.size()) {
                    s.write_casted(input_buffer.data() + offset, 1, ch->ch.data_type);
                }
                offset += len;
            }

            for (size_t j = 0; j < config.indexes.size(); ++j)
                fr.series->at(series_idx++).write(midpoint);
        }

        return res;
    }

    [[nodiscard]] synnax::WriterConfig writer_config() const override {
        return config.writer_config();
    }

    [[nodiscard]] std::vector<synnax::Channel> channels() const override {
        return config.data_channels();
    }
};
}
