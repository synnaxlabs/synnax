//
// Created by Emiliano Bonilla on 1/21/25.
//

#pragma once

/// external.
extern "C" {
#include <lua.h>
#include <lualib.h>
#include <lauxlib.h>
}

/// internal.
#include "driver/pipeline/control.h"
#include "driver/sequence/source.h"

inline void apply(lua_State *L, const std::string &name, const synnax::Series &series) {
    if (series.data_type == synnax::FLOAT64) lua_pushnumber(L, series.at<double>(0));
    else if (series.data_type == synnax::FLOAT32) lua_pushnumber(L, series.at<float>(0));
    else if (series.data_type == synnax::INT64) lua_pushinteger(L, series.at<int64_t>(0));
    else if (series.data_type == synnax::INT32) lua_pushinteger(L, series.at<int32_t>(0));
    else if (series.data_type == synnax::INT16) lua_pushinteger(L, series.at<int16_t>(0));
    else if (series.data_type == synnax::INT8) lua_pushinteger(L, series.at<int8_t>(0));
    else if (series.data_type == synnax::UINT64) lua_pushinteger(L, series.at<uint64_t>(0));
    else if (series.data_type == synnax::UINT32) lua_pushinteger(L, series.at<uint32_t>(0));
    else if (series.data_type == synnax::SY_UINT16) lua_pushinteger(L, series.at<uint16_t>(0));
    else if (series.data_type == synnax::SY_UINT8) lua_pushinteger(L, series.at<uint8_t>(0));
    else if (series.data_type == synnax::STRING) lua_pushstring(L, series.at<std::string>(0).c_str());
    lua_setglobal(L, name.c_str());  // Set as global variable instead of table entry
}

class ChannelSource final : public sequence::Source, public pipeline::Sink {
    std::mutex frame_mutex;
    std::unordered_map<synnax::ChannelKey, synnax::Series> latest_values;
    std::unordered_map<synnax::ChannelKey, synnax::Channel> channels;
public:
    explicit ChannelSource(
        const std::unordered_map<synnax::ChannelKey, synnax::Channel> &channels
    ): channels(channels) {
    }
    freighter::Error write(const synnax::Frame &frame) override {
        std::lock_guard lock(this->frame_mutex);
        for (int i = 0; i < frame.size(); i++) {
            const auto key = frame.channels->at(i);
            this->latest_values.emplace(key, std::move(frame.series->at(i)));
        }
        return freighter::NIL;
    }

    freighter::Error bind(lua_State *L) override {
        std::lock_guard<std::mutex> lock(this->frame_mutex);
        for (const auto &[key, series] : this->latest_values)
            apply(L, this->channels[key].name, series);
        return freighter::NIL;
    }
};

