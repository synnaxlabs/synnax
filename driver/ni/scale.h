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
#include <string>
#include <utility>
#include <vector>
#include <cstdint>
#include <map>

/// external
#include "nlohmann/json.hpp"
#include "glog/logging.h"

/// module
#include "x/cpp/xjson/xjson.h"

/// internal
#include "driver/ni/util.h"
#include "driver/ni/daqmx/nidaqmx.h"

namespace ni {
static std::string next_scale_key() {
    static std::atomic counter = 0;
    return "scale_" + std::to_string(counter++);
}

struct Scale {
    virtual ~Scale() = default;

    virtual std::pair<std::string, int32> apply(
        const std::shared_ptr<DAQmx> &dmx
    ) { return {"", 0}; }
};

struct BaseScale : Scale {
    std::string type;
    std::string pre_scaled_units;
    std::string scaled_units;

    explicit BaseScale(xjson::Parser &cfg):
        type(cfg.required<std::string>("type")),
        pre_scaled_units(cfg.optional<std::string>("pre_scaled_units", "Volts")),
        scaled_units(cfg.optional<std::string>("scaled_units", "Volts")) {
    }
};

class LinearScale final : public BaseScale {
    float64 slope, offset;

public:
    explicit LinearScale(xjson::Parser &cfg) :
        BaseScale(cfg),
        slope(cfg.required<double>("slope")),
        offset(cfg.required<double>("y_intercept")) {
    }

    std::pair<std::string, int32> apply(const std::shared_ptr<DAQmx> &dmx) override {
        auto key = next_scale_key();
        return {
            key,
            dmx->CreateLinScale(
                key.c_str(),
                this->slope,
                this->offset,
                UNITS_MAP.at(this->pre_scaled_units),
                this->scaled_units.c_str()
            )
        };
    }
};

class MapScale final : public BaseScale {
    float64 pre_scaled_min, pre_scaled_max, scaled_min, scaled_max;

public:
    explicit MapScale(xjson::Parser &cfg):
        BaseScale(cfg),
        pre_scaled_min(cfg.required<double>("pre_scaled_min")),
        pre_scaled_max(cfg.required<double>("pre_scaled_max")),
        scaled_min(cfg.required<double>("scaled_min")),
        scaled_max(cfg.required<double>("scaled_max")) {
    }

    std::pair<std::string, int32> apply(const std::shared_ptr<DAQmx> &dmx) override {
        auto key = next_scale_key();
        return {
            key,
            dmx->CreateMapScale(
                key.c_str(),
                this->pre_scaled_min,
                this->pre_scaled_max,
                this->scaled_min,
                this->scaled_max,
                ni::UNITS_MAP.at(this->pre_scaled_units),
                this->scaled_units.c_str()
            )
        };
    }
};

class PolynomialScale final : public BaseScale {
    std::vector<double> forward_coeffs, reverse_coeffs;
    uint32_t num_coeffs;
    float64 min_x, max_x;
    int32 num_points, poly_order;

public:
    explicit PolynomialScale(xjson::Parser &cfg):
        BaseScale(cfg),
        num_coeffs(cfg.required<int>("num_coeffs")),
        min_x(cfg.required<double>("min_x")),
        max_x(cfg.required<double>("max_x")),
        num_points(cfg.optional<int>("num_reverse_coeffs", 0)),
        poly_order(cfg.required<int>("poly_order")) {
        forward_coeffs.resize(num_coeffs * 2);
        reverse_coeffs.resize(num_coeffs * 2);

        if (!cfg.ok()) {
            LOG(ERROR) <<
                    "[ni.analog] failed to parse custom polynomial scale configuration";
            return;
        }

        //TODO: handle if there is reverse coeffs of different size than forward coeffs
        json j = cfg.get_json();
        // get forward coeffs (prescale -> scale conversions)
        if (!j.contains("coeffs")) {
            LOG(ERROR) <<
                    "[ni.analog] failed to parse custom polynomial scale configuration: missing coeffs";
            return;
        }

        for (int i = 0; i < num_coeffs; i++) forward_coeffs[i] = j["coeffs"][i];
    }

    std::pair<std::string, int32> apply(const std::shared_ptr<DAQmx> &dmx) override {
        auto key = next_scale_key();
        dmx->CalculateReversePolyCoeff(
            this->forward_coeffs.data(),
            this->num_coeffs,
            this->min_x,
            this->max_x,
            this->num_coeffs,
            -1,
            this->reverse_coeffs.data()
        );
        return {
            key,
            dmx->CreatePolynomialScale(
                key.c_str(),
                this->forward_coeffs.data(),
                this->num_coeffs,
                this->reverse_coeffs.data(),
                this->num_coeffs,
                ni::UNITS_MAP.at(this->pre_scaled_units),
                this->scaled_units.c_str()
            )
        };
    }
};

class TableScale final : public BaseScale {
    std::vector<double> pre_scaled, scaled;
public:
    explicit TableScale(xjson::Parser &cfg):
        BaseScale(cfg),
        pre_scaled(cfg.required_vec<double>("pre_scaled_units")),
        scaled(cfg.required_vec<double>("scaled_units")) {
        if (pre_scaled.size() == scaled.size()) return;
        cfg.field_err("pre_scaled_vals",
                      "pre_scaled and scaled values must be the same size");
    }

    std::pair<std::string, int32> apply(const std::shared_ptr<DAQmx> &dmx) override {
        auto key = next_scale_key();
        return {
            key,
            dmx->CreateTableScale(
                key.c_str(),
                this->pre_scaled.data(),
                this->pre_scaled.size(),
                this->scaled.data(),
                this->pre_scaled.size(),
                ni::UNITS_MAP.at(this->pre_scaled_units),
                this->scaled_units.c_str()
            )
        };
    }
};

inline std::unique_ptr<Scale> parse_scale(const xjson::Parser &parent_cfg,
                                          const std::string &path) {
    auto cfg = parent_cfg.child(path);
    const auto type = cfg.required<std::string>("type");
    if (type == "linear") return std::make_unique<LinearScale>(cfg);
    if (type == "map") return std::make_unique<MapScale>(cfg);
    if (type == "polynomial") return std::make_unique<PolynomialScale>(cfg);
    if (type == "table") return std::make_unique<TableScale>(cfg);
    if (type != "none")
        cfg.field_err("type", "invalid scale type");
    return std::make_unique<Scale>();
}
}
