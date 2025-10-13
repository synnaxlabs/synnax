// Copyright 2025 Synnax Labs, Inc.
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

#include "x/cpp/xjson/xjson.h"

#include "driver/ni/channel/units.h"
#include "driver/ni/daqmx/sugared.h"

namespace channel {
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
    virtual std::pair<std::string, xerrors::Error>
    apply(const std::shared_ptr<daqmx::SugaredAPI> &dmx) {
        return {"", xerrors::NIL};
    }
};

/// @brief base scale data structure for all scale types.
struct BaseScale : Scale {
    const std::string type, scaled_units;
    const int pre_scaled_units;

    bool is_none() override { return false; }

    explicit BaseScale(xjson::Parser &cfg):
        type(cfg.required<std::string>("type")),
        pre_scaled_units(parse_units(cfg, "pre_scaled_units")),
        scaled_units(cfg.optional<std::string>("scaled_units", "Volts")) {}
};

/// @brief Linear scaling that applies y = mx + b transformation
/// @details Transforms values using a linear equation with configurable slope and
/// y-intercept
struct LinearScale final : BaseScale {
    /// @brief The slope (m) in the linear equation
    const double slope;
    /// @brief The y-intercept (b) in the linear equation
    const double offset;

    explicit LinearScale(xjson::Parser &cfg):
        BaseScale(cfg),
        slope(cfg.required<double>("slope")),
        offset(cfg.required<double>("y_intercept")) {}

    std::pair<std::string, xerrors::Error>
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

    explicit MapScale(xjson::Parser &cfg):
        BaseScale(cfg),
        pre_scaled_min(cfg.required<double>("pre_scaled_min")),
        pre_scaled_max(cfg.required<double>("pre_scaled_max")),
        scaled_min(cfg.required<double>("scaled_min")),
        scaled_max(cfg.required<double>("scaled_max")) {}

    std::pair<std::string, xerrors::Error>
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

/// @brief the default mode for calculating the reverse polynomial is to use the
/// same number of coefficients as the forward polynomial.
constexpr int REVERSE_POLY_ORDER_SAME_AS_FORWARD = -1;

/// @brief Polynomial scaling that applies an nth-order polynomial transformation
/// @details Transforms values using both forward and reverse polynomial
/// coefficients
struct PolynomialScale final : BaseScale {
    /// @brief Coefficients for the forward polynomial transformation
    std::vector<double> forward_coeffs;
    /// @brief Minimum input value for the polynomial
    const double min_x;
    /// @brief Maximum input value for the polynomial
    const double max_x;
    /// @brief Order of the reverse polynomial (or -1 to match forward order)
    const int reverse_poly_order;
    /// @brief Number of points used to compute reverse coefficients
    const size_t num_points_to_compute;

    explicit PolynomialScale(xjson::Parser &cfg):
        BaseScale(cfg),
        forward_coeffs(cfg.required_vec<double>("forward_coeffs")),
        min_x(cfg.required<double>("min_x")),
        max_x(cfg.required<double>("max_x")),
        reverse_poly_order(cfg.optional<int>("poly_order", -1)),
        num_points_to_compute(cfg.optional<size_t>("num_points_to_compute", 100)) {}

    std::pair<std::string, xerrors::Error>
    apply(const std::shared_ptr<daqmx::SugaredAPI> &dmx) override {
        auto key = next_scale_key();
        std::vector<double> reverse_coeffs(this->forward_coeffs.size());
        if (const auto err = dmx->CalculateReversePolyCoeff(
                this->forward_coeffs.data(),
                this->forward_coeffs.size(),
                this->min_x,
                this->max_x,
                this->num_points_to_compute,
                this->reverse_poly_order,
                reverse_coeffs.data()
            ))
            return {key, err};
        return {
            key,
            dmx->CreatePolynomialScale(
                key.c_str(),
                this->forward_coeffs.data(),
                this->forward_coeffs.size(),
                reverse_coeffs.data(),
                reverse_coeffs.size(),
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

    explicit TableScale(xjson::Parser &cfg):
        BaseScale(cfg),
        pre_scaled(cfg.required_vec<double>("pre_scaled")),
        scaled(cfg.required_vec<double>("scaled")) {
        if (pre_scaled.size() == scaled.size()) return;
        cfg.field_err(
            "pre_scaled_vals",
            "pre_scaled and scaled values must be the same size"
        );
    }

    std::pair<std::string, xerrors::Error>
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

/// @brief Creates a Scale object based on configuration
/// @param parent_cfg The parent configuration parser
/// @param path The path to the scale configuration within the parent
/// @return A unique pointer to the created Scale object
inline std::unique_ptr<Scale>
parse_scale(const xjson::Parser &parent_cfg, const std::string &path) {
    auto cfg = parent_cfg.child(path);
    const auto type = cfg.required<std::string>("type");
    if (type == "linear") return std::make_unique<LinearScale>(cfg);
    if (type == "map") return std::make_unique<MapScale>(cfg);
    if (type == "polynomial") return std::make_unique<PolynomialScale>(cfg);
    if (type == "table") return std::make_unique<TableScale>(cfg);
    if (type == "none") return std::make_unique<Scale>();
    cfg.field_err("type", "invalid scale type");
    return nullptr;
}
}
