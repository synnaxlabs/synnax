// Copyright 2024 Synnax Labs, Inc.
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

    LinearScale(config::Parser &parser)
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

    MapScale(config::Parser &parser)
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

    PolynomialScale(
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

        // get reverse coeffs (scale -> prescale conversions)
        // TODO: reverse coeffs might be smaller than forward_coeffs
        // dmx->CalculateReversePolyCoeff(
        //     forward_coeffs.data(),
        //     num_coeffs,
        //     min_x,
        //     max_x,
        //     num_coeffs,
        //     -1,
        //     reverse_coeffs.data()
        // ); // FIXME: reversePoly order should be user inputted?
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

    TableScale(config::Parser &parser) {
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
    config::Parser parser;
    LinearScale linear;
    MapScale map;
    PolynomialScale polynomial;
    TableScale table;

    ScaleConfig() = default;

    ScaleConfig(config::Parser &parser, std::string &name)
        : name(name),
          type(parser.required<std::string>("type")),
          prescaled_units(
              parser.optional<std::string>("pre_scaled_units", "Volts")),
          scaled_units(parser.optional<std::string>("scaled_units", "Volts")),
          parser(parser) {
        if (!parser.ok()) {
            LOG(ERROR)
                    << "[ni.analog] failed to parse custom scale configuration for "
                    << name;
            return;
        }

        if (type == "linear") new(&linear) LinearScale(parser);
        else if (type == "map") new(&map) MapScale(parser);
        else if (type == "polynomial") new(&polynomial) PolynomialScale(parser);
        else if (type == "table") new(&table) TableScale(parser);
    }

    ScaleConfig(const ScaleConfig &other)
        : name(other.name),
          type(other.type),
          prescaled_units(other.prescaled_units),
          scaled_units(other.scaled_units),
          parser(other.parser) {
        if (type == "linear") linear = LinearScale(parser);
        else if (type == "map") map = MapScale(parser);
        else if (type == "polynomial") polynomial = PolynomialScale(parser);
        else if (type == "table") {
            table = TableScale(parser);
            table.prescaled = other.table.prescaled;
            table.scaled = other.table.scaled;
            table.num_points = other.table.num_points;
        }
    }

    ~ScaleConfig() {
        if (type == "linear") linear.~LinearScale();
        else if (type == "map") map.~MapScale();
        else if (type == "polynomial") polynomial.~PolynomialScale();
        else if (type == "table") table.~TableScale();
    }

    ScaleConfig &operator=(const ScaleConfig &other) {
        if (this == &other) return *this;

        name = other.name;
        type = other.type;
        prescaled_units = other.prescaled_units;
        scaled_units = other.scaled_units;
        parser = other.parser;

        if (type == "linear") linear = LinearScale(parser);
        else if (type == "map") map = MapScale(parser);
        else if (type == "polynomial") polynomial = PolynomialScale(parser);
        else if (type == "table") table = TableScale(parser);

        return *this;
    }

    ScaleConfig(ScaleConfig &&other) = delete;

    int32 create_ni_scale(const std::shared_ptr<DAQmx> &dmx) {
        if (type == "linear")
            return dmx->CreateLinScale(
                name.c_str(),
                linear.slope,
                linear.offset,
                ni::UNITS_MAP.at(prescaled_units),
                scaled_units.c_str()
            );
        else if (type == "map")
            return dmx->CreateMapScale(
                name.c_str(),
                map.prescaled_min,
                map.prescaled_max,
                map.scaled_min,
                map.scaled_max,
                ni::UNITS_MAP.at(prescaled_units),
                scaled_units.c_str()
            );
        else if (type == "polynomial")
            return dmx->CreatePolynomialScale(
                name.c_str(),
                polynomial.forward_coeffs.data(),
                polynomial.num_coeffs,
                polynomial.reverse_coeffs.data(),
                polynomial.num_coeffs,
                ni::UNITS_MAP.at(prescaled_units),
                scaled_units.c_str()
            );
        else if (type == "table")
            return dmx->CreateTableScale(
                name.c_str(),
                table.prescaled.data(),
                table.num_points,
                table.scaled.data(),
                table.num_points,
                ni::UNITS_MAP.at(prescaled_units),
                scaled_units.c_str()
            );
        return 0;
    }
};
};
