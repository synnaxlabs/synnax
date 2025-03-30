// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// std
#include <memory>
#include <functional>
#include <vector>
#include <map>
#include <string>
#include <variant>
#include <thread>

/// external
#include "glog/logging.h"

/// module
#include "client/cpp/synnax.h"
#include "x/cpp/xjson/xjson.h"
#include "x/cpp/telem/telem.h"

namespace transform {
class Transform {
public:
    virtual ~Transform() = default;

    virtual xerrors::Error transform(synnax::Frame &frame) = 0;
};

class Chain final : public Transform {
public:
    void add(const std::shared_ptr<Transform> &transforms) {
        this->transforms.push_back(transforms);
    }

    xerrors::Error transform(synnax::Frame &frame) override {
        if (transforms.empty()) return xerrors::NIL;
        for (const auto &t: this->transforms)
            if (const auto err = t->transform(frame))
                return err;
        return xerrors::NIL;
    }

private:
    std::vector<std::shared_ptr<Transform> > transforms;
};

/// @brief middleware to tare data written to channels based on the last frame
/// processed at the time of taring. This middleware should be added to the pipeline
/// middleware chain first so that it can tare the data before any other middleware
/// can process it.
class Tare final : public Transform {
    std::map<synnax::ChannelKey, double> tare_values;
    std::map<synnax::ChannelKey, double> last_raw_value;
    std::unordered_map<synnax::ChannelKey, synnax::Channel> tare_channels;
    std::mutex mutex;

public:
    explicit Tare(const std::vector<synnax::Channel> &channels): tare_channels(
        synnax::map_channel_Keys(channels)) {
        for (auto &[key, ch]: this->tare_channels)
            tare_values[key] = 0;
    }

    xerrors::Error tare(json &arg) {
        xjson::Parser parser(arg);
        const auto channels = parser.optional_vec<synnax::ChannelKey>("keys", std::vector<synnax::ChannelKey>{});
        if (parser.error()) return parser.error();

        std::lock_guard lock(mutex);
        if (channels.empty()) {
            for (auto &[ch_key, tare_value]: tare_values) {
                auto it = this->last_raw_value.find(ch_key);
                if (it != last_raw_value.end())
                    tare_values[ch_key] = it->second;
            }
            return xerrors::NIL;
        }

        for (auto &key: channels) {
            auto it = this->last_raw_value.find(key);
            if (it != last_raw_value.end()) {
                tare_values[key] = it->second;
            } else {
                parser.field_err("keys", "Channel " + std::to_string(key) +
                                         " is not a configured channel to tare.");
                return parser.error();
            }
        }
        return xerrors::NIL;
    }

    xerrors::Error transform(synnax::Frame &frame) override {
        std::lock_guard lock(mutex);
        for (const auto &[ch_key, series]: frame) {
            auto tare_it = tare_values.find(ch_key);
            if (tare_it == tare_values.end()) continue;
            this->last_raw_value[ch_key] = telem::cast<double>(series.at(-1));
            auto tare = tare_it->second;
            series.sub_inplace(tare);
        }
        return xerrors::NIL;
    }
};

class UnaryLinearScale {
    double slope;
    double offset;
    telem::DataType dt;

public:
    explicit UnaryLinearScale(
        xjson::Parser &parser,
        const telem::DataType &dt
    ) : slope(parser.required<double>("slope")),
        offset(parser.required<double>("offset")),
        dt(dt) {
    }

    xerrors::Error transform_inplace(telem::Series &series) const {
        if (this->dt != series.data_type())
            return xerrors::Error(xerrors::VALIDATION, "series data type " + series.data_type().name() +
                                                       " does not match scale data type " + this->dt.name());
        
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
    explicit UnaryMapScale(
        xjson::Parser &parser,
        const telem::DataType &dt
    ) : prescaled_min(parser.required<double>("pre_scaled_min")),
        prescaled_max(parser.required<double>("pre_scaled_max")),
        scaled_min(parser.required<double>("scaled_min")),
        scaled_max(parser.required<double>("scaled_max")),
        dt(dt) {
    }

    xerrors::Error transform_inplace(telem::Series &series) const {
        if (this->dt != series.data_type())
            return xerrors::Error(xerrors::VALIDATION, "series data type " + series.data_type().name() +
                                                       " does not match scale data type " + this->dt.name());
        
        // (v - prescaled_min) / (prescaled_max - prescaled_min) * (scaled_max - scaled_min) + scaled_min
        series.sub_inplace(prescaled_min);
        series.divide_inplace(prescaled_max - prescaled_min);
        series.multiply_inplace(scaled_max - scaled_min);  // * (scaled_max - scaled_min)
        series.add_inplace(scaled_min);
        
        return xerrors::NIL;
    }
};

class Scale final : public Transform {
    std::map<synnax::ChannelKey, std::variant<UnaryLinearScale, UnaryMapScale> > scales;

public:
    explicit Scale(const xjson::Parser &parser, const std::unordered_map<synnax::ChannelKey, synnax::Channel> &channels) {
        parser.iter("channels", [this, &channels](xjson::Parser &channel_parser) {
            const auto key = channel_parser.required<synnax::ChannelKey>("channel");
            const auto enabled = channel_parser.optional<bool>("enabled", true);
            auto scale_parser = channel_parser.optional_child("scale");
            if (!channel_parser.ok() || !enabled) return;
            const auto ch_t = channels.find(key);
            if (ch_t == channels.end()) {
                channel_parser.field_err("channel", "Channel " + std::to_string(key) + " is not a configured channel.");
                return;
            }
            const auto type = scale_parser.required<std::string>("type");
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

    xerrors::Error transform(synnax::Frame &frame) override {
        if (frame.empty()) return xerrors::NIL;
        for (const auto [key, series]: frame) {
            auto it = scales.find(key);
            if (it == scales.end()) continue;
            xerrors::Error err = std::visit([&series](const auto& scale) -> xerrors::Error {
                return scale.transform_inplace(series);
            }, it->second);
            if (err) return err;
        }
        return xerrors::NIL;
    }
};
}
