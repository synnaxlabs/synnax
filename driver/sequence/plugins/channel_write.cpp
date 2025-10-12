// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/sequence/plugins/plugins.h"
#include "x/cpp/xlua/xlua.h"

/// @brief an implementation of Sink backed by a Synnax writer.
plugins::SynnaxFrameSink::SynnaxFrameSink(
    const std::shared_ptr<synnax::Synnax> &client,
    synnax::WriterConfig cfg
):
    client(client), cfg(std::move(cfg)) {}

xerrors::Error plugins::SynnaxFrameSink::open() {
    if (this->writer != nullptr) return xerrors::NIL;
    auto [w, err] = this->client->telem.open_writer(this->cfg);
    if (err) return err;
    this->writer = std::make_unique<synnax::Writer>(std::move(w));
    return xerrors::NIL;
}

xerrors::Error plugins::SynnaxFrameSink::write(const synnax::Frame &frame) {
    if (frame.empty()) return xerrors::NIL;
    return this->writer->write(frame);
}

xerrors::Error plugins::SynnaxFrameSink::set_authority(
    const std::vector<synnax::ChannelKey> &keys,
    const std::vector<telem::Authority> &authorities
) {
    return this->writer->set_authority(keys, authorities);
}

xerrors::Error plugins::SynnaxFrameSink::close() {
    if (this->writer == nullptr) return xerrors::NIL;
    const auto err = this->writer->close();
    this->writer = nullptr;
    return err;
}

plugins::ChannelWrite::ChannelWrite(
    std::shared_ptr<plugins::FrameSink> sink,
    const std::vector<synnax::Channel> &channels
):
    frame(synnax::Frame(channels.size())),
    sink(std::move(sink)),
    channels(channels.size()),
    names_to_keys(channels.size()) {
    for (const auto &ch: channels) {
        this->channels[ch.key] = ch;
        this->names_to_keys[ch.name] = ch.key;
    }
}

std::pair<synnax::Channel, bool>
plugins::ChannelWrite::resolve(const std::string &name) {
    const auto it = this->names_to_keys.find(name);
    if (it == this->names_to_keys.end()) return {synnax::Channel(), false};
    return {this->channels[it->second], true};
}

/// @brief implements sequence::Operator to bind channel set functions to the
/// sequence on startup.
xerrors::Error plugins::ChannelWrite::before_all(lua_State *L) {
    if (const auto err = this->sink->open()) return err;
    // Configuring the "set" closure used to set a channel value.
    lua_pushlightuserdata(L, this);
    lua_pushcclosure(
        L,
        [](lua_State *cL) -> int {
            auto *op = static_cast<plugins::ChannelWrite *>(
                lua_touserdata(cL, lua_upvalueindex(1))
            );
            const char *channel_name = lua_tostring(cL, 1);
            const auto [channel, found] = op->resolve(channel_name);
            if (!found) {
                lua_pushfstring(cL, "Channel %s not found", channel_name);
                lua_error(cL);
                return 0;
            }

            // Create Series in a nested scope to ensure cleanup before luaL_error
            std::string error_msg;
            bool has_error = false;
            {
                auto [value, s_err] = xlua::to_series(cL, 2, channel.data_type);
                if (s_err) {
                    error_msg = s_err.message();
                    has_error = true;
                    // value is destroyed here when exiting scope
                } else {
                    op->frame.emplace(channel.key, std::move(value));
                }
            } // Series is destroyed here if there was an error

            if (has_error) {
                luaL_error(cL, error_msg.c_str());
                return 0;
            }
            return 0;
        },
        1
    );
    lua_setglobal(L, "set");

    /// Configuring the "set_authority" closure used to change control authority on
    /// channels.
    lua_pushlightuserdata(L, this);
    lua_pushcclosure(
        L,
        [](lua_State *cL) -> int {
            auto *op = static_cast<plugins::ChannelWrite *>(
                lua_touserdata(cL, lua_upvalueindex(1))
            );

            std::vector<synnax::ChannelKey> keys;
            std::vector<telem::Authority> authorities;

            // Switching against the various possible overloads.
            if (lua_gettop(cL) == 1 && lua_isnumber(cL, 1)) {
                // set_authority(auth number)
                auto auth = static_cast<telem::Authority>(lua_tonumber(cL, 1));
                for (const auto &[key, _]: op->channels) {
                    keys.push_back(key);
                    authorities.push_back(auth);
                }
            } else if (lua_gettop(cL) == 2 && lua_isstring(cL, 1) &&
                       lua_isnumber(cL, 2)) {
                // set_authority(channel_name string, auth number)
                const char *channel_name = lua_tostring(cL, 1);
                auto auth = static_cast<telem::Authority>(lua_tonumber(cL, 2));
                const auto [channel, found] = op->resolve(channel_name);
                if (!found) {
                    lua_pushfstring(cL, "Channel %s not found", channel_name);
                    lua_error(cL);
                    return 0;
                }
                keys.push_back(channel.key);
                authorities.push_back(auth);
            } else if (lua_gettop(cL) == 2 && lua_istable(cL, 1) &&
                       lua_isnumber(cL, 2)) {
                // set_authority(channel_names table, auth number)
                auto auth = static_cast<telem::Authority>(lua_tonumber(cL, 2));

                lua_pushnil(cL);
                while (lua_next(cL, 1) != 0) {
                    const char *channel_name = lua_tostring(cL, -1);
                    const auto [channel, found] = op->resolve(channel_name);
                    if (!found) {
                        lua_pushfstring(cL, "Channel %s not found", channel_name);
                        lua_error(cL);
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
                    auto auth = static_cast<telem::Authority>(lua_tonumber(cL, -1));

                    const auto [channel, found] = op->resolve(channel_name);
                    if (!found) {
                        lua_pushfstring(cL, "Channel %s not found", channel_name);
                        lua_error(cL);
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
                lua_pushstring(cL, err.message().c_str());
                lua_error(cL);
                return 0;
            }
            return 0;
        },
        1
    );
    lua_setglobal(L, "set_authority");
    return xerrors::NIL;
}

/// @brief implements plugins::Plugin to close the sink after the sequence
/// is complete.
xerrors::Error plugins::ChannelWrite::after_all(lua_State *L) {
    return this->sink->close();
}

/// @brief clears out the previous written frame before the next iteration.
xerrors::Error plugins::ChannelWrite::before_next(lua_State *L) {
    this->frame.clear();
    this->frame.reserve(this->channels.size());
    return xerrors::NIL;
}

/// @brief writes the frame to the sink after the iteration.
xerrors::Error plugins::ChannelWrite::after_next(lua_State *L) {
    if (this->frame.empty()) return xerrors::NIL;
    const auto now = telem::TimeStamp::now();
    std::vector<synnax::ChannelKey> index_keys;
    for (const auto key: *this->frame.channels) {
        auto it = this->channels.find(key);
        if (it == this->channels.end())
            return xerrors::Error(xerrors::NOT_FOUND, "channel not found");
        synnax::Channel ch = it->second;
        if (!ch.is_virtual) index_keys.push_back(ch.index);
    }
    for (const auto index: index_keys)
        frame.emplace(index, telem::Series(now));
    return this->sink->write(this->frame);
}
