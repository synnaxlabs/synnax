// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/sequence/plugins/plugins.h"

/// @brief binds the sample value to the lua state as global variable.
void apply(
    lua_State *L,
    const std::string &name,
    const telem::SampleValue &value
) {
    switch (value.index()) {
        case 0: // float64
            lua_pushnumber(L, std::get<double>(value));
            break;
        case 1: // float32
            lua_pushnumber(L, std::get<float>(value));
            break;
        case 2: // int64
            lua_pushinteger(L, std::get<int64_t>(value));
            break;
        case 3: // int32
            lua_pushinteger(L, std::get<int32_t>(value));
            break;
        case 4: // int16
            lua_pushinteger(L, std::get<int16_t>(value));
            break;
        case 5: // int8
            lua_pushinteger(L, std::get<int8_t>(value));
            break;
        case 6: // uint64
            lua_pushinteger(L, std::get<uint64_t>(value));
            break;
        case 7: // uint32
            lua_pushinteger(L, std::get<uint32_t>(value));
            break;
        case 8: // uint16
            lua_pushinteger(L, std::get<uint16_t>(value));
            break;
        case 9: // uint8
            lua_pushinteger(L, std::get<uint8_t>(value));
            break;
        case 10: // string
            lua_pushstring(L, std::get<std::string>(value).c_str());
            break;
        default: ;
    }
    lua_setglobal(L, name.c_str());
}


plugins::ChannelReceive::ChannelReceive(
    const std::shared_ptr<pipeline::StreamerFactory> &factory,
    const std::vector<synnax::Channel> &read_from
) :
    pipe(
        factory,
        synnax::StreamerConfig{
            .channels = [&read_from] {
                std::vector<synnax::ChannelKey> keys;
                keys.reserve(read_from.size());
                for (const auto &ch: read_from) keys.push_back(ch.key);
                return keys;
            }()
        },
        std::make_shared<Sink>(Sink(*this)),
        breaker::Config{}
    ),
    latest_values(read_from.size()) {
    for (const auto &channel: read_from) this->channels[channel.key] = channel;
}

plugins::ChannelReceive::ChannelReceive(
    const std::shared_ptr<synnax::Synnax> &client,
    const std::vector<synnax::Channel> &read_from
) :
    ChannelReceive(
        std::make_shared<pipeline::SynnaxStreamerFactory>(client),
        read_from
    ) {
}

/// @brief implements plugins::Plugin to start receiving values from the read pipeline.
xerrors::Error plugins::ChannelReceive::before_all(lua_State *L) {
    this->pipe.start();
    return xerrors::NIL;
}

/// @brief implements plugins::Plugin to start receiving values from the write pipeline.
xerrors::Error plugins::ChannelReceive::after_all(lua_State *L) {
    this->pipe.stop();
    return xerrors::NIL;
}

/// @brief implements pipeline::Sink to receive values from a streamer and bind them
/// into the latest values state.
xerrors::Error plugins::ChannelReceive::Sink::write(const synnax::Frame &frame) {
    std::lock_guard lock(this->receiver.mu);
    for (int i = 0; i < frame.size(); i++) {
        const auto key = frame.channels->at(i);
        this->receiver.latest_values[key] = frame.series->at(i).at(-1);
    }
    return xerrors::NIL;
}

/// @brief implements plugins::Plugin to bind the latest values to the lua state
/// on every sequence iteration.
xerrors::Error plugins::ChannelReceive::before_next(lua_State *L) {
    std::lock_guard lock(this->mu);
    for (const auto &[key, value]: this->latest_values) {
        const auto ch = this->channels.at(key);
        apply(L, ch.name, value);
    }
    return xerrors::NIL;
}
