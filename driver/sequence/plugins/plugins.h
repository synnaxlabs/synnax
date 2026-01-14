// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <memory>
#include <string>

#include "nlohmann/json.hpp"

extern "C" {
#include <lauxlib.h>
#include <lua.h>
}

#include "driver/pipeline/control.h"

using json = x::json::json;

namespace plugins {
/// @brief an interface that allows for plugins to inject custom functions
/// and variables into a sequence.
class Plugin {
public:
    virtual ~Plugin() = default;

    /// @brief called before the sequence starts. The caller can optionally override
    /// this method to perform any setup that is required before the sequence
    /// starts.
    ///
    /// In no scenario should the called store the provided lua_State for later use,
    /// as it is not guaranteed to remain valid after this method returns.
    virtual x::errors::Error before_all(lua_State *L) { return x::errors::NIL; }

    /// @brief called after the sequence ends. The caller can optionally override
    /// this method to perform any cleanup that is required after the sequence ends.
    ///
    /// In no scenario should the called store the provided lua_State for later use,
    /// as it is not guaranteed to remain valid after this method returns.
    virtual x::errors::Error after_all(lua_State *L) { return x::errors::NIL; }

    /// @brief called before each iteration of the sequence. The caller can
    /// optionally override this method to bind any variables or functions that must
    /// be updated on every loop iteration.
    ///
    /// In no scenario should the called store the provided lua_State for later use,
    /// as it is not guaranteed to remain valid after this method returns.
    virtual x::errors::Error before_next(lua_State *L) { return x::errors::NIL; }

    /// @brief called after each iteration of the sequence. The caller can
    /// optionally override this method to perform any cleanup that is required
    /// after each loop iteration.
    ///
    /// In no scenario should the called store the provided lua_State for later use,
    /// as it is not guaranteed to remain valid after this method returns.
    virtual x::errors::Error after_next(lua_State *L) { return x::errors::NIL; }
};

/// @brief a Plugin implementation that wraps several plugins into a unified
/// interface.
class MultiPlugin final : public Plugin {
    std::vector<std::shared_ptr<Plugin>> plugins;

public:
    explicit MultiPlugin(std::vector<std::shared_ptr<Plugin>> ops):
        plugins(std::move(ops)) {}

    /// @brief implements Plugin::before_all.
    x::errors::Error before_all(lua_State *L) override {
        for (const auto &op: plugins)
            if (auto err = op->before_all(L)) return err;
        return x::errors::NIL;
    }

    /// @brief implements Plugin::after_all.
    x::errors::Error after_all(lua_State *L) override {
        auto err = x::errors::NIL;
        for (const auto &op: plugins)
            if (const auto t_err = op->after_all(L)) err = t_err;
        return err;
    }

    /// @brief implements Plugin::before_next.
    x::errors::Error before_next(lua_State *L) override {
        for (const auto &op: plugins)
            if (auto err = op->before_next(L)) return err;
        return x::errors::NIL;
    }

    /// @brief implements Plugin::after_next.
    x::errors::Error after_next(lua_State *L) override {
        for (const auto &op: plugins)
            if (auto err = op->after_next(L)) return err;
        return x::errors::NIL;
    }
};

/// @brief Sink is used to abstract away the communication of frames to Synnax,
/// mainly for the purposes of mocking the implementation during tests.
class FrameSink {
public:
    virtual ~FrameSink() = default;

    /// @brief writes the frame to the sink.
    virtual x::errors::Error write(x::telem::Frame &frame) = 0;

    /// @brief sets the authority of the channels being written to.
    virtual x::errors::Error set_authority(
        const std::vector<synnax::channel::Key> &keys,
        const std::vector<x::control::Authority> &authorities
    ) = 0;

    [[nodiscard]] virtual x::errors::Error close() { return x::errors::NIL; }
    [[nodiscard]] virtual x::errors::Error open() { return x::errors::NIL; }
};

/// @brief a FrameSink implementation that writes frames to Synnax.
class SynnaxFrameSink final : public FrameSink {
    /// @brief we store the client so it lets us lazy open a writer only when
    /// needed.
    const std::shared_ptr<synnax::Synnax> client;
    /// @brief the configuration for opening the writer.
    const synnax::framer::WriterConfig cfg;
    /// @brief the current writer to write to.
    std::unique_ptr<synnax::framer::Writer> writer;

public:
    explicit SynnaxFrameSink(
        const std::shared_ptr<synnax::Synnax> &client,
        synnax::framer::WriterConfig cfg
    );

    x::errors::Error write(x::telem::Frame &frame) override;

    x::errors::Error set_authority(
        const std::vector<synnax::channel::Key> &keys,
        const std::vector<x::control::Authority> &authorities
    ) override;

    [[nodiscard]] x::errors::Error close() override;

    [[nodiscard]] x::errors::Error open() override;
};

/// @brief a plugin implementation that lets the sequence write to Synnax channels.
class ChannelWrite final : public Plugin {
    /// @brief the current output frame to write.
    x::telem::Frame frame;
    /// @brief the sink to write the frame to. This is typically backed by a Synnax
    /// writer.
    std::shared_ptr<FrameSink> sink;
    /// @brief a map of channel names to info on the channel.
    std::unordered_map<synnax::channel::Key, synnax::channel::Channel> channels;
    /// @brief a map that allows the user to resolve a channel by its name.
    std::unordered_map<std::string, synnax::channel::Key> names_to_keys;

public:
    ChannelWrite(
        std::shared_ptr<FrameSink> sink,
        const std::vector<synnax::channel::Channel> &channels
    );

    std::pair<synnax::channel::Channel, bool> resolve(const std::string &name);

    x::errors::Error before_all(lua_State *L) override;

    x::errors::Error after_all(lua_State *L) override;

    x::errors::Error before_next(lua_State *L) override;

    x::errors::Error after_next(lua_State *L) override;
};

struct LatestValue {
    x::telem::SampleValue value;
    bool changed;
};

/// @brief a plugin implementation that binds global variables containing channel
/// values to the sequence.
class ChannelReceive final : public Plugin {
    /// @brief mutex that locks the sequence variable binding from the value
    /// receiving.
    std::mutex mu;
    /// @brief the pipeline used to manage the lifecycle of the receiver.
    driver::pipeline::Control pipe;
    /// @brief keeps all the latest sample values for the channels.
    std::unordered_map<synnax::channel::Key, LatestValue> latest_values;
    /// @brief maps channel keys to channels in order to bind variable names
    /// appropriately.
    std::unordered_map<synnax::channel::Key, synnax::channel::Channel> channels;

    class Sink final : public driver::pipeline::Sink {
        ChannelReceive &receiver;

    public:
        explicit Sink(ChannelReceive &receiver): receiver(receiver) {}

        x::errors::Error write(x::telem::Frame &frame) override;
    };

public:
    explicit ChannelReceive(
        const std::shared_ptr<synnax::Synnax> &client,
        const std::vector<synnax::channel::Channel> &read_from
    );

    /// @brief alternative constructor that can be used to stub Synnax for test
    /// cases.
    explicit ChannelReceive(
        const std::shared_ptr<driver::pipeline::StreamerFactory> &factory,
        const std::vector<synnax::channel::Channel> &read_from
    );

    x::errors::Error before_all(lua_State *L) override;

    x::errors::Error after_all(lua_State *L) override;

    x::errors::Error before_next(lua_State *L) override;
};

/// @brief a plugin that binds JSON data as global variables to the sequence.
class JSON final : public Plugin {
    const json data;

public:
    explicit JSON(json source_data);

    x::errors::Error before_all(lua_State *L) override;
};

/// @brief a plugin that adds timing utilities to the sequence.
class Time final : public Plugin {
    /// @brief a function that returns the current time.
    const x::telem::NowFunc now;
    /// @brief the start time for the sequence.
    x::telem::TimeStamp start_time;
    /// @brief the total elapsed time since the sequence started.
    x::telem::TimeSpan elapsed;
    /// @brief the current iteration of the sequence. We use an int64 as it's the
    /// highest precision integer lua supports.
    int64_t iteration;

public:
    explicit Time(x::telem::NowFunc now = x::telem::TimeStamp::now):
        now(std::move(now)), start_time(0), elapsed(0), iteration(0) {}

    x::errors::Error before_all(lua_State *L) override;

    x::errors::Error before_next(lua_State *L) override;
};
}
