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

    virtual xerrors::Error transform(Frame &frame) = 0;
};

class Chain final : public Transform {
public:
    void add(const std::shared_ptr<Transform> &transforms) {
        this->transforms.push_back(transforms);
    }

    xerrors::Error transform(Frame &frame) override {
        if (transforms.empty()) return xerrors::NIL;
        for (const auto &t: this->transforms)
            if (const auto err = t->transform(frame))
                return err;
        return xerrors::NIL;
    }

private:
    std::vector<std::shared_ptr<Transform> > transforms;
};

/// @brief middleware to tare data written to channels based on the last frame processed at the time of taring
/// This middleware should added to the pipeline middleware chain first so that it can tare the data before any other middleware
/// can process it.
class Tare final : public Transform {
    std::map<synnax::ChannelKey, telem::NumericSampleValue> tare_values;
    std::map<synnax::ChannelKey, telem::NumericSampleValue> last_raw_value;
    std::unordered_map<synnax::ChannelKey, synnax::Channel> tare_channels;
    std::mutex mutex;

public:
    explicit Tare(const std::vector<synnax::Channel> &channels): tare_channels(
        synnax::map_channel_Keys(channels)) {
        for (auto &[key, ch]: this->tare_channels)
            tare_values[key] = telem::narrow_numeric(ch.data_type.cast(0));
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

    xerrors::Error transform(Frame &frame) override {
        std::lock_guard lock(mutex);
        for (const auto &[ch_key, series]: frame) {
            auto tare_it = tare_values.find(ch_key);
            if (tare_it == tare_values.end()) continue;
            auto v = telem::narrow_numeric(series.at(-1));
            this->last_raw_value[ch_key] = v;
            auto tare = tare_it->second;
            series.map_inplace([tare](const telem::NumericSampleValue &val) {
                return val - tare;
            });
        }
        return xerrors::NIL;
    }
};

class UnaryLinearScale {
    telem::NumericSampleValue slope;
    telem::NumericSampleValue offset;
    telem::DataType dt;

public:
    explicit UnaryLinearScale(
        xjson::Parser &parser,
        telem::DataType dt
    ) : slope(telem::narrow_numeric(dt.cast(parser.required<double>("slope")))),
        offset(telem::narrow_numeric(dt.cast(parser.required<double>("offset")))),
        dt(dt) {
    }

    xerrors::Error transform_inplace(telem::Series &series) const {
        if (this->dt != series.data_type())
            return xerrors::Error(xerrors::VALIDATION, "series data type " + series.data_type().name() +
                                                       " does not match scale data type " + this->dt.name());
        series.map_inplace([this, &series](const telem::NumericSampleValue &val) {
            return val * this->slope + this->offset;
        });
        return xerrors::NIL;
    }
};

class UnaryMapScale {
    telem::NumericSampleValue prescaled_min;
    telem::NumericSampleValue prescaled_max;
    telem::NumericSampleValue scaled_min;
    telem::NumericSampleValue scaled_max;
    telem::DataType dt;

public:
    explicit UnaryMapScale(
        xjson::Parser &parser,
        const telem::DataType &dt
    ) : prescaled_min(telem::narrow_numeric(dt.cast(parser.required<double>("pre_scaled_min")))),
        prescaled_max(telem::narrow_numeric(dt.cast(parser.required<double>("pre_scaled_max")))),
        scaled_min(telem::narrow_numeric(dt.cast(parser.required<double>("scaled_min")))),
        scaled_max(telem::narrow_numeric(dt.cast(parser.required<double>("scaled_max")))),
        dt(dt) {
    }

    xerrors::Error transform_inplace(telem::Series &series) const {
        if (this->dt != series.data_type())
            return xerrors::Error(xerrors::VALIDATION, "series data type " + series.data_type().name() +
                                                       " does not match scale data type " + this->dt.name());
        series.map_inplace([this](const telem::NumericSampleValue &v) {
            return (v - prescaled_min) / (prescaled_max - prescaled_min) * (
                       scaled_max - scaled_min) +
                   scaled_min;
        });
        return xerrors::NIL;
    }
};

class Scale final : public Transform {
    std::map<synnax::ChannelKey, std::variant<UnaryLinearScale, UnaryMapScale> > scales;

public:
    explicit Scale(const xjson::Parser &parser, const std::unordered_map<synnax::ChannelKey, synnax::Channel> &channels) {
        parser.iter("channels", [this, &channels](xjson::Parser &channel_parser) {
            const auto key = channel_parser.required<synnax::ChannelKey>("channel");
            auto scale = channel_parser.optional_child("scale");
            if (!channel_parser.ok()) return;
            auto ch_t = channels.find(key);
            if (ch_t == channels.end()) {
                channel_parser.field_err("channel", "Channel " + std::to_string(key) + " is not a configured channel.");
                return;
            }
            const auto type = scale.required<std::string>("type");
            auto dt = ch_t->second.data_type;
            if (type == "linear") {
                UnaryLinearScale linear_scale(scale, dt);
                scales.emplace(key, std::move(linear_scale));
            } else if (type == "map") {
                UnaryMapScale map_scale(scale, dt);
                scales.emplace(key, std::move(map_scale));
            }
        });
    }

    xerrors::Error transform(Frame &frame) override {
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
