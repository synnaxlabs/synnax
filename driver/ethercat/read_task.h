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
#include "driver/ethercat/engine/engine.h"
#include "driver/task/common/read_task.h"
#include "driver/task/common/sample_clock.h"

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
    ReadTaskConfig cfg;
    std::shared_ptr<engine::Engine> engine;
    std::unique_ptr<engine::Engine::Reader> reader;

public:
    explicit ReadTaskSource(std::shared_ptr<engine::Engine> eng, ReadTaskConfig cfg):
        cfg(std::move(cfg)), engine(std::move(eng)) {}

    xerrors::Error start() override {
        std::vector<PDOEntry> entries;
        entries.reserve(this->cfg.channels.size());
        for (const auto &ch: this->cfg.channels)
            entries.push_back(ch->to_pdo_entry(true));

        auto [rdr, err] = this->engine->open_reader(std::move(entries));
        if (err) return err;
        this->reader = std::move(rdr);
        return xerrors::NIL;
    }

    xerrors::Error stop() override {
        this->reader.reset();
        return xerrors::NIL;
    }

    common::ReadResult read(breaker::Breaker &breaker, telem::Frame &fr) override {
        common::ReadResult res;
        const size_t n_channels = this->cfg.channels.size();
        const size_t n_samples = this->cfg.samples_per_chan;
        common::initialize_frame(fr, this->cfg.channels, this->cfg.indexes, n_samples);
        for (auto &ser: *fr.series)
            ser.clear();
        const auto start = telem::TimeStamp::now();
        for (size_t i = 0; i < n_samples; ++i)
            if (res.error = this->reader->read(breaker, fr); res.error) return res;
        const auto end = telem::TimeStamp::now();
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

    [[nodiscard]] synnax::WriterConfig writer_config() const override {
        return cfg.writer_config();
    }

    [[nodiscard]] std::vector<synnax::Channel> channels() const override {
        return cfg.data_channels();
    }
};
}
