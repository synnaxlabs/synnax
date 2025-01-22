//
// Created by Emiliano Bonilla on 1/21/25.
//

#pragma once

/// std. lib.
#include <memory>
#include <string>

/// external.
#include "client/cpp/synnax.h"

extern "C" {
#include <lua.h>
#include <lauxlib.h>
}

/// internal.
#include "driver/sequence/operator.h"

inline synnax::Series luaToSeries(lua_State *L, int index, const synnax::Channel &ch) {
    if (ch.data_type == synnax::FLOAT64)
        return synnax::Series(synnax::FLOAT64, lua_tonumber(L, index));
    if (ch.data_type == synnax::INT64)
        return synnax::Series(synnax::INT64, lua_tointeger(L, index));
    if (ch.data_type == synnax::SY_UINT8)
        return synnax::Series(synnax::SY_UINT8,
                              static_cast<uint8_t>(lua_toboolean(L, index)));
    if (ch.data_type == synnax::STRING)
        return synnax::Series(std::string(lua_tostring(L, index)));
    if (ch.data_type == synnax::FLOAT32)
        return synnax::Series(synnax::FLOAT32,
                              static_cast<float>(lua_tonumber(L, index)));

    luaL_error(L, "Unsupported data type for channel %u", ch.key);
    return synnax::Series(synnax::DATA_TYPE_UNKNOWN, 0);
}

class Sink {
public:
    virtual ~Sink() = default;

    virtual freighter::Error write(synnax::Frame &frame) = 0;
};

class SynnaxSink final: public Sink {
private:
    std::unique_ptr<synnax::Writer> writer;

public:
    explicit SynnaxSink(std::unique_ptr<synnax::Writer> writer)
        : writer(std::move(writer)) {
    }

    freighter::Error write(synnax::Frame &frame) override {
        if (const bool ok = this->writer->write(frame); !ok) return this->writer->error();
        return freighter::NIL;
    }
};

class ChannelSetOperator final : public sequence::Operator {
    synnax::Frame frame;
    std::shared_ptr<Sink> sink;
    std::unordered_map<std::string, synnax::Channel> channels;

public:
    ChannelSetOperator(
        std::shared_ptr<Sink> sink,
        const std::unordered_map<std::string, synnax::Channel> &channels
    )
        : sink(std::move(sink))
          , channels(channels)
          , frame(Frame(channels.size())) {
    }

    void bind(lua_State *L) override {
        // Push the SetOperator instance as userdata
        lua_pushlightuserdata(L, this);

        // Create closure with the SetOperator instance as upvalue
        lua_pushcclosure(L, [](lua_State *L) -> int {
            auto *op = static_cast<ChannelSetOperator *>(
                lua_touserdata(L, lua_upvalueindex(1)));

            const char *channel_name = lua_tostring(L, 1);
            auto it = op->channels.find(channel_name);
            if (it == op->channels.end()) {
                luaL_error(L, "Channel '%s' not found", channel_name);
                return 0;
            }
            const synnax::Channel &channel = it->second;
            auto value = luaToSeries(L, 2, channel);
            op->frame.add(channel.key, std::move(value));
            return 0;
        }, 1);

        lua_setglobal(L, "set");
    }

    void next() override {
        this->frame = synnax::Frame(channels.size());
    }

    freighter::Error flush() override {
        return sink->write(frame);
    }
};
