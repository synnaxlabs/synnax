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

#include "daqmx.h"
#include "nisyscfg.h"

#include "driver/ni/ni.h"
#include "driver/ni/nidaqmx_api.h"
#include "driver/config/config.h"

#include "nlohmann/json.hpp"
#include "glog/logging.h"

namespace ni {
extern const std::map<std::string, int32_t> UNITS_MAP;

///////////////////////////////////////////////////////////////////////////////////
//                                    LinearScale                                //
///////////////////////////////////////////////////////////////////////////////////
typedef struct LinearScale {
    float64 slope;
    float64 offset;

    LinearScale() = default;


    LinearScale(config::Parser &parser)
        : slope(parser.required<double>("slope")),
          offset(parser.required<double>("y_intercept")) {
        if (!parser.ok()) LOG(ERROR) <<
                          "[ni.analog] failed to parse custom linear configuration";
    }
} LinearScale;

///////////////////////////////////////////////////////////////////////////////////
//                                    MapScale                                   //
///////////////////////////////////////////////////////////////////////////////////
typedef struct MapScale {
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
        if (!parser.ok()) LOG(ERROR) <<
                          "[ni.analog] failed to parse custom map configuration";
    }
} MapScale;

///////////////////////////////////////////////////////////////////////////////////
//                                  PolynomialScale                              //
///////////////////////////////////////////////////////////////////////////////////
typedef struct PolynomialScale {
    float64 *forward_coeffs;
    float64 *reverse_coeffs;
    uint32_t num_coeffs;
    float64 min_x;
    float64 max_x;
    int32 num_points;
    int32 poly_order;

    PolynomialScale() = default;

    PolynomialScale(config::Parser &parser)
        : num_coeffs(parser.required<int>("num_coeffs")),
          min_x(parser.required<double>("min_x")),
          max_x(parser.required<double>("max_x")),
          num_points(parser.optional<int>("num_reverse_coeffs", 0)),
          poly_order(parser.required<int>("poly_order")) {
        forward_coeffs = new double[num_coeffs];
        reverse_coeffs = new double[num_coeffs];

        if (!parser.ok()) {
            LOG(ERROR) <<
                    "[ni.analog] failed to parse custom polynomial scale configuration";
            forward_coeffs = nullptr;
            reverse_coeffs = nullptr;
            return;
        }

        //TODO: handle if there is reverse coeffs of different size than forward coeffs

        //get json from parser
        json j = parser.get_json();
        // get forward coeffs (prescale -> scale conversions)
        if (!j.contains("coeffs")) {
            return; // TODO: log error
        }
        std::vector<double> forward_coeffs_vec = j["coeffs"].get<std::vector<
            double> >();
        forward_coeffs = new double[num_coeffs];
        // std::memcpy(forward_coeffs, other.forward_coeffs, num_coeffs * sizeof(double)); do this instead?
        for (int i = 0; i < forward_coeffs_vec.size(); i++) {
            forward_coeffs[i] = forward_coeffs_vec[i];
        }
        // get reverse coeffs (scale -> prescale conversions)
        reverse_coeffs = new double[num_coeffs];
        // TODO: reverse coeffs might be smaller than forward_coeffs
        ni::NiDAQmxInterface::CalculateReversePolyCoeff(
            forward_coeffs,
            num_coeffs,
            min_x,
            max_x,
            num_coeffs,
            -1,
            reverse_coeffs
        ); // FIXME: reversePoly order should be user inputted?
    }

    ~PolynomialScale() {
        if (forward_coeffs != nullptr) delete[] forward_coeffs;
        if (reverse_coeffs != nullptr) delete[] reverse_coeffs;
    }
} PolynomialScale;

///////////////////////////////////////////////////////////////////////////////////
//                                    TableScale                                 //
///////////////////////////////////////////////////////////////////////////////////
typedef struct TableScale {
    float64 *prescaled;
    float64 *scaled;
    uint32_t num_points;

    TableScale() = default;

    TableScale(config::Parser &parser)
        : num_points(parser.required<int>("num_points")) {
        if (!parser.ok()) {
            LOG(ERROR) << "[ni.analog] failed to parse custom table configuration";
            prescaled = nullptr;
            scaled = nullptr;
            return;
        }

        //get json from parser
        json j = parser.get_json();
        if (!j.contains("pre_scaled_vals") || !j.contains("scaled_vals")) {
            LOG(ERROR)
                    << "[ni.analog] failed to parse custom table configuration: missing pre_scaled_vals or scaled_vals";
            return; // TODO: log error
        }
        std::vector<double> prescaled_vec = j["pre_scaled_vals"].get<std::vector<
            double> >();
        std::vector<double> scaled_vec = j["scaled_vals"].get<std::vector<double> >();

        prescaled = new double[num_points];
        scaled = new double[num_points];

        for (int i = 0; i < prescaled_vec.size(); i++) {
            prescaled[i] = prescaled_vec[i];
            scaled[i] = scaled_vec[i];
        }
    }

    ~TableScale() {
        if (prescaled != nullptr) delete[] prescaled;
        if (scaled != nullptr) delete[] scaled;
    }
} TableScale;

///////////////////////////////////////////////////////////////////////////////////
//                                    ScaleConfig                                //
//////////////////////////////////////////////////////////////////////////////////
typedef union Scale {
    LinearScale linear;
    MapScale map;
    PolynomialScale polynomial;
    TableScale table;

    // Destructor
    Scale() {
    }

    ~Scale() {
    }
} Scale;

typedef struct ScaleConfig {
    std::string name;
    std::string type;
    std::string prescaled_units;
    std::string scaled_units;
    config::Parser parser;
    Scale scale;

    ScaleConfig() = default;

    // Constructor
    ScaleConfig(config::Parser &parser, std::string &name)
        : name(name),
          type(parser.required<std::string>("type")),
          prescaled_units(parser.optional<std::string>("pre_scaled_units", "")),
          scaled_units(parser.optional<std::string>("scaled_units", "")),
          parser(parser) {
        if (!parser.ok()) {
            LOG(ERROR) << "[ni.analog] failed to parse custom scale configuration for "
                    << name;
            return;
        }
        if (type == "linear") {
            scale.linear = LinearScale(parser);
        } else if (type == "map") {
            scale.map = MapScale(parser);
        } else if (type == "polynomial") {
            scale.polynomial = PolynomialScale(parser);
        } else if (type == "table") {
            scale.table = TableScale(parser);
        }
    }

    // copy constructor
    ScaleConfig(const ScaleConfig &other)
        : name(other.name),
          type(other.type),
          prescaled_units(other.prescaled_units),
          scaled_units(other.scaled_units),
          parser(other.parser) {
        if (type == "linear") {
            scale.linear = LinearScale(parser);
        } else if (type == "map") {
            scale.map = MapScale(parser);
        } else if (type == "polynomial") {
            scale.polynomial = PolynomialScale(parser);
        } else if (type == "table") {
            scale.table = TableScale(parser);
        }
    }

    // copy assignment operator
    ScaleConfig &operator=(const ScaleConfig &other) {
        if (this == &other) return *this;

        name = other.name;
        type = other.type;
        prescaled_units = other.prescaled_units;
        scaled_units = other.scaled_units;
        parser = other.parser;
        if (type == "linear") {
            scale.linear = LinearScale(parser);
        } else if (type == "map") {
            scale.map = MapScale(parser);
        } else if (type == "polynomial") {
            scale.polynomial = PolynomialScale(parser);
        } else if (type == "table") {
            scale.table = TableScale(parser);
        }
        return *this;
    }

    // move constructor
    ScaleConfig(ScaleConfig &&other) = delete;

    // create NI Scale
    int32 createNIScale() {
        if (type == "linear") {
            return ni::NiDAQmxInterface::CreateLinScale(
                name.c_str(),
                scale.linear.slope,
                scale.linear.offset,
                ni::UNITS_MAP.at(prescaled_units),
                scaled_units.c_str()
            );
        } else if (type == "map") {
            return ni::NiDAQmxInterface::CreateMapScale(
                name.c_str(),
                scale.map.prescaled_min,
                scale.map.prescaled_max,
                scale.map.scaled_min,
                scale.map.scaled_max,
                ni::UNITS_MAP.at(prescaled_units),
                scaled_units.c_str()
            );
        } else if (type == "polynomial") {
            float64 forward_coeffs_in[1000];
            float64 reverse_coeffs_in[1000];
            for (int i = 0; i < scale.polynomial.num_coeffs; i++) {
                forward_coeffs_in[i] = scale.polynomial.forward_coeffs[i];
                reverse_coeffs_in[i] = scale.polynomial.reverse_coeffs[i];
            }
            return ni::NiDAQmxInterface::CreatePolynomialScale(
                name.c_str(),
                forward_coeffs_in,
                scale.polynomial.num_coeffs,
                reverse_coeffs_in,
                scale.polynomial.num_coeffs,
                ni::UNITS_MAP.at(prescaled_units),
                scaled_units.c_str()
            );
        } else if (type == "table") {
            float64 prescaled_in[10000];
            float64 scaled_in[10000];

            for (int i = 0; i < scale.table.num_points; i++) {
                prescaled_in[i] = scale.table.prescaled[i];
                scaled_in[i] = scale.table.scaled[i];
            }

            return ni::NiDAQmxInterface::CreateTableScale(
                name.c_str(),
                prescaled_in,
                scale.table.num_points,
                scaled_in,
                scale.table.num_points,
                ni::UNITS_MAP.at(prescaled_units),
                scaled_units.c_str()
            );
        }
        return 0;
    }
} ScaleConfig;
};

//TODO: do parser checks all over here
