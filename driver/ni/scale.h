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
#include <utility>
#include <cstdint>
#include <map>
#include <variant>

#include "driver/ni/util.h"
#include "nidaqmx/nidaqmx.h"
#include "driver/config/config.h"

#include "nlohmann/json.hpp"
#include "glog/logging.h"

namespace ni {
extern const std::map<std::string, int32_t> UNITS_MAP;

///////////////////////////////////////////////////////////////////////////////////
//                                    LinearScale                                //
///////////////////////////////////////////////////////////////////////////////////
struct LinearScale {
    float64 slope;
    float64 offset;

    LinearScale() = default;

    explicit LinearScale(config::Parser &parser)
        : slope(parser.required<double>("slope")),
          offset(parser.required<double>("y_intercept")) {
        if (!parser.ok())
            LOG(ERROR) <<
                    "[ni.analog] failed to parse custom linear configuration";
    }
};

///////////////////////////////////////////////////////////////////////////////////
//                                    MapScale                                   //
///////////////////////////////////////////////////////////////////////////////////
struct MapScale {
    float64 prescaled_min;
    float64 prescaled_max;
    float64 scaled_min;
    float64 scaled_max;

    MapScale() = default;

    explicit MapScale(config::Parser &parser)
        : prescaled_min(parser.required<double>("pre_scaled_min")),
          prescaled_max(parser.required<double>("pre_scaled_max")),
          scaled_min(parser.required<double>("scaled_min")),
          scaled_max(parser.required<double>("scaled_max")) {
        if (!parser.ok())
            LOG(ERROR) <<
                    "[ni.analog] failed to parse custom map configuration";
    }
};

///////////////////////////////////////////////////////////////////////////////////
//                                  PolynomialScale                              //
///////////////////////////////////////////////////////////////////////////////////
struct PolynomialScale {
    std::vector<double> forward_coeffs;
    std::vector<double> reverse_coeffs;
    uint32_t num_coeffs;
    float64 min_x;
    float64 max_x;
    int32 num_points;
    int32 poly_order;

    PolynomialScale() = default;

    explicit PolynomialScale(
        config::Parser &parser
    )
        : num_coeffs(parser.required<int>("num_coeffs")),
          min_x(parser.required<double>("min_x")),
          max_x(parser.required<double>("max_x")),
          num_points(parser.optional<int>("num_reverse_coeffs", 0)),
          poly_order(parser.required<int>("poly_order")) {
        forward_coeffs.resize(num_coeffs * 2);
        reverse_coeffs.resize(num_coeffs * 2);

        if (!parser.ok()) {
            LOG(ERROR) <<
                    "[ni.analog] failed to parse custom polynomial scale configuration";
            return;
        }

        //TODO: handle if there is reverse coeffs of different size than forward coeffs
        json j = parser.get_json();
        // get forward coeffs (prescale -> scale conversions)
        if (!j.contains("coeffs")) {
            LOG(ERROR) <<
                    "[ni.analog] failed to parse custom polynomial scale configuration: missing coeffs";
            return;
        }

        for (int i = 0; i < num_coeffs; i++) forward_coeffs[i] = j["coeffs"][i];
    }

    void calculate_coeffs(const std::shared_ptr<DAQmx> &dmx) {
        // TODO: reverse coeffs might be smaller than forward_coeffs
        dmx->CalculateReversePolyCoeff(
            this->forward_coeffs.data(),
            this->num_coeffs,
            this->min_x,
            this->max_x,
            this->num_coeffs,
            -1,
            this->reverse_coeffs.data()
        );
    }
};

///////////////////////////////////////////////////////////////////////////////////
//                                    TableScale                                 //
///////////////////////////////////////////////////////////////////////////////////
struct TableScale {
    std::vector<double> prescaled;
    std::vector<double> scaled;
    uint32_t num_points;

    TableScale() = default;

    explicit TableScale(config::Parser &parser) {
        if (!parser.ok()) {
            LOG(ERROR) << "[ni.analog] failed to parse custom table configuration";
            return;
        }

        prescaled = parser.required_vector<double>("pre_scaled_vals");
        scaled = parser.required_vector<double>("scaled_vals");

        if (prescaled.size() != scaled.size()) {
            LOG(ERROR) <<
                    "[ni.analog] failed to parse custom table configuration: prescaled and scaled values must be the same size";
            return;
        }

        num_points = prescaled.size();
    }
};

///////////////////////////////////////////////////////////////////////////////////
//                                    ScaleConfig                                //
//////////////////////////////////////////////////////////////////////////////////
struct ScaleConfig {
    std::string name;
    std::string type;
    std::string prescaled_units;
    std::string scaled_units;
    std::variant<LinearScale, MapScale, PolynomialScale, TableScale> scale;

    ScaleConfig() = default;

    ScaleConfig(config::Parser &parser, const std::string &name)
        : name(name),
          type(parser.required<std::string>("type")),
          prescaled_units(parser.optional<std::string>("pre_scaled_units", "Volts")),
          scaled_units(parser.optional<std::string>("scaled_units", "Volts")) {
        if (!parser.ok()) {
            LOG(ERROR) << "[ni.analog] failed to parse custom scale configuration for "
                    << name;
            return;
        }

        if (type == "linear") scale.emplace<LinearScale>(parser);
        else if (type == "map") scale.emplace<MapScale>(parser);
        else if (type == "polynomial") scale.emplace<PolynomialScale>(parser);
        else if (type == "table") scale.emplace<TableScale>(parser);
    }

    // Copy constructor can use default
    ScaleConfig(const ScaleConfig &) = default;

    // Move constructor no longer needs to be deleted
    ScaleConfig(ScaleConfig &&) = default;

    // Assignment operator can use default
    ScaleConfig &operator=(const ScaleConfig &) = default;

    int32 create_ni_scale(const std::shared_ptr<DAQmx> &dmx) {
        if (const auto linear = std::get_if<LinearScale>(&scale))
            return dmx->CreateLinScale(
                this->name.c_str(),
                linear->slope,
                linear->offset,
                ni::UNITS_MAP.at(prescaled_units),
                this->scaled_units.c_str()
            );
        if (const auto map = std::get_if<MapScale>(&scale))
            return dmx->CreateMapScale(
                this->name.c_str(),
                map->prescaled_min,
                map->prescaled_max,
                map->scaled_min,
                map->scaled_max,
                ni::UNITS_MAP.at(this->prescaled_units),
                this->scaled_units.c_str()
            );
        if (auto poly = std::get_if<PolynomialScale>(&scale)) {
            poly->calculate_coeffs(dmx);
            return dmx->CreatePolynomialScale(
                name.c_str(),
                poly->forward_coeffs.data(),
                poly->num_coeffs,
                poly->reverse_coeffs.data(),
                poly->num_coeffs,
                ni::UNITS_MAP.at(this->prescaled_units),
                this->scaled_units.c_str()
            );
        }
        if (const auto table = std::get_if<TableScale>(&scale))
            return dmx->CreateTableScale(
                this->name.c_str(),
                table->prescaled.data(),
                table->num_points,
                table->scaled.data(),
                table->num_points,
                ni::UNITS_MAP.at(this->prescaled_units),
                this->scaled_units.c_str()
            );
        return 0;
    }
};
};
