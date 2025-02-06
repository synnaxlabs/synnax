//
// Created by Emiliano Bonilla on 1/21/25.
//

#pragma once

/// external.
extern "C" {
#include <lua.h>
}

/// internal.
#include "driver/pipeline/control.h"
#include "driver/sequence/operator.h"

inline void apply(lua_State *L, const std::string &name, const synnax::SampleValue &value) {
    switch (value.index()) {
        case 0:  // float64
            lua_pushnumber(L, std::get<double>(value));
            break;
        case 1:  // float32
            lua_pushnumber(L, std::get<float>(value));
            break;
        case 2:  // int64
            lua_pushinteger(L, std::get<int64_t>(value));
            break;
        case 3:  // int32
            lua_pushinteger(L, std::get<int32_t>(value));
            break;
        case 4:  // int16
            lua_pushinteger(L, std::get<int16_t>(value));
            break;
        case 5:  // int8
            lua_pushinteger(L, std::get<int8_t>(value));
            break;
        case 6:  // uint64
            lua_pushinteger(L, std::get<uint64_t>(value));
            break;
        case 7:  // uint32
            lua_pushinteger(L, std::get<uint32_t>(value));
            break;
        case 8:  // uint16
            lua_pushinteger(L, std::get<uint16_t>(value));
            break;
        case 9:  // uint8
            lua_pushinteger(L, std::get<uint8_t>(value));
            break;
        case 10:  // string
            lua_pushstring(L, std::get<std::string>(value).c_str());
            break;
        default: ;
    }
    lua_setglobal(L, name.c_str());
}


class ReceiveChannelValueOperator final : public sequence::Operator, public pipeline::Sink {
    std::mutex frame_mutex;
    std::unordered_map<synnax::ChannelKey, synnax::SampleValue> latest_values;
    std::unordered_map<synnax::ChannelKey, synnax::Channel> channels;
public:
    explicit ReceiveChannelValueOperator(const std::vector<synnax::Channel>& channel_vec) {
        for (const auto& channel : channel_vec)
            this->channels[channel.key] = channel;
        this->latest_values.reserve(channel_vec.size());
    }

    // Implements pipeline::Sink;
    freighter::Error write(const synnax::Frame &frame) override {
        std::lock_guard lock(this->frame_mutex);
        for (int i = 0; i < frame.size(); i++) {
            const auto key = frame.channels->at(i);
            this->latest_values[key] = frame.series->at(i).at(-1);
        }
        return freighter::NIL;
    }

    // Implements sequence::Operator.
    freighter::Error before_next(lua_State *L) override {
        std::lock_guard lock(this->frame_mutex);
        for (const auto &[key, value]: this->latest_values) {
            const auto ch = this->channels.at(key);
            apply(L, ch.name, value);
        }
        return freighter::NIL;
    }
};
