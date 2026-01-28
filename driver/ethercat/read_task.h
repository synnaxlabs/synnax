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
#include <cstring>
#include <memory>
#include <set>
#include <vector>

#include "client/cpp/synnax.h"
#include "x/cpp/xjson/xjson.h"

#include "driver/ethercat/channels.h"
#include "driver/ethercat/cyclic_engine.h"
#include "driver/task/common/read_task.h"

namespace ethercat {
/// Configuration for an EtherCAT read task.
struct ReadTaskConfig : common::BaseReadTaskConfig {
    /// The key of the device (network interface) to read from.
    std::string device_key;

    /// Index keys of all data channels in the task.
    std::set<synnax::ChannelKey> indexes;

    /// Input channels to read from.
    std::vector<channel::Input> channels;

    /// Number of samples per channel to read on each read() call.
    size_t samples_per_chan;

    ReadTaskConfig(ReadTaskConfig &&other) noexcept:
        BaseReadTaskConfig(std::move(other)),
        device_key(std::move(other.device_key)),
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
        cfg.iter("channels", [this](xjson::Parser &ch) {
            const auto type = ch.field<std::string>("type");
            if (type == "input") this->channels.emplace_back(ch);
        });

        channel::sort_by_position(this->channels);

        std::vector<synnax::ChannelKey> keys;
        keys.reserve(this->channels.size());
        for (const auto &ch: this->channels)
            keys.push_back(ch.synnax_key);

        auto [synnax_channels, err] = client->channels.retrieve(keys);
        if (err) {
            cfg.field_err("channels", err.message());
            return;
        }

        for (size_t i = 0; i < this->channels.size(); i++) {
            this->channels[i].bind_remote_info(synnax_channels[i]);
            if (synnax_channels[i].index != 0)
                this->indexes.insert(synnax_channels[i].index);
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
        result.reserve(this->channels.size());
        for (const auto &ch: this->channels)
            result.push_back(ch.ch);
        return result;
    }

    /// Returns the configuration for opening a Synnax writer for the task.
    [[nodiscard]] synnax::WriterConfig writer_config() const {
        std::vector<synnax::ChannelKey> keys;
        keys.reserve(this->channels.size() + this->indexes.size());
        for (const auto &ch: this->channels)
            keys.push_back(ch.ch.key);
        for (const auto &idx: this->indexes)
            keys.push_back(idx);
        return synnax::WriterConfig{
            .channels = keys,
            .mode = common::data_saving_writer_mode(this->data_saving),
        };
    }
};

/// Implements common::Source to read from EtherCAT slaves via the CyclicEngine.
class ReadTaskSource final : public common::Source {
    ReadTaskConfig config_;
    std::shared_ptr<CyclicEngine> engine_;
    std::vector<uint8_t> input_buffer_;
    std::atomic<bool> stopped_;

public:
    /// Constructs a ReadTaskSource with the given engine and configuration.
    /// @param engine The CyclicEngine to use for cyclic PDO exchange.
    /// @param cfg The task configuration.
    explicit ReadTaskSource(std::shared_ptr<CyclicEngine> engine, ReadTaskConfig cfg):
        config_(std::move(cfg)), engine_(std::move(engine)), stopped_(false) {}

    xerrors::Error start() override {
        // Register PDOs and store registration indices
        std::vector<size_t> registration_indices;
        registration_indices.reserve(config_.channels.size());

        for (const auto &ch : config_.channels) {
            auto [reg_index, err] = engine_->register_input_pdo(ch.to_pdo_entry(true));
            if (err) return err;
            registration_indices.push_back(reg_index);
        }

        // Activate the engine (this resolves actual offsets)
        if (auto err = engine_->add_task(); err) return err;

        // Now get the actual offsets
        for (size_t i = 0; i < config_.channels.size(); ++i) {
            config_.channels[i].buffer_offset =
                engine_->get_actual_input_offset(registration_indices[i]);
        }

        return xerrors::NIL;
    }

    xerrors::Error stop() override {
        stopped_ = true;
        engine_->remove_task();
        return xerrors::NIL;
    }

    common::ReadResult read(breaker::Breaker &breaker, telem::Frame &fr) override {
        common::ReadResult res;
        const size_t n_channels = config_.channels.size();
        const size_t n_samples = config_.samples_per_chan;
        const size_t total_count = n_channels + config_.indexes.size();

        if (fr.size() != total_count) {
            fr.reserve(total_count);
            for (const auto &ch: config_.channels)
                fr.emplace(ch.ch.key, telem::Series(ch.ch.data_type, n_samples));
            for (const auto &idx: config_.indexes)
                fr.emplace(idx, telem::Series(telem::TIMESTAMP_T, n_samples));
        }

        for (auto &ser: *fr.series)
            ser.clear();

        for (size_t i = 0; i < n_samples; ++i) {
            const auto start = telem::TimeStamp::now();

            res.error = engine_->wait_for_inputs(input_buffer_, stopped_);
            if (res.error) return res;

            const auto end = telem::TimeStamp::now();
            const auto midpoint = telem::TimeStamp::midpoint(start, end);

            size_t series_idx = 0;
            for (const auto &ch: config_.channels) {
                auto &s = fr.series->at(series_idx++);
                if (ch.buffer_offset + ch.byte_length() <= input_buffer_.size())
                    s.write_casted(
                        input_buffer_.data() + ch.buffer_offset,
                        1,
                        ch.ch.data_type
                    );
            }

            for (size_t j = 0; j < config_.indexes.size(); ++j)
                fr.series->at(series_idx++).write(midpoint);
        }

        return res;
    }

    [[nodiscard]] synnax::WriterConfig writer_config() const override {
        return config_.writer_config();
    }

    [[nodiscard]] std::vector<synnax::Channel> channels() const override {
        return config_.data_channels();
    }
};
}
