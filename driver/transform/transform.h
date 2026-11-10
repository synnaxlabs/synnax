// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <algorithm>
#include <cmath>
#include <functional>
#include <map>
#include <memory>
#include <mutex>
#include <ranges>
#include <string>
#include <unordered_set>
#include <variant>
#include <vector>

#include "client/cpp/channel/channel.h"
#include "client/cpp/task/json.gen.h"
#include "x/cpp/json/json.h"
#include "x/cpp/telem/frame.h"
#include "x/cpp/telem/series.h"
#include "x/cpp/telem/telem.h"

namespace driver::transform {
class Transform {
public:
    virtual ~Transform() = default;

    virtual x::errors::Error transform(x::telem::Frame &frame) = 0;
};

class Chain final : public Transform {
public:
    void add(const std::shared_ptr<Transform> &transforms) {
        this->transforms.push_back(transforms);
    }

    x::errors::Error transform(x::telem::Frame &frame) override {
        if (transforms.empty()) return x::errors::NIL;
        for (const auto &t: this->transforms)
            if (const auto err = t->transform(frame)) return err;
        return x::errors::NIL;
    }

private:
    std::vector<std::shared_ptr<Transform>> transforms;
};

/// @brief middleware to tare data written to channels based on the last frame
/// processed at the time of taring. This middleware should be added to the pipeline
/// middleware chain first so that it can tare the data before any other middleware
/// can process it.
class Tare final : public Transform {
    std::unordered_map<synnax::channel::Key, double> tare_values;
    std::unordered_map<synnax::channel::Key, synnax::channel::Channel> tare_channels;
    std::unordered_set<synnax::channel::Key> channels_to_tare;
    bool tare_all{false};
    std::mutex mutex;

public:
    explicit Tare(const std::vector<synnax::channel::Channel> &channels):
        tare_channels(synnax::channel::map_channel_keys(channels)) {}

    x::errors::Error tare(x::json::json::object_t &arg) {
        x::json::Parser parser(arg);
        const auto channels = parser.field<std::vector<synnax::channel::Key>>(
            "keys",
            std::vector<synnax::channel::Key>{}
        );
        if (parser.error()) return parser.error();

        std::lock_guard lock(mutex);
        if (channels.empty()) {
            tare_all = true;
            channels_to_tare.clear();
            return x::errors::NIL;
        }

        for (const auto &key: channels) {
            if (!tare_channels.contains(key)) {
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
        return x::errors::NIL;
    }

    x::errors::Error transform(x::telem::Frame &frame) override {
        std::lock_guard lock(mutex);
        if (tare_all || !channels_to_tare.empty()) {
            for (const auto &[ch_key, series]: frame)
                if (tare_all || channels_to_tare.contains(ch_key))
                    tare_values[ch_key] = series.avg<double>();
            tare_all = false;
            channels_to_tare.clear();
        }

        for (const auto &[ch_key, series]: frame) {
            auto tare_it = tare_values.find(ch_key);
            if (tare_it == tare_values.end()) continue;
            series.sub_inplace(tare_it->second);
        }
        return x::errors::NIL;
    }
};

class UnaryLinearScale {
    double slope;
    double offset;
    x::telem::DataType dt;

public:
    UnaryLinearScale(const double slope, const double offset, x::telem::DataType dt):
        slope(slope), offset(offset), dt(std::move(dt)) {}

    [[nodiscard]] x::errors::Error
    transform_inplace(const x::telem::Series &series) const {
        if (this->dt != series.data_type())
            return x::errors::Error(
                x::errors::VALIDATION,
                "series data type " + series.data_type().name() +
                    " does not match scale data type " + this->dt.name()
            );

        // val * slope + offset
        series.multiply_inplace(slope);
        series.add_inplace(offset);

        return x::errors::NIL;
    }
};

class UnaryMapScale {
    double prescaled_min;
    double prescaled_max;
    double scaled_min;
    double scaled_max;
    x::telem::DataType dt;

public:
    UnaryMapScale(
        const double pre_scaled_min,
        const double pre_scaled_max,
        const double scaled_min,
        const double scaled_max,
        x::telem::DataType dt
    ):
        prescaled_min(pre_scaled_min),
        prescaled_max(pre_scaled_max),
        scaled_min(scaled_min),
        scaled_max(scaled_max),
        dt(std::move(dt)) {}

    [[nodiscard]] x::errors::Error
    transform_inplace(const x::telem::Series &series) const {
        if (this->dt != series.data_type())
            return x::errors::Error(
                x::errors::VALIDATION,
                "series data type " + series.data_type().name() +
                    " does not match scale data type " + this->dt.name()
            );

        // (v - prescaled_min) / (prescaled_max - prescaled_min) * (scaled_max -
        // scaled_min) + scaled_min
        series.sub_inplace(prescaled_min);
        series.divide_inplace(prescaled_max - prescaled_min);
        series.multiply_inplace(scaled_max - scaled_min); // * (scaled_max - scaled_min)
        series.add_inplace(scaled_min);

        return x::errors::NIL;
    }
};

class UnaryPolynomialScale {
    std::vector<double> coeffs;
    x::telem::DataType dt;

public:
    UnaryPolynomialScale(std::vector<double> coeffs, x::telem::DataType dt):
        coeffs(std::move(coeffs)), dt(std::move(dt)) {}

    [[nodiscard]] x::errors::Error
    transform_inplace(const x::telem::Series &series) const {
        if (this->dt != series.data_type())
            return x::errors::Error(
                x::errors::VALIDATION,
                "series data type " + series.data_type().name() +
                    " does not match scale data type " + this->dt.name()
            );
        series.map_inplace([this](const double v) {
            double result = 0;
            for (const double c: std::views::reverse(this->coeffs))
                result = result * v + c;
            return result;
        });
        return x::errors::NIL;
    }
};

/// @brief interpolates between table points in a straight line. Values outside the
/// table clip to its first or last scaled value, and NaN stays NaN. pre_scaled must
/// hold at least two values in increasing order, and scaled the same count.
class UnaryTableScale {
    std::vector<double> pre_scaled;
    std::vector<double> scaled;
    x::telem::DataType dt;

public:
    UnaryTableScale(
        std::vector<double> pre_scaled,
        std::vector<double> scaled,
        x::telem::DataType dt
    ):
        pre_scaled(std::move(pre_scaled)),
        scaled(std::move(scaled)),
        dt(std::move(dt)) {}

    [[nodiscard]] x::errors::Error
    transform_inplace(const x::telem::Series &series) const {
        if (this->dt != series.data_type())
            return x::errors::Error(
                x::errors::VALIDATION,
                "series data type " + series.data_type().name() +
                    " does not match scale data type " + this->dt.name()
            );
        series.map_inplace([this](const double v) { return this->interpolate(v); });
        return x::errors::NIL;
    }

private:
    [[nodiscard]] double interpolate(const double v) const {
        if (std::isnan(v)) return v;
        if (v <= this->pre_scaled.front()) return this->scaled.front();
        if (v >= this->pre_scaled.back()) return this->scaled.back();
        const auto hi = static_cast<size_t>(
            std::ranges::upper_bound(this->pre_scaled, v) - this->pre_scaled.begin()
        );
        const auto lo = hi - 1;
        const double t = (v - this->pre_scaled[lo]) /
                         (this->pre_scaled[hi] - this->pre_scaled[lo]);
        return this->scaled[lo] + t * (this->scaled[hi] - this->scaled[lo]);
    }
};

/// @brief binds an error to each invalid field of the scale configured at path.
inline void validate_scale(
    const x::json::Parser &parser,
    const std::string &path,
    const ::synnax::task::Scale &scale
) {
    if (const auto *m = std::get_if<::synnax::task::MapScale>(&scale)) {
        if (m->pre_scaled_min == m->pre_scaled_max)
            parser.field_err(
                path + ".pre_scaled_max",
                "Pre-scaled min and pre-scaled max must differ"
            );
        return;
    }
    if (const auto *p = std::get_if<::synnax::task::PolynomialScale>(&scale)) {
        if (p->coeffs.empty())
            parser.field_err(path + ".coeffs", "At least one coefficient is required");
        return;
    }
    const auto *t = std::get_if<::synnax::task::TableScale>(&scale);
    if (t == nullptr) return;
    const auto pre_path = path + ".pre_scaled_vals";
    if (t->pre_scaled_vals.size() != t->scaled_vals.size()) {
        const std::string
            msg = "Pre-scaled and scaled values must have the same length";
        parser.field_err(pre_path, msg);
        parser.field_err(path + ".scaled_vals", msg);
        return;
    }
    if (t->pre_scaled_vals.size() < 2) {
        parser.field_err(pre_path, "At least two points are required");
        return;
    }
    if (std::ranges::adjacent_find(t->pre_scaled_vals, std::greater_equal<>()) !=
        t->pre_scaled_vals.end())
        parser.field_err(pre_path, "Pre-scaled values must be in increasing order");
}

class Scale final : public Transform {
public:
    /// @brief a single-channel scale operation.
    using Unary = std::
        variant<UnaryLinearScale, UnaryMapScale, UnaryPolynomialScale, UnaryTableScale>;

    /// @brief registers the scale to apply to the given channel's series.
    void add(const synnax::channel::Key key, Unary scale) {
        this->scales.emplace(key, std::move(scale));
    }

    /// @brief registers the configured scale for the channel's samples of type dt.
    /// Registers nothing for an unscaled channel. The scale must pass validate_scale.
    void
    add(const synnax::channel::Key key,
        const x::telem::DataType &dt,
        const ::synnax::task::Scale &scale) {
        if (const auto *l = std::get_if<::synnax::task::LinearScale>(&scale))
            this->add(key, UnaryLinearScale(l->slope, l->offset, dt));
        else if (const auto *m = std::get_if<::synnax::task::MapScale>(&scale))
            this->add(
                key,
                UnaryMapScale(
                    m->pre_scaled_min,
                    m->pre_scaled_max,
                    m->scaled_min,
                    m->scaled_max,
                    dt
                )
            );
        else if (const auto *p = std::get_if<::synnax::task::PolynomialScale>(&scale))
            this->add(key, UnaryPolynomialScale(p->coeffs, dt));
        else if (const auto *t = std::get_if<::synnax::task::TableScale>(&scale))
            this->add(key, UnaryTableScale(t->pre_scaled_vals, t->scaled_vals, dt));
    }

    x::errors::Error transform(x::telem::Frame &frame) override {
        if (frame.empty()) return x::errors::NIL;
        for (const auto [key, series]: frame) {
            auto it = scales.find(key);
            if (it == scales.end()) continue;
            x::errors::Error err = std::visit(
                [&series](const auto &scale) -> x::errors::Error {
                    return scale.transform_inplace(series);
                },
                it->second
            );
            if (err) return err;
        }
        return x::errors::NIL;
    }

private:
    std::map<synnax::channel::Key, Unary> scales;
};
}
