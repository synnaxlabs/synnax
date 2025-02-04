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

inline synnax::Series lua_to_series(lua_State *L, int index, const synnax::Channel &ch) {
    if (ch.data_type == synnax::FLOAT32)
        return synnax::Series(
            static_cast<float>(lua_tonumber(L, index)),
            ch.data_type
        );
    if (ch.data_type == synnax::FLOAT64)
        return synnax::Series(
            lua_tonumber(L, index),
            ch.data_type
        );
    if (ch.data_type == synnax::INT8)
        return synnax::Series(
            static_cast<int8_t>(lua_tonumber(L, index)),
            ch.data_type
        );
    if (ch.data_type == synnax::INT16)
        return synnax::Series(
            static_cast<int16_t>(lua_tonumber(L, index)),
            ch.data_type
        );
    if (ch.data_type == synnax::INT32)
        return synnax::Series(
            static_cast<int32_t>(lua_tonumber(L, index)),
            ch.data_type
        );
    if (ch.data_type == synnax::INT64)
        return synnax::Series(
            lua_tointeger(L, index),
            ch.data_type
        );
    if (ch.data_type == synnax::SY_UINT8)
        return synnax::Series(
            static_cast<uint8_t>(lua_isnumber(L, index) ? 
                lua_tonumber(L, index) : lua_toboolean(L, index)),
            ch.data_type
        );
    if (ch.data_type == synnax::SY_UINT16)
        return synnax::Series(
            static_cast<uint16_t>(lua_tonumber(L, index)),
            ch.data_type
        );
    if (ch.data_type == synnax::UINT32)
        return synnax::Series(
            static_cast<uint32_t>(lua_tonumber(L, index)),
            ch.data_type
        );
    if (ch.data_type == synnax::UINT64)
        return synnax::Series(
            static_cast<uint64_t>(lua_tonumber(L, index)),
            ch.data_type
        );
    if (ch.data_type == synnax::STRING)
        return synnax::Series(
            std::string(lua_tostring(L, index)),
            synnax::STRING
        );
    if (ch.data_type == synnax::FLOAT32)
        return synnax::Series(
            static_cast<float>(lua_tonumber(L, index)),
            ch.data_type
        );
    luaL_error(L, "Unsupported data type for channel %u", ch.key);
    return synnax::Series(synnax::DATA_TYPE_UNKNOWN, 0);
}

class Sink {
public:
    virtual ~Sink() = default;

    virtual freighter::Error write(synnax::Frame &frame) = 0;

    virtual freighter::Error set_authority(const std::vector<synnax::ChannelKey> &keys, const std::vector<synnax::Authority> &authorities) = 0;
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

    freighter::Error set_authority(const std::vector<synnax::ChannelKey> &keys, const std::vector<synnax::Authority> &authorities) override {
        this->writer->set_authority(keys, authorities);
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
            auto value = lua_to_series(L, 2, channel);
            op->frame.emplace(channel.key, std::move(value));
            return 0;
        }, 1);
        lua_setglobal(L, "set");
        
        lua_pushlightuserdata(L, this);
        lua_pushcclosure(L, [](lua_State *L) -> int {
            auto *op = static_cast<ChannelSetOperator *>(
                lua_touserdata(L, lua_upvalueindex(1))
            );
            
            std::vector<synnax::ChannelKey> keys;
            std::vector<synnax::Authority> authorities;

            if (lua_gettop(L) == 1 && lua_isnumber(L, 1)) {
                // set_authority(auth number)
                auto auth = static_cast<synnax::Authority>(lua_tonumber(L, 1));
                for (const auto& [key, _] : op->channels) {
                    keys.push_back(key);
                    authorities.push_back(auth);
                }
            } else if (lua_gettop(L) == 2 && lua_isstring(L, 1) && lua_isnumber(L, 2)) {
                // set_authority(channel_name string, auth number)
                const char* channel_name = lua_tostring(L, 1);
                auto auth = static_cast<synnax::Authority>(lua_tonumber(L, 2));
                
                const auto [channel, err] = op->resolve(channel_name);
                if (err) {
                    luaL_error(L, err.message().c_str());
                    return 0;
                }
                keys.push_back(channel.key);
                authorities.push_back(auth);
            } else if (lua_gettop(L) == 2 && lua_istable(L, 1) && lua_isnumber(L, 2)) {
                // set_authority(channel_names table, auth number)
                auto auth = static_cast<synnax::Authority>(lua_tonumber(L, 2));
                
                lua_pushnil(L);
                while (lua_next(L, 1) != 0) {
                    const char* channel_name = lua_tostring(L, -1);
                    const auto [channel, err] = op->resolve(channel_name);
                    if (err) {
                        luaL_error(L, err.message().c_str());
                        return 0;
                    }
                    keys.push_back(channel.key);
                    authorities.push_back(auth);
                    lua_pop(L, 1);
                }
            } else if (lua_gettop(L) == 1 && lua_istable(L, 1)) {
                // set_authority(authorities table<channel_name, auth>)
                lua_pushnil(L);
                while (lua_next(L, 1) != 0) {
                    const char* channel_name = lua_tostring(L, -2);
                    auto auth = static_cast<synnax::Authority>(lua_tonumber(L, -1));
                    
                    const auto [channel, err] = op->resolve(channel_name);
                    if (err) {
                        luaL_error(L, err.message().c_str());
                        return 0;
                    }
                    keys.push_back(channel.key);
                    authorities.push_back(auth);
                    lua_pop(L, 1);
                }
            } else {
                luaL_error(L, "Invalid arguments for set_authority");
                return 0;
            }

            auto err = op->sink->set_authority(keys, authorities);
            if (err) {
                luaL_error(L, err.message().c_str());
                return 0;
            }
            return 0;
        }, 1);
        lua_setglobal(L, "set_authority");
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
