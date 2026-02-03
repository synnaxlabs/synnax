// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <functional>
#include <map>
#include <memory>
#include <string>
#include <unordered_set>
#include <variant>
#include <vector>

#include "glog/logging.h"

#include "client/cpp/synnax.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/xjson/xjson.h"

namespace transform {
class Transform {
public:
    virtual ~Transform() = default;

    virtual xerrors::Error transform(telem::Frame &frame) = 0;
};

class Chain final : public Transform {
public:
    void add(const std::shared_ptr<Transform> &transforms) {
        this->transforms.push_back(transforms);
    }

    xerrors::Error transform(telem::Frame &frame) override {
        if (transforms.empty()) return xerrors::NIL;
        for (const auto &t: this->transforms)
            if (const auto err = t->transform(frame)) return err;
        return xerrors::NIL;
    }

private:
    std::vector<std::shared_ptr<Transform>> transforms;
};

/// @brief middleware to tare data written to channels based on the last frame
/// processed at the time of taring. This middleware should be added to the pipeline
/// middleware chain first so that it can tare the data before any other middleware
/// can process it.
class Tare final : public Transform {
    std::unordered_map<synnax::ChannelKey, double> tare_values;
    std::unordered_map<synnax::ChannelKey, synnax::Channel> tare_channels;
    std::unordered_set<synnax::ChannelKey> channels_to_tare;
    bool tare_all;
    std::mutex mutex;

public:
    explicit Tare(const std::vector<synnax::Channel> &channels):
        tare_channels(map_channel_Keys(channels)), tare_all(false) {}

    xerrors::Error tare(json &arg) {
        xjson::Parser parser(arg);
        const auto channels = parser.field<std::vector<synnax::ChannelKey>>(
            "keys",
            std::vector<synnax::ChannelKey>{}
        );
        if (parser.error()) return parser.error();

        std::lock_guard lock(mutex);
        if (channels.empty()) {
            tare_all = true;
            channels_to_tare.clear();
            return xerrors::NIL;
        }

        for (auto &key: channels) {
            if (tare_channels.find(key) == tare_channels.end()) {
                parser.field_err(
                    "keys",
                    "Channel " + std::to_string(key) +
                        " is not a configured channel to tare."
                );
                return parser.error();
            }
            channels_to_tare.insert(key);
        }
        tare_all = false;
        return xerrors::NIL;
    }

    xerrors::Error transform(telem::Frame &frame) override {
        std::lock_guard lock(mutex);
        if (tare_all || !channels_to_tare.empty()) {
            for (const auto &[ch_key, series]: frame)
                if (tare_all || channels_to_tare.find(ch_key) != channels_to_tare.end())
                    tare_values[ch_key] = series.avg<double>();
            tare_all = false;
            channels_to_tare.clear();
        }

        for (const auto &[ch_key, series]: frame) {
            auto tare_it = tare_values.find(ch_key);
            if (tare_it == tare_values.end()) continue;
            series.sub_inplace(tare_it->second);
        }
        return xerrors::NIL;
    }
};

class UnaryLinearScale {
    double slope;
    double offset;
    telem::DataType dt;

public:
    explicit UnaryLinearScale(xjson::Parser &parser, const telem::DataType &dt):
        slope(parser.field<double>("slope")),
        offset(parser.field<double>("offset")),
        dt(dt) {}

    xerrors::Error transform_inplace(const telem::Series &series) const {
        if (this->dt != series.data_type())
            return xerrors::Error(
                xerrors::VALIDATION,
                "series data type " + series.data_type().name() +
                    " does not match scale data type " + this->dt.name()
            );

        // val * slope + offset
        series.multiply_inplace(slope);
        series.add_inplace(offset);

        return xerrors::NIL;
    }
};

class UnaryMapScale {
    double prescaled_min;
    double prescaled_max;
    double scaled_min;
    double scaled_max;
    telem::DataType dt;

public:
    explicit UnaryMapScale(xjson::Parser &parser, const telem::DataType &dt):
        prescaled_min(parser.field<double>("pre_scaled_min")),
        prescaled_max(parser.field<double>("pre_scaled_max")),
        scaled_min(parser.field<double>("scaled_min")),
        scaled_max(parser.field<double>("scaled_max")),
        dt(dt) {}

    xerrors::Error transform_inplace(const telem::Series &series) const {
        if (this->dt != series.data_type())
            return xerrors::Error(
                xerrors::VALIDATION,
                "series data type " + series.data_type().name() +
                    " does not match scale data type " + this->dt.name()
            );

        // (v - prescaled_min) / (prescaled_max - prescaled_min) * (scaled_max -
        // scaled_min) + scaled_min
        series.sub_inplace(prescaled_min);
        series.divide_inplace(prescaled_max - prescaled_min);
        series.multiply_inplace(scaled_max - scaled_min); // * (scaled_max - scaled_min)
        series.add_inplace(scaled_min);

        return xerrors::NIL;
    }
};

class Scale final : public Transform {
    std::map<synnax::ChannelKey, std::variant<UnaryLinearScale, UnaryMapScale>> scales;

public:
    explicit Scale(
        const xjson::Parser &parser,
        const std::unordered_map<synnax::ChannelKey, synnax::Channel> &channels
    ) {
        parser.iter("channels", [this, &channels](xjson::Parser &channel_parser) {
            const auto key = channel_parser.field<synnax::ChannelKey>("channel");
            const auto enabled = channel_parser.field<bool>("enabled", true);
            auto scale_parser = channel_parser.optional_child("scale");
            if (!channel_parser.ok() || !enabled) return;
            const auto ch_t = channels.find(key);
            if (ch_t == channels.end()) {
                channel_parser.field_err(
                    "channel",
                    "Channel " + std::to_string(key) + " is not a configured channel."
                );
                return;
            }
            const auto type = scale_parser.field<std::string>("type");
            const auto dt = ch_t->second.data_type;
            if (type == "linear") {
                UnaryLinearScale linear_scale(scale_parser, dt);
                scales.emplace(key, std::move(linear_scale));
            } else if (type == "map") {
                UnaryMapScale map_scale(scale_parser, dt);
                scales.emplace(key, std::move(map_scale));
            }
        });
    }

    xerrors::Error transform(telem::Frame &frame) override {
        if (frame.empty()) return xerrors::NIL;
        for (const auto [key, series]: frame) {
            auto it = scales.find(key);
            if (it == scales.end()) continue;
            xerrors::Error err = std::visit(
                [&series](const auto &scale) -> xerrors::Error {
                    return scale.transform_inplace(series);
                },
                it->second
            );
            if (err) return err;
        }
        return xerrors::NIL;
    }
};
}
