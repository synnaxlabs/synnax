// Copyright 2025 Synnax Labs, Inc.

//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <set>

#include "driver/visa/channels.h"
#include "driver/visa/device/device.h"
#include "driver/visa/util/parse.h"
#include "driver/task/common/read_task.h"
#include "driver/task/common/sample_clock.h"

namespace visa {

/// @brief Configuration for a VISA read task.
struct ReadTaskConfig : common::BaseReadTaskConfig {
    /// @brief The total number of data channels in the task.
    size_t data_channel_count;
    /// @brief The key of the device to read from.
    std::string device_key;
    /// @brief The indexes of all data channels in the task.
    std::set<synnax::ChannelKey> indexes;
    /// @brief The list of input channels to read.
    std::vector<channel::InputChannel> channels;
    /// @brief The connection configuration for the device.
    device::ConnectionConfig conn;
    /// @brief The number of samples per channel to read on each read() call.
    std::size_t samples_per_chan;

    /// @brief Default constructor for testing.
    ReadTaskConfig():
        BaseReadTaskConfig(),
        data_channel_count(0),
        device_key(""),
        indexes({}),
        channels({}),
        conn(),
        samples_per_chan(1) {}

    ReadTaskConfig(ReadTaskConfig &&other) noexcept:
        BaseReadTaskConfig(std::move(other)),
        data_channel_count(other.data_channel_count),
        device_key(std::move(other.device_key)),
        indexes(std::move(other.indexes)),
        channels(std::move(other.channels)),
        conn(std::move(other.conn)),
        samples_per_chan(other.samples_per_chan) {}

    ReadTaskConfig(const ReadTaskConfig &) = delete;

    const ReadTaskConfig &operator=(const ReadTaskConfig &) = delete;

    explicit ReadTaskConfig(
        const std::shared_ptr<synnax::Synnax> &client,
        xjson::Parser &cfg
    ):
        BaseReadTaskConfig(cfg),
        data_channel_count(0),
        device_key(cfg.required<std::string>("device")),
        samples_per_chan(sample_rate / stream_rate) {
        auto [dev, dev_err] = client->hardware.retrieve_device(this->device_key);
        if (dev_err) {
            cfg.field_err("device", dev_err.message());
            return;
        }

        auto conn_parser = xjson::Parser(dev.properties);
        this->conn = device::ConnectionConfig(conn_parser.child("connection"));
        if (conn_parser.error()) {
            cfg.field_err("device", conn_parser.error().message());
            return;
        }

        cfg.iter("channels", [&, this](xjson::Parser &ch) {
            this->channels.emplace_back(ch);
            if (this->channels.back().enabled)
                this->data_channel_count++;
        });

        // Retrieve Synnax channels
        std::vector<synnax::ChannelKey> keys;
        keys.reserve(channels.size());
        for (const auto &ch: channels)
            keys.push_back(ch.synnax_key);

        auto [synnax_channels, err] = client->channels.retrieve(keys);
        if (err) {
            cfg.field_err("channels", err.message());
            return;
        }

        for (size_t i = 0; i < channels.size(); i++)
            channels[i].ch = synnax_channels[i];

        // Collect unique index channels
        for (const auto &ch: synnax_channels)
            if (ch.index != 0)
                this->indexes.insert(ch.index);
    }

    /// @brief parses the configuration for the task from its JSON representation.
    static std::pair<ReadTaskConfig, xerrors::Error>
    parse(const std::shared_ptr<synnax::Synnax> &client, const synnax::Task &task) {
        auto parser = xjson::Parser(task.config);
        return {ReadTaskConfig(client, parser), parser.error()};
    }

    /// @brief all Synnax channels that the task will write to, excluding indexes.
    [[nodiscard]] std::vector<synnax::Channel> data_channels() const {
        std::vector<synnax::Channel> result;
        result.reserve(this->data_channel_count);
        for (const auto &ch: this->channels)
            if (ch.enabled)
                result.push_back(ch.ch);
        return result;
    }

    /// @brief configuration for opening a Synnax writer for the task.
    [[nodiscard]] synnax::WriterConfig writer_config() const {
        std::vector<synnax::ChannelKey> keys;
        const auto data_channels = this->data_channels();
        keys.reserve(data_channels.size() + this->indexes.size());
        for (const auto &ch: data_channels)
            keys.push_back(ch.key);
        for (const auto &idx: this->indexes)
            keys.push_back(idx);
        return synnax::WriterConfig{
            .channels = keys,
            .start = telem::TimeStamp::now(),
            .mode = synnax::data_saving_writer_mode(this->data_saving),
        };
    }
};

/// @brief Implements common::Source to read from a VISA device.
class ReadTaskSource final : public common::Source {
    /// @brief The configuration for the task.
    const ReadTaskConfig config;
    /// @brief The VISA session to read from.
    std::shared_ptr<device::Session> session;
    /// @brief The sample clock to regulate the read rate.
    common::SoftwareTimedSampleClock sample_clock;

public:
    /// @brief Constructor accepting any SessionImpl type (for testing with mocks).
    template<typename API_T>
    explicit ReadTaskSource(
        const std::shared_ptr<device::SessionImpl<API_T>> &sess,
        ReadTaskConfig cfg
    ):
        config(std::move(cfg)),
        session(std::reinterpret_pointer_cast<device::Session>(sess)),
        sample_clock(this->config.sample_rate) {}

    common::ReadResult read(breaker::Breaker &breaker, synnax::Frame &fr) override {
        common::ReadResult res;
        const auto n_channels = this->config.data_channel_count;
        const auto n_samples = this->config.samples_per_chan;
        const auto total_channel_count = n_channels + this->config.indexes.size();

        // Initialize frame if needed
        if (fr.size() != total_channel_count) {
            fr.reserve(total_channel_count);
            for (const auto &ch: this->config.data_channels())
                fr.emplace(ch.key, telem::Series(ch.data_type, n_samples));
            for (const auto &idx: this->config.indexes)
                fr.emplace(idx, telem::Series(telem::TIMESTAMP_T, n_samples));
        }

        // Clear existing data
        for (auto &ser: *fr.series)
            ser.clear();

        // Collect samples
        for (size_t i = 0; i < n_samples; ++i) {
            const auto start = this->sample_clock.wait(breaker);

            // Read each enabled channel
            size_t offset = 0;
            for (const auto &ch: this->config.channels) {
                if (!ch.enabled) continue;

                // Send SCPI query
                char response[4096];
                if (res.error = session->query(
                        ch.scpi_command.c_str(),
                        response,
                        sizeof(response)
                    ); res.error)
                    return res;

                // Parse response
                auto [value, parse_err] = util::parse_response(response, ch);
                if (res.error = parse_err; res.error) return res;

                // Write to frame
                auto &series = fr.series->at(offset);
                series.write(value);
                offset++;
            }

            const auto end = this->sample_clock.end();

            // Add timestamps to index channels
            const auto timestamp = telem::TimeStamp(end - (end - start) / 2);
            for (size_t j = offset; j < this->config.indexes.size() + offset; ++j)
                fr.series->at(j).write(timestamp);
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
