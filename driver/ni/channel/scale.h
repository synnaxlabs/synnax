// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <string>
#include <vector>

#include "client/cpp/ni/types.gen.h"
#include "x/cpp/json/json.h"

#include "driver/ni/channel/units.h"
#include "driver/ni/daqmx/sugared.h"

namespace driver::ni::channel {
/// @brief Generates a unique scale key using an atomic counter
/// @return A unique string identifier for a scale in the format "scale_<number>"
static std::string next_scale_key() {
    static std::atomic<size_t> counter = 0;
    return "scale_" + std::to_string(counter++);
}

/// @brief abstract class for a scale that will be applied to a channel.
struct Scale {
    virtual ~Scale() = default;

    /// @brief returns true if the scale should not be applied.
    virtual bool is_none() { return true; }

    /// @brief applies the scale to the DAQmx task, returning a key for the scale
    /// and any error that occurred during application.
    virtual std::pair<std::string, x::errors::Error>
    apply(const std::shared_ptr<daqmx::SugaredAPI> &dmx) {
        return {"", x::errors::NIL};
    }
};

/// @brief base scale data structure for all scale types.
struct BaseScale : Scale {
    const std::string scaled_units;
    const int pre_scaled_units;

    bool is_none() override { return false; }

    BaseScale(std::string scaled_units, const int pre_scaled_units):
        scaled_units(std::move(scaled_units)), pre_scaled_units(pre_scaled_units) {}
};

/// @brief Linear scaling that applies y = mx + b transformation
/// @details Transforms values using a linear equation with configurable slope and
/// y-intercept
struct LinearScale final : BaseScale {
    /// @brief The slope (m) in the linear equation
    const double slope;
    /// @brief The y-intercept (b) in the linear equation
    const double offset;

    LinearScale(const ::synnax::ni::ScaleLinear &s, const int pre_scaled_units):
        BaseScale(s.scaled_units, pre_scaled_units),
        slope(s.slope),
        offset(s.y_intercept) {}

    std::pair<std::string, x::errors::Error>
    apply(const std::shared_ptr<daqmx::SugaredAPI> &dmx) override {
        auto key = next_scale_key();
        return {
            key,
            dmx->CreateLinScale(
                key.c_str(),
                this->slope,
                this->offset,
                this->pre_scaled_units,
                this->scaled_units.c_str()
            )
        };
    }
};

/// @brief Map scaling that performs linear interpolation between configured ranges
/// @details Maps values from one range [pre_scaled_min, pre_scaled_max] to another
/// range [scaled_min, scaled_max]
struct MapScale final : BaseScale {
    /// @brief Minimum value in the pre-scaled range
    const double pre_scaled_min;
    /// @brief Maximum value in the pre-scaled range
    const double pre_scaled_max;
    /// @brief Minimum value in the scaled range
    const double scaled_min;
    /// @brief Maximum value in the scaled range
    const double scaled_max;

    MapScale(const ::synnax::ni::ScaleMap &s, const int pre_scaled_units):
        BaseScale(s.scaled_units, pre_scaled_units),
        pre_scaled_min(s.pre_scaled_min),
        pre_scaled_max(s.pre_scaled_max),
        scaled_min(s.scaled_min),
        scaled_max(s.scaled_max) {}

    std::pair<std::string, x::errors::Error>
    apply(const std::shared_ptr<daqmx::SugaredAPI> &dmx) override {
        auto key = next_scale_key();
        return {
            key,
            dmx->CreateMapScale(
                key.c_str(),
                this->pre_scaled_min,
                this->pre_scaled_max,
                this->scaled_min,
                this->scaled_max,
                this->pre_scaled_units,
                this->scaled_units.c_str()
            )
        };
    }
};

/// @brief Polynomial scaling that applies an nth-order polynomial transformation
/// @details Transforms values using both forward and reverse polynomial
/// coefficients
struct PolynomialScale final : BaseScale {
    /// @brief Coefficients for the forward polynomial transformation
    const std::vector<double> forward_coeffs;
    /// @brief Coefficients for the reverse polynomial transformation
    const std::vector<double> reverse_coeffs;

    PolynomialScale(const ::synnax::ni::ScalePolynomial &s, const int pre_scaled_units):
        BaseScale(s.scaled_units, pre_scaled_units),
        forward_coeffs(s.forward_coeffs),
        reverse_coeffs(s.reverse_coeffs) {}

    std::pair<std::string, x::errors::Error>
    apply(const std::shared_ptr<daqmx::SugaredAPI> &dmx) override {
        auto key = next_scale_key();
        return {
            key,
            dmx->CreatePolynomialScale(
                key.c_str(),
                this->forward_coeffs.data(),
                this->forward_coeffs.size(),
                this->reverse_coeffs.data(),
                this->reverse_coeffs.size(),
                this->pre_scaled_units,
                this->scaled_units.c_str()
            )
        };
    }
};

/// @brief Table scaling that performs lookup-based transformation
/// @details Transforms values using a lookup table with linear interpolation
/// between points
struct TableScale final : BaseScale {
    /// @brief Input values for the lookup table
    const std::vector<double> pre_scaled;
    /// @brief Output values for the lookup table
    const std::vector<double> scaled;

    TableScale(const ::synnax::ni::ScaleTable &s, const int pre_scaled_units):
        BaseScale(s.scaled_units, pre_scaled_units),
        pre_scaled(s.pre_scaled_vals),
        scaled(s.scaled_vals) {}

    std::pair<std::string, x::errors::Error>
    apply(const std::shared_ptr<daqmx::SugaredAPI> &dmx) override {
        auto key = next_scale_key();
        return {
            key,
            dmx->CreateTableScale(
                key.c_str(),
                this->pre_scaled.data(),
                this->pre_scaled.size(),
                this->scaled.data(),
                this->pre_scaled.size(),
                this->pre_scaled_units,
                this->scaled_units.c_str()
            )
        };
    }
};

/// @brief Creates a Scale from a generated scale union value, validating units
/// and structural invariants.
inline std::pair<std::unique_ptr<Scale>, x::errors::Error>
make_scale(const ::synnax::ni::Scale &s) {
    return std::visit(
        [](const auto &v) -> std::pair<std::unique_ptr<Scale>, x::errors::Error> {
            using T = std::decay_t<decltype(v)>;
            if constexpr (std::is_same_v<T, ::synnax::ni::ScaleNone>)
                return {std::make_unique<Scale>(), x::errors::NIL};
            else {
                auto [units, err] = parse_units(v.pre_scaled_units);
                if (err) return {nullptr, err};
                if constexpr (std::is_same_v<T, ::synnax::ni::ScaleLinear>)
                    return {std::make_unique<LinearScale>(v, units), x::errors::NIL};
                else if constexpr (std::is_same_v<T, ::synnax::ni::ScaleMap>)
                    return {std::make_unique<MapScale>(v, units), x::errors::NIL};
                else if constexpr (std::is_same_v<T, ::synnax::ni::ScaleTable>) {
                    if (v.pre_scaled_vals.size() != v.scaled_vals.size())
                        return {
                            nullptr,
                            x::errors::Error(
                                x::errors::VALIDATION,
                                "pre_scaled and scaled values must be the same size"
                            )
                        };
                    return {std::make_unique<TableScale>(v, units), x::errors::NIL};
                } else {
                    if (v.reverse_coeffs.empty())
                        return {
                            nullptr,
                            x::errors::Error(
                                x::errors::VALIDATION,
                                "polynomial scale requires reverse coefficients"
                            )
                        };
                    return {
                        std::make_unique<PolynomialScale>(v, units),
                        x::errors::NIL
                    };
                }
            }
        },
        s
    );
}
}
