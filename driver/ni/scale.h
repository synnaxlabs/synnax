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

#include "nidaqmx_api.h"
#include "daqmx.h"
#include "nisyscfg.h"
#include "driver/config/config.h"
#include "nlohmann/json.hpp" 
#include "glog/logging.h"
#include "driver/ni/ni.h"

namespace ni{
    extern const std::map<std::string, int32_t> UNITS_MAP;
    typedef struct LinearScale{
        float64 slope;
        float64 offset;

        LinearScale() = default;


        LinearScale(config::Parser & parser) 
        :   slope(parser.required<double>("slope")),
            offset(parser.required<double>("offset")){}

    } LinearScale;

    typedef struct MapScale{
        float64 prescaled_min;
        float64 prescaled_max;
        float64 scaled_min;
        float64 scaled_max;

        MapScale() = default;

        MapScale(config::Parser & parser) 
        :   prescaled_min(parser.required<double>("prescaled_min")),
            prescaled_max(parser.required<double>("prescaled_max")),
            scaled_min(parser.required<double>("scaled_min")),
            scaled_max(parser.required<double>("scaled_max")){}


    } MapScale;

    typedef struct PolynomialScale{
        float64* forward_coeffs;
        float64* reverse_coeffs; 
        uint32_t num_coeffs;
        float64 min_x;
        float64 max_x;
        int32 num_points;
        int32 poly_order;

        PolynomialScale() = default;

        PolynomialScale(config::Parser & parser) 
        :   num_coeffs(parser.required<int>("num_coeffs")),
            min_x(parser.required<double>("min_x")),
            max_x(parser.required<double>("max_x")),
            num_points(parser.required<int>("num_points")),
            poly_order(parser.required<int>("poly_order")){
            //get json from parser
            json j = parser.get_json();
            // get forward coeffs (prescale -> scale conversions)   
            if(!j.contains("forward_coeffs")){
                return; // TODO: log error
            }
            std::vector<double> forward_coeffs_vec = j["forward_coeffs"]; 
            forward_coeffs = new double[num_coeffs];
            // std::memcpy(forward_coeffs, other.forward_coeffs, num_coeffs * sizeof(double)); do this instead?
            for(int i = 0; i < forward_coeffs_vec.size(); i++){
                forward_coeffs[i] = forward_coeffs_vec[i];
            }
            // get reverse coeffs (scale -> prescale conversions)
            reverse_coeffs = new double[num_coeffs]; // TODO: reverse coeffs might be smaller than forward_coeffs
            ni::NiDAQmxInterface::CalculateReversePolyCoeff(forward_coeffs, num_coeffs, min_x, max_x, num_points, -1,  reverse_coeffs); // FIXME: reversePoly order should be user inputted?
        }

        ~PolynomialScale(){
            delete[] forward_coeffs;
            delete[] reverse_coeffs;
        }

    } PolynomialScale;

    typedef struct TableScale{
        float64* prescaled;
        float64* scaled;
        uint32_t num_points;

        TableScale() = default;

        TableScale(config::Parser & parser) 
        : num_points(parser.required<int>("num_points")){
            //get json from parser
            json j = parser.get_json();
            if(!j.contains("prescaled") || !j.contains("scaled")){
                return; // TODO: log error
            }
            std::vector<double> prescaled_vec = j["prescaled"]; 
            std::vector<double> scaled_vec = j["scaled"]; 
            prescaled = new double[num_points];
            scaled = new double[num_points];
            for(int i = 0; i < prescaled_vec.size(); i++){
                prescaled[i] = prescaled_vec[i];
                scaled[i] = scaled_vec[i];
            }
        }
        ~TableScale(){
            delete[] prescaled;
            delete[] scaled;
        }
    } TableScale;

    typedef union Scale{
        LinearScale linear;
        MapScale map;
        PolynomialScale polynomial;
        TableScale table;
        // Destructor
        Scale() {}
        ~Scale() {} 
    } Scale;

    typedef struct ScaleConfig{
        std::string name;
        std::string type;
        std::string prescaled_units;
        std::string scaled_units;
        config::Parser parser;
        Scale scale;

        ScaleConfig() = default;

        ScaleConfig(config::Parser & parser) 
        :  type(parser.required<std::string>("type")),
            prescaled_units(parser.optional<std::string>("pre_scaled_units", "")),
            scaled_units(parser.optional<std::string>("scaled_units", "")),
            parser(parser){
            if(type == "linear"){
                scale.linear = LinearScale(parser);
            } else if(type == "map"){
                scale.map = MapScale(parser);
            } else if(type == "polynomial"){
                scale.polynomial = PolynomialScale(parser);
            } else if(type == "table"){
                scale.table = TableScale(parser);
            } else{
                LOG(ERROR) << "failed to parse custom scale configuration for";
            }
        }

        // copy constructor
        ScaleConfig(const ScaleConfig & other) 
        : name(other.name),
          type(other.type),
          prescaled_units(other.prescaled_units),
          scaled_units(other.scaled_units),
          parser(other.parser){
            if(type == "linear"){
                scale.linear = LinearScale(parser);
            } else if(type == "map"){
                scale.map = MapScale(parser);
            } else if(type == "polynomial"){
                scale.polynomial = PolynomialScale(parser);
            } else if(type == "table"){
                scale.table = TableScale(parser);
            } 
        } 

        // copy assignment operator
        ScaleConfig & operator=(const ScaleConfig & other){
            if(this == &other) return *this;
            
            name = other.name;
            type = other.type;
            prescaled_units = other.prescaled_units;
            scaled_units = other.scaled_units;
            parser = other.parser;
            if(type == "linear"){
                scale.linear = LinearScale(parser);
            } else if(type == "map"){
                scale.map = MapScale(parser);
            } else if(type == "polynomial"){
                scale.polynomial = PolynomialScale(parser);
            } else if(type == "table"){
                scale.table = TableScale(parser);
            } 
            return *this;
        }

        // move constructor
        ScaleConfig(ScaleConfig && other)  = delete;
    } ScaleConfig;
};

//TODO: do parser checks all over here