// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/sequence/plugins/plugins.h"

synnax::Series lua_to_series(
    lua_State *L,
    const int index,
    const synnax::Channel &ch
) {
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
            static_cast<uint8_t>(lua_isnumber(L, index)
                                     ? lua_tonumber(L, index)
                                     : lua_toboolean(L, index)),
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


/// @brief an implementation of Sink backed by a Synnax writer.
plugins::SynnaxFrameSink::SynnaxFrameSink(
    const std::shared_ptr<synnax::Synnax> &client,
    synnax::WriterConfig cfg
)
    : client(client), cfg(std::move(cfg)) {
}

freighter::Error plugins::SynnaxFrameSink::write(synnax::Frame &frame) {
    if (frame.empty()) return freighter::NIL;
    if (this->writer == nullptr) {
        auto [w, err] = this->client->telem.open_writer(this->cfg);
        if (err) return err;
        this->writer = std::make_unique<Writer>(std::move(w));
    }
    if (const bool ok = this->writer->write(frame); !ok)
        return this->writer->error();
    return freighter::NIL;
}

freighter::Error plugins::SynnaxFrameSink::set_authority(
    const std::vector<synnax::ChannelKey> &keys,
    const std::vector<synnax::Authority> &authorities
) {
    if (const bool ok = this->writer->set_authority(keys, authorities); !ok)
        return this->writer->error();
    return freighter::NIL;
}

freighter::Error plugins::SynnaxFrameSink::close() {
    if (this->writer == nullptr) return freighter::NIL;
    const auto err = this->writer->close();
    this->writer = nullptr;
    return err;
}

plugins::ChannelWrite::ChannelWrite(
    std::shared_ptr<plugins::FrameSink> sink,
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

/// @brief resolves a channel key by its name.
std::pair<synnax::Channel, freighter::Error> plugins::ChannelWrite::resolve(
    const std::string &name) {
    const auto it = this->names_to_keys.find(name);
    if (it == this->names_to_keys.end())
        return {
            synnax::Channel(),
            freighter::Error(synnax::NOT_FOUND, "Channel" + name + " not found")
        };
    return {this->channels[it->second], freighter::NIL};
}

/// @brief implements sequence::Operator to bind channel set functions to the
/// sequence on startup.
freighter::Error plugins::ChannelWrite::before_all(lua_State *L) {
    // Configuring the "set" closure used to set a channel value.
    lua_pushlightuserdata(L, this);
    lua_pushcclosure(L, [](lua_State *cL) -> int {
        auto *op = static_cast<plugins::ChannelWrite *>(
            lua_touserdata(cL, lua_upvalueindex(1))
        );
        const char *channel_name = lua_tostring(cL, 1);
        const auto [channel, err] = op->resolve(channel_name);
        if (err) {
            luaL_error(cL, err.message().c_str());
            return 0;
        }
        auto value = lua_to_series(cL, 2, channel);
        op->frame.emplace(channel.key, std::move(value));
        return 0;
    }, 1);
    lua_setglobal(L, "set");

    /// Configuring the "set_authority" closure used to change control authority on channels.
    lua_pushlightuserdata(L, this);
    lua_pushcclosure(L, [](lua_State *cL) -> int {
        auto *op = static_cast<plugins::ChannelWrite *>(
            lua_touserdata(cL, lua_upvalueindex(1))
        );

        std::vector<synnax::ChannelKey> keys;
        std::vector<synnax::Authority> authorities;

        // Switching against the various possible overloads.
        if (lua_gettop(cL) == 1 && lua_isnumber(cL, 1)) {
            // set_authority(auth number)
            auto auth = static_cast<synnax::Authority>(lua_tonumber(cL, 1));
            for (const auto &[key, _]: op->channels) {
                keys.push_back(key);
                authorities.push_back(auth);
            }
        } else if (
            lua_gettop(cL) == 2 &&
            lua_isstring(cL, 1) &&
            lua_isnumber(cL, 2)
        ) {
            // set_authority(channel_name string, auth number)
            const char *channel_name = lua_tostring(cL, 1);
            auto auth = static_cast<synnax::Authority>(lua_tonumber(cL, 2));
            const auto [channel, err] = op->resolve(channel_name);
            if (err) {
                luaL_error(cL, err.message().c_str());
                return 0;
            }
            keys.push_back(channel.key);
            authorities.push_back(auth);
        } else if (
            lua_gettop(cL) == 2 &&
            lua_istable(cL, 1) &&
            lua_isnumber(cL, 2)
        ) {
            // set_authority(channel_names table, auth number)
            auto auth = static_cast<synnax::Authority>(lua_tonumber(cL, 2));

            lua_pushnil(cL);
            while (lua_next(cL, 1) != 0) {
                const char *channel_name = lua_tostring(cL, -1);
                const auto [channel, err] = op->resolve(channel_name);
                if (err) {
                    luaL_error(cL, err.message().c_str());
                    return 0;
                }
                keys.push_back(channel.key);
                authorities.push_back(auth);
                lua_pop(cL, 1);
            }
        } else if (lua_gettop(cL) == 1 && lua_istable(cL, 1)) {
            // set_authority(authorities table<channel_name, auth>)
            lua_pushnil(cL);
            while (lua_next(cL, 1) != 0) {
                const char *channel_name = lua_tostring(cL, -2);
                auto auth = static_cast<synnax::Authority>(lua_tonumber(cL, -1));

                const auto [channel, err] = op->resolve(channel_name);
                if (err) {
                    luaL_error(cL, err.message().c_str());
                    return 0;
                }
                keys.push_back(channel.key);
                authorities.push_back(auth);
                lua_pop(cL, 1);
            }
        } else {
            luaL_error(cL, "Invalid arguments for set_authority");
            return 0;
        }

        if (auto err = op->sink->set_authority(keys, authorities)) {
            luaL_error(cL, err.message().c_str());
            return 0;
        }
        return 0;
    }, 1);
    lua_setglobal(L, "set_authority");
    return freighter::NIL;
}

/// @brief implements plugins::Plugin to close the sink after the sequence
/// is complete.
freighter::Error plugins::ChannelWrite::after_all(lua_State *L) {
    return this->sink->close();
}

/// @brief clears out the previous written frame before the next iteration.
freighter::Error plugins::ChannelWrite::before_next(lua_State *_) {
    this->frame = synnax::Frame(channels.size());
    return freighter::NIL;
}

/// @brief writes the frame to the sink after the iteration.
freighter::Error plugins::ChannelWrite::after_next(lua_State *_) {
    if (this->frame.empty()) return freighter::NIL;
    const auto now = synnax::TimeStamp::now();
    std::vector<synnax::ChannelKey> index_keys;
    for (const auto key: *this->frame.channels) {
        auto it = this->channels.find(key);
        if (it == this->channels.end())
            return freighter::Error(synnax::NOT_FOUND, "Channel not found");
        synnax::Channel ch = it->second;
        if (!ch.is_virtual) index_keys.push_back(ch.index);
    }
    for (const auto index: index_keys)
        frame.emplace(index, synnax::Series(now));
    return this->sink->write(this->frame);
}
