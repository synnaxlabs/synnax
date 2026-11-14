// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <array>
#include <bit>
#include <cmath>
#include <memory>
#include <set>
#include <string>
#include <vector>

#include "client/cpp/ethercat/json.gen.h"
#include "client/cpp/synnax.h"
#include "x/cpp/json/json.h"

#include "driver/common/read_task.h"
#include "driver/common/sample_clock.h"
#include "driver/ethercat/channel/channel.h"
#include "driver/ethercat/engine/engine.h"
#include "driver/ethercat/errors/errors.h"
#include "driver/ethercat/slaves.h"
#include "driver/ethercat/telem/telem.h"
#include "driver/transform/transform.h"

namespace driver::ethercat {
/// @brief configuration for EtherCAT read tasks.
struct ReadTaskConfig : common::BaseReadTaskConfig {
    /// @brief the slaves that the channels name.
    Slaves slaves;
    /// @brief index channel keys for timestamp generation.
    /// Dynamically populated by querying the core.
    std::set<synnax::channel::Key> indexes;
    /// @brief configured input channels.
    std::vector<std::unique_ptr<channel::Input>> channels;
    /// @brief number of samples per channel per batch.
    size_t samples_per_chan = 0;
    /// @brief scales the raw samples of scaled channels into engineering units.
    transform::Scale scales;
    /// @brief the values the task writes to each slave when it starts.
    engine::Settings settings;

    ReadTaskConfig(ReadTaskConfig &&other) noexcept:
        BaseReadTaskConfig(std::move(other)),
        slaves(std::move(other.slaves)),
        indexes(std::move(other.indexes)),
        channels(std::move(other.channels)),
        samples_per_chan(other.samples_per_chan),
        scales(std::move(other.scales)),
        settings(std::move(other.settings)) {}

    ReadTaskConfig(const ReadTaskConfig &) = delete;
    const ReadTaskConfig &operator=(const ReadTaskConfig &) = delete;

    explicit ReadTaskConfig(
        const std::shared_ptr<synnax::Synnax> &client,
        x::json::Parser &cfg
    ):
        BaseReadTaskConfig(cfg) {
        if (this->sample_rate > x::telem::Rate(0) &&
            this->stream_rate > x::telem::Rate(0)) {
            const auto ratio = telem::whole_ratio(this->sample_rate, this->stream_rate);
            if (!ratio) {
                cfg.field_err(
                    "stream_rate",
                    "sample_rate must be divisible by stream_rate"
                );
                return;
            }
            this->samples_per_chan = *ratio;
        }

        channel::Claims claims;
        cfg.iter("channels", [&](x::json::Parser &ch) {
            // The channels share one error list, so an earlier channel's error shows
            // in ch.error().
            const auto errors_before = ch.errors->size();
            const auto parsed = ::synnax::ethercat::parse_read_channel(ch);
            if (ch.errors->size() != errors_before) return;
            const auto &base = ::synnax::ethercat::base(parsed);
            if (base.disabled) return;
            const auto *slave = this->slaves.resolve(*client, base.device, ch);
            if (slave == nullptr) return;
            auto input = channel::make_input(parsed, *slave, ch);
            if (input == nullptr) return;
            transform::validate_scale(ch, "scale", input->scale);
            claims.claim(input->entry, *slave, ch);
            this->channels.push_back(std::move(input));
        });

        if (cfg.has("startup_parameters"))
            cfg.iter("startup_parameters", [&](x::json::Parser &s) {
                const auto *slave = this->slaves.resolve(
                    *client,
                    s.field<std::string>("device"),
                    s
                );
                if (slave == nullptr) return;
                auto &params = this->settings[slave->position];
                s.iter("parameters", [&params](x::json::Parser &param) {
                    params.push_back(slave::StartupParameter::parse(param));
                });
            });

        if (cfg.error()) return;

        if (this->channels.empty()) {
            cfg.field_err("channels", "task must have at least one enabled channel");
            return;
        }

        channel::sort_by_position(this->channels);

        std::vector<synnax::channel::Key> keys;
        keys.reserve(this->channels.size());
        for (const auto &ch: this->channels)
            keys.push_back(ch->synnax_key);

        auto [synnax_channels, ch_err] = client->channels.retrieve(keys);
        if (ch_err) {
            cfg.field_err("channels", ch_err.data);
            return;
        }

        static const std::vector FLOAT_TYPES = {
            x::telem::FLOAT32_T,
            x::telem::FLOAT64_T
        };
        for (size_t i = 0; i < this->channels.size(); i++) {
            auto &ch = *this->channels[i];
            ch.ch = synnax_channels[i];
            if (synnax_channels[i].index != 0)
                this->indexes.insert(synnax_channels[i].index);
            if (!ch.scaled() && !ch.calibration) continue;
            if (!ch.ch.data_type.matches(FLOAT_TYPES)) {
                cfg.field_err(
                    "channels",
                    "channel " + ch.ch.name + " is " + ch.ch.data_type.name() +
                        ", but a " + (ch.scaled() ? "scaled" : "calibrated") +
                        " channel must be float32 or float64. Pick a float channel."
                );
                continue;
            }
            if (ch.scaled()) this->scales.add(ch.ch.key, ch.ch.data_type, ch.scale);
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
            .mode = common::data_saving_writer_mode(data_saving_disabled),
        };
    }
};

/// @brief source implementation for EtherCAT read tasks.
class ReadTaskSource final : public common::Source {
    ReadTaskConfig cfg;
    std::shared_ptr<engine::Engine> engine;
    std::unique_ptr<engine::Engine::Reader> reader;
    /// @brief the calibration of each calibrated channel, from the gain and offset that
    /// its slave held at start().
    transform::Scale calibrations;
    /// @brief the bus cycle time of each sample in the latest read.
    x::telem::Series times;

    /// @brief reads the gain and offset of each calibrated channel from its slave on
    /// the running bus.
    x::errors::Error calibrate() {
        for (const auto &ch: this->cfg.channels) {
            if (!ch->calibration) continue;
            const auto &c = *ch->calibration;
            std::array<float, 2> values{};
            const std::array sub_indexes{c.gain_sub_index, c.offset_sub_index};
            for (size_t i = 0; i < values.size(); i++) {
                auto [bits, err] = this->engine->read(
                    ch->entry.slave_position,
                    c.index,
                    sub_indexes[i]
                );
                if (err) {
                    err.data = errors::sentence(
                        "Failed to read the calibration of channel " + ch->ch.name +
                        ": " + err.data
                    );
                    return err;
                }
                values[i] = std::bit_cast<float>(bits);
            }
            const auto [slope, offset] = c.line(values[0], values[1]);
            this->calibrations.add(
                ch->ch.key,
                transform::UnaryLinearScale(slope, offset, ch->ch.data_type)
            );
        }
        return x::errors::NIL;
    }

public:
    explicit ReadTaskSource(std::shared_ptr<engine::Engine> eng, ReadTaskConfig cfg):
        cfg(std::move(cfg)),
        engine(std::move(eng)),
        times(x::telem::TIMESTAMP_T, this->cfg.samples_per_chan) {}

    x::errors::Error start(const std::string &, x::breaker::Breaker &cancel) override {
        const auto [slaves, discover_err] = this->engine->discover();
        if (discover_err) return discover_err;
        if (auto err = this->cfg.slaves.validate(slaves)) return err;

        std::vector<pdo::Entry> entries;
        entries.reserve(this->cfg.channels.size());
        for (const auto &ch: this->cfg.channels)
            entries.push_back(ch->entry);

        auto [rdr, err] = this->engine->open_reader(
            entries,
            this->cfg.sample_rate,
            this->cfg.samples_per_chan,
            cancel,
            this->cfg.settings
        );
        if (err) return err;
        this->reader = std::move(rdr);
        this->calibrations = transform::Scale();
        if (auto calibration_err = this->calibrate()) {
            this->reader.reset();
            return calibration_err;
        }
        return x::errors::NIL;
    }

    x::errors::Error stop() override {
        this->reader.reset();
        return x::errors::NIL;
    }

    common::ReadResult
    read(x::breaker::Breaker &breaker, x::telem::Frame &fr) override {
        common::ReadResult res;
        common::initialize_frame(
            fr,
            this->cfg.channels,
            this->cfg.indexes,
            this->cfg.samples_per_chan
        );
        for (auto &ser: *fr.series)
            ser.clear();
        this->times.clear();
        if (res.error = this->reader->read(breaker, fr, this->times); res.error)
            return res;
        if (!breaker.running()) {
            fr.clear();
            return res;
        }
        if (auto description = this->reader->warning(); !description.empty())
            res.warning = {
                .message = "Dropped samples",
                .description = std::move(description),
            };
        if (this->times.empty()) {
            fr.clear();
            return res;
        }
        if (res.error = this->calibrations.transform(fr); res.error) return res;
        if (res.error = this->cfg.scales.transform(fr); res.error) return res;
        for (size_t i = this->cfg.channels.size(); i < fr.series->size(); ++i)
            fr.series->at(i).write(this->times);
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
