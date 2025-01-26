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
        return synnax::Series(lua_tonumber(L, index), synnax::FLOAT64);
    if (ch.data_type == synnax::INT64)
        return synnax::Series(lua_tointeger(L, index), synnax::INT64);
    if (ch.data_type == synnax::SY_UINT8)
        return synnax::Series(static_cast<uint8_t>(lua_toboolean(L, index)),
                              synnax::SY_UINT8);
    if (ch.data_type == synnax::STRING)
        return synnax::Series(std::string(lua_tostring(L, index)), synnax::STRING);
    if (ch.data_type == synnax::FLOAT32)
        return synnax::Series(static_cast<float>(lua_tonumber(L, index)),
                              synnax::FLOAT32);

    luaL_error(L, "Unsupported data type for channel %u", ch.key);
    return synnax::Series(synnax::DATA_TYPE_UNKNOWN, 0);
}

class Sink {
public:
    virtual ~Sink() = default;

    virtual freighter::Error write(synnax::Frame &frame) = 0;
};

class SynnaxSink final : public Sink {
    const std::shared_ptr<synnax::Synnax> client;
    const synnax::WriterConfig &cfg;
    std::unique_ptr<synnax::Writer> writer;

public:
    explicit SynnaxSink(
        const std::shared_ptr<synnax::Synnax> client,
        const synnax::WriterConfig &cfg
    )
        : client(client), cfg(cfg) {
    }

    freighter::Error write(synnax::Frame &frame) override {
        if (this->writer == nullptr) {
            auto [w, err] = this->client->telem.open_writer(this->cfg);
            if (err) return err;
            this->writer = std::make_unique<Writer>(std::move(w));
        }
        if (const bool ok = this->writer->write(frame); !ok)
            return this->writer->error();
        return freighter::NIL;
    }

    [[nodiscard]] freighter::Error close() const {
        if (this->writer == nullptr) return freighter::NIL;
        return this->writer->close();
    }
};

/// @brief ChannelSetOperator allows the user of a sequence to write values to channels.
/// It binds a "set" method to the lua state of the form `set(channel_name, value)`.
class ChannelSetOperator final : public sequence::Operator {
    /// @brief the current output frame to write.
    synnax::Frame frame;
    /// @brief the sink to write the frame to. This is typically backed by a Synnax
    /// writer.
    std::shared_ptr<Sink> sink;
    /// @brief a map of channel names to info on the channel.
    std::unordered_map<ChannelKey, synnax::Channel> channels;
    std::unordered_map<std::string, ChannelKey> names_to_keys;

public:
    ChannelSetOperator(
        std::shared_ptr<Sink> sink,
        const std::vector<Channel> &channels
    ): frame(Frame(channels.size()))
       , sink(std::move(sink))
       , channels(channels.size())
       , names_to_keys(channels.size()) {
        for (const auto &ch: channels) {
            this->channels[ch.key] = ch;
            this->names_to_keys[ch.name] = ch.key;
        }
    }

    std::pair<synnax::Channel, freighter::Error> resolve(const std::string &name) {
        const auto it = this->names_to_keys.find(name);
        if (it == this->names_to_keys.end())
            return {
                synnax::Channel(),
                freighter::Error(synnax::NOT_FOUND, "Channel" + name + " not found")
            };
        return {this->channels[it->second], freighter::NIL};
    }

    void bind(lua_State *L) override {
        lua_pushlightuserdata(L, this);
        lua_pushcclosure(L, [](lua_State *L) -> int {
            auto *op = static_cast<ChannelSetOperator *>(
                lua_touserdata(L, lua_upvalueindex(1))
            );
            const char *channel_name = lua_tostring(L, 1);
            const auto [channel, err] = op->resolve(channel_name);
            if (err) {
                luaL_error(L, err.message().c_str());
                return 0;
            }
            auto value = luaToSeries(L, 2, channel);
            op->frame.emplace(channel.key, std::move(value));
            return 0;
        }, 1);
        lua_setglobal(L, "set");
    }

    void next() override {
        this->frame = synnax::Frame(channels.size());
    }

    freighter::Error flush() override {
        const auto now = synnax::TimeStamp::now();
        for (const auto key: *this->frame.channels) {
            auto it = this->channels.find(key);
            if (it == this->channels.end())
                return freighter::Error(synnax::NOT_FOUND, "Channel not found");
            synnax::Channel ch = it->second;
            if (ch.is_virtual) continue;
            frame.emplace(ch.index, std::move(synnax::Series(now)));
        }
        return this->sink->write(this->frame);
    }
};
