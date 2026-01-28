// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "glog/logging.h"

#include "x/cpp/lua/lua.h"

#include "driver/sequence/plugins/plugins.h"

namespace driver::sequence::plugins {
ChannelReceive::ChannelReceive(
    const std::shared_ptr<driver::pipeline::StreamerFactory> &factory,
    const std::vector<synnax::channel::Channel> &read_from
):
    pipe(
        factory,
        synnax::framer::StreamerConfig{
            .channels = synnax::channel::keys_from_channels(read_from)
        },
        std::make_shared<Sink>(Sink(*this)),
        x::breaker::default_config("sequence.plugins.channel_receive")
    ),
    latest_values(read_from.size()),
    channels(synnax::channel::map_channel_Keys(read_from)) {}

ChannelReceive::ChannelReceive(
    const std::shared_ptr<synnax::Synnax> &client,
    const std::vector<synnax::channel::Channel> &read_from
):
    ChannelReceive(
        std::make_shared<driver::pipeline::SynnaxStreamerFactory>(client),
        read_from
    ) {}

/// @brief implements Plugin to start receiving values from the read pipeline.
x::errors::Error ChannelReceive::before_all(lua_State *L) {
    this->pipe.start();
    return x::errors::NIL;
}

/// @brief implements Plugin to start receiving values from the write pipeline.
x::errors::Error ChannelReceive::after_all(lua_State *L) {
    this->pipe.stop();
    return x::errors::NIL;
}

/// @brief implements driver::pipeline::Sink to receive values from a streamer and bind
/// them into the latest values state.
x::errors::Error ChannelReceive::Sink::write(x::telem::Frame &frame) {
    std::lock_guard lock(this->receiver.mu);
    for (size_t i = 0; i < frame.size(); i++) {
        const auto key = frame.channels->at(i);
        if (!frame.series->at(i).empty())
            this->receiver.latest_values[key] = {frame.series->at(i).at(-1), true};
    }
    return x::errors::NIL;
}

/// @brief implements Plugin to bind the latest values to the lua state
/// on every sequence iteration.
x::errors::Error ChannelReceive::before_next(lua_State *L) {
    std::lock_guard lock(this->mu);
    for (const auto &[key, latest]: this->latest_values) {
        if (!latest.changed) continue;
        const auto res = this->channels.find(key);
        if (res == this->channels.end()) {
            LOG(WARNING) << "[sequence.plugins.channel_receive] received value for "
                            "unknown channel key: "
                         << key;
            continue;
        }
        const auto ch = res->second;
        if (const auto err = x::lua::set_global_sample_value(
                L,
                ch.name,
                ch.data_type,
                latest.value
            )) {
            LOG(WARNING) << "[sequence.plugins.channel_receive] failed to set global "
                            "sample value. using nil instead: "
                         << err;
            }
        this->latest_values[key].changed = false;
    }
    return x::errors::NIL;
}
}