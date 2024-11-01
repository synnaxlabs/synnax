// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <memory>
#include <functional>
#include <vector>
#include <map>
#include <string>
#include <variant>
#include <thread>

#include "client/cpp/synnax.h"
#include "driver/config/config.h"

namespace pipeline {
    ///////////////////////////////////////////////////////////////////////////////////
    //                                    Middleware                                 //
    ///////////////////////////////////////////////////////////////////////////////////
    class Middleware {
    public:
        virtual ~Middleware() = default;
        virtual bool handle(Frame& frame) = 0;
    }; // class Middleware

    ///////////////////////////////////////////////////////////////////////////////////
    //                                  MiddlewareChain                              //
    ///////////////////////////////////////////////////////////////////////////////////
    class MiddlewareChain {
    public:
        void add(std::shared_ptr <Middleware> middleware) {
            middlewares.push_back(middleware);
        }

       bool empty() {
            return middlewares.empty();
        }

        freighter::Error exec(Frame &frame) {
            for (auto &middleware: middlewares) {
                if (!middleware->handle(frame)) {
                    return freighter::Error("Middleware failed");
                }
            }
            return freighter::NIL;
        }

    private:
        std::vector <std::shared_ptr<Middleware>> middlewares;
    }; // class MiddlewareChain

    ///////////////////////////////////////////////////////////////////////////////////
    //                                  TareMiddleware                               //
    ///////////////////////////////////////////////////////////////////////////////////
    //TODO: this needs to be the first middleware in the chain (somehow check/force that)?
    class TareMiddleware : public Middleware {
    public:
        explicit TareMiddleware(std::vector<synnax::ChannelKey> keys) {
            for (auto &key: keys) {
                tare_values[key] = 0.0;
            }
        }

        void tare(json &channels) {
            // if json contains no keys, tare everything
            if(channels.empty()){
                std::lock_guard <std::mutex> lock(mutex);
                for(auto &pair: tare_values){
                   auto it = this->last_raw_value.find(pair.first);
                   if(it != last_raw_value.end())
                       tare_values[pair.first] = it->second;
                }
                return;
            }

            for (auto &channel: channels) {
                auto key = channel.get<int32_t>();
                std::lock_guard <std::mutex> lock(mutex);
                auto it = this->last_raw_value.find(key);
                if(it != last_raw_value.end()){
                    tare_values[key] = it->second;
                }
            }
        }

        void clear(){
            std::lock_guard <std::mutex> lock(mutex);
            for(auto &pair: tare_values)
                pair.second = 0.0;
            for(auto &pair: last_raw_value)
                pair.second = 0.0;
        }

        bool handle(Frame &frame) override {
            for (size_t i = 0; i < frame.channels->size(); i++) {
                auto channel_key = frame.channels->at(i);

                // update last raw value first
                auto &series = frame.series->at(i);
                {
                    std::lock_guard <std::mutex> lock(mutex);
                    if (series.size > 0 && series.data_type == synnax::FLOAT64)
                        last_raw_value[channel_key] = series.at<double>(0);
                    else if (series.size > 0 && series.data_type == synnax::FLOAT32)
                        last_raw_value[channel_key] = static_cast<double>(series.at<float>(0));
                }

                double tare = 0.0;
                {
                    std::lock_guard <std::mutex> lock(mutex);
                    auto it = tare_values.find(channel_key);
                    if (it != tare_values.end())
                        tare = it->second;
                    else continue;
                }

                if (series.data_type == synnax::FLOAT64) {
                    series.transform_inplace<double>(
                            [tare](double val) { return val - static_cast<double>(tare); }
                    );
                } else if (series.data_type == synnax::FLOAT32) {
                    series.transform_inplace<float>(
                            [tare](float val) { return val - static_cast<float>(tare); }
                    );
                }
            }
            return true;
        }

    private:
        std::map<synnax::ChannelKey, double> tare_values; // TODO: gonna need some mutex action for these 2
        std::map<synnax::ChannelKey, double> last_raw_value;
        std::mutex mutex;
    }; // class TareMiddleware

    ///////////////////////////////////////////////////////////////////////////////////
    //                                  Linear Scale                                 //
    ///////////////////////////////////////////////////////////////////////////////////
    struct LinearScale {
        double slope;
        double offset;

        LinearScale() = default;

        explicit LinearScale(
                config::Parser &parser
        ) : slope(parser.required<double>("slope")),
            offset(parser.required<double>("offset")) {
            if (!parser.ok())
                LOG(ERROR) << "[driver] failed to parse custom linear configuration";
        }

        void transform_inplace(Series &series) {
            if(series.data_type == synnax::FLOAT64){
                series.transform_inplace<double>(
                    [this](double val) {
                        return (val * slope + offset);
                    }
                );
            } else if(series.data_type == synnax::FLOAT32){
                series.transform_inplace<float>(
                    [this](float val) {
                        return (val * slope + offset);
                    }
                );
            }
        }
    };

    ///////////////////////////////////////////////////////////////////////////////////
    //                                   Map Scale                                   //
    ///////////////////////////////////////////////////////////////////////////////////
    struct MapScale {
        double prescaled_min;
        double prescaled_max;
        double scaled_min;
        double scaled_max;

        MapScale() = default;

        explicit MapScale(
                config::Parser &parser
        ) : prescaled_min(parser.required<double>("pre_scaled_min")),
            prescaled_max(parser.required<double>("pre_scaled_max")),
            scaled_min(parser.required<double>("scaled_min")),
            scaled_max(parser.required<double>("scaled_max")) {
            if (!parser.ok())
                LOG(ERROR) << "[driver] failed to parse custom linear configuration";
        }

        void transform_inplace(Series &series) {
            if(series.data_type == synnax::FLOAT64){
                series.transform_inplace<double>(
                    [this](double val) {
                        return (val - prescaled_min) / (prescaled_max - prescaled_min) * (scaled_max - scaled_min) + scaled_min;
                    }
                );
            } else if(series.data_type == synnax::FLOAT32){
                series.transform_inplace<float>(
                    [this](float val) {
                        return (val - static_cast<float>(prescaled_min)) / (static_cast<float>(prescaled_max) - static_cast<float>(prescaled_min)) * (static_cast<float>(scaled_max) - static_cast<float>(scaled_min)) + static_cast<float>(scaled_min);
                    }
                );
            }
        }
    };

    ///////////////////////////////////////////////////////////////////////////////////
    //                                  ScaleMiddleware                              //
    ///////////////////////////////////////////////////////////////////////////////////
    class ScaleMiddleware : public Middleware {
    public:
        explicit ScaleMiddleware(
            config::Parser &parser
        ) {
            parser.iter("channels", [this](config::Parser &channel_parser) {
                auto key = channel_parser.required<synnax::ChannelKey>("channel");
//                auto scale_config = channel_parser.optional<json>("custom_scale"); // TODO: see if this works
                if(channel_parser.get_json().contains("scale")){
                    auto scale_config = channel_parser.child("scale");
                    auto type = scale_config.required<std::string>("type");
                    if (type == "linear") {
                        scales[key] = LinearScale(scale_config);
                    } else if (type == "map") {
                        scales[key] = MapScale(scale_config);
                    }
                }
            });
        }

        bool handle(Frame &frame) override {
            for(size_t i = 0; i < frame.channels->size(); i++) {
                auto channel_key = frame.channels->at(i);
                auto it = scales.find(channel_key);
                if(it != scales.end()) {
                    std::visit([&](auto& scale) {
                        scale.transform_inplace(frame.series->at(i));
                    }, it->second);
                }
            }
            return true;
        }

    private:
        std::map<synnax::ChannelKey, std::variant<LinearScale, MapScale>> scales;
    }; // class ScalingMiddleWare
} // namespace pipeline