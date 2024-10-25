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

#include "client/cpp/synnax.h"

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
    class TareMiddleware : public Middleware {
    public:
        explicit TareMiddleware(std::vector<synnax::ChannelKey> keys) {
            for (auto &key: keys) {
                tare_values[key] = 0.0;
            }
        }
        // setting unladen value to subtract
        void set_tare_value(synnax::ChannelKey key, double value) {
            tare_values[key] = value;
        }

        void clear(){
            for(auto &pair: tare_values){
                pair.second = 0.0;
            }
        }

        bool handle(Frame &frame) override {
            for(size_t i = 0; i < frame.channels->size(); i++){
                auto channel_key = frame.channels->at(i);

                auto it = tare_values.find(channel_key);
                double tare = 0.0;
                if(it != tare_values.end())
                   tare = it->second;
                else continue;

                auto &series = frame.series->at(i);

                if(series.data_type == synnax::FLOAT64){
                    series.transform_inplace<double>(
                        [tare](double val) {return val - static_cast<double>(tare); }
                    );
                } else if (series.data_type == synnax::FLOAT32){
                    series.transform_inplace<float>(
                        [tare](float val) {return val - static_cast<float>(tare); }
                    );
                }
            } // for
            return true;
        } // handle

    private:
        std::map<synnax::ChannelKey, double> tare_values;
    }; // class TareMiddleware
} // namespace pipeline