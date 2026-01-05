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

using json = nlohmann::json;

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
    virtual xerrors::Error before_all(lua_State *L) { return xerrors::NIL; }

    /// @brief called after the sequence ends. The caller can optionally override
    /// this method to perform any cleanup that is required after the sequence ends.
    ///
    /// In no scenario should the called store the provided lua_State for later use,
    /// as it is not guaranteed to remain valid after this method returns.
    virtual xerrors::Error after_all(lua_State *L) { return xerrors::NIL; }

    /// @brief called before each iteration of the sequence. The caller can
    /// optionally override this method to bind any variables or functions that must
    /// be updated on every loop iteration.
    ///
    /// In no scenario should the called store the provided lua_State for later use,
    /// as it is not guaranteed to remain valid after this method returns.
    virtual xerrors::Error before_next(lua_State *L) { return xerrors::NIL; }

    /// @brief called after each iteration of the sequence. The caller can
    /// optionally override this method to perform any cleanup that is required
    /// after each loop iteration.
    ///
    /// In no scenario should the called store the provided lua_State for later use,
    /// as it is not guaranteed to remain valid after this method returns.
    virtual xerrors::Error after_next(lua_State *L) { return xerrors::NIL; }
};

/// @brief a Plugin implementation that wraps several plugins into a unified
/// interface.
class MultiPlugin final : public Plugin {
    std::vector<std::shared_ptr<Plugin>> plugins;

public:
    explicit MultiPlugin(std::vector<std::shared_ptr<Plugin>> ops):
        plugins(std::move(ops)) {}

    /// @brief implements Plugin::before_all.
    xerrors::Error before_all(lua_State *L) override {
        for (const auto &op: plugins)
            if (auto err = op->before_all(L)) return err;
        return xerrors::NIL;
    }

    /// @brief implements Plugin::after_all.
    xerrors::Error after_all(lua_State *L) override {
        auto err = xerrors::NIL;
        for (const auto &op: plugins)
            if (const auto t_err = op->after_all(L)) err = t_err;
        return err;
    }

    /// @brief implements Plugin::before_next.
    xerrors::Error before_next(lua_State *L) override {
        for (const auto &op: plugins)
            if (auto err = op->before_next(L)) return err;
        return xerrors::NIL;
    }

    /// @brief implements Plugin::after_next.
    xerrors::Error after_next(lua_State *L) override {
        for (const auto &op: plugins)
            if (auto err = op->after_next(L)) return err;
        return xerrors::NIL;
    }
};

/// @brief Sink is used to abstract away the communication of frames to Synnax,
/// mainly for the purposes of mocking the implementation during tests.
class FrameSink {
public:
    virtual ~FrameSink() = default;

    /// @brief writes the frame to the sink.
    virtual xerrors::Error write(telem::Frame &frame) = 0;

    /// @brief sets the authority of the channels being written to.
    virtual xerrors::Error set_authority(
        const std::vector<synnax::ChannelKey> &keys,
        const std::vector<telem::Authority> &authorities
    ) = 0;

    [[nodiscard]] virtual xerrors::Error close() { return xerrors::NIL; }
    [[nodiscard]] virtual xerrors::Error open() { return xerrors::NIL; }
};

/// @brief a FrameSink implementation that writes frames to Synnax.
class SynnaxFrameSink final : public FrameSink {
    /// @brief we store the client so it lets us lazy open a writer only when
    /// needed.
    const std::shared_ptr<synnax::Synnax> client;
    /// @brief the configuration for opening the writer.
    const synnax::WriterConfig cfg;
    /// @brief the current writer to write to.
    std::unique_ptr<synnax::Writer> writer;

public:
    explicit SynnaxFrameSink(
        const std::shared_ptr<synnax::Synnax> &client,
        synnax::WriterConfig cfg
    );

    xerrors::Error write(telem::Frame &frame) override;

    xerrors::Error set_authority(
        const std::vector<synnax::ChannelKey> &keys,
        const std::vector<telem::Authority> &authorities
    ) override;

    [[nodiscard]] xerrors::Error close() override;

    [[nodiscard]] xerrors::Error open() override;
};

/// @brief a plugin implementation that lets the sequence write to Synnax channels.
class ChannelWrite final : public Plugin {
    /// @brief the current output frame to write.
    telem::Frame frame;
    /// @brief the sink to write the frame to. This is typically backed by a Synnax
    /// writer.
    std::shared_ptr<FrameSink> sink;
    /// @brief a map of channel names to info on the channel.
    std::unordered_map<synnax::ChannelKey, synnax::Channel> channels;
    /// @brief a map that allows the user to resolve a channel by its name.
    std::unordered_map<std::string, synnax::ChannelKey> names_to_keys;

public:
    ChannelWrite(
        std::shared_ptr<FrameSink> sink,
        const std::vector<synnax::Channel> &channels
    );

    std::pair<synnax::Channel, bool> resolve(const std::string &name);

    xerrors::Error before_all(lua_State *L) override;

    xerrors::Error after_all(lua_State *L) override;

    xerrors::Error before_next(lua_State *L) override;

    xerrors::Error after_next(lua_State *L) override;
};

struct LatestValue {
    telem::SampleValue value;
    bool changed;
};

/// @brief a plugin implementation that binds global variables containing channel
/// values to the sequence.
class ChannelReceive final : public Plugin {
    /// @brief mutex that locks the sequence variable binding from the value
    /// receiving.
    std::mutex mu;
    /// @brief the pipeline used to manage the lifecycle of the receiver.
    pipeline::Control pipe;
    /// @brief keeps all the latest sample values for the channels.
    std::unordered_map<synnax::ChannelKey, LatestValue> latest_values;
    /// @brief maps channel keys to channels in order to bind variable names
    /// appropriately.
    std::unordered_map<synnax::ChannelKey, synnax::Channel> channels;

    class Sink final : public pipeline::Sink {
        ChannelReceive &receiver;

    public:
        explicit Sink(ChannelReceive &receiver): receiver(receiver) {}

        xerrors::Error write(telem::Frame &frame) override;
    };

public:
    explicit ChannelReceive(
        const std::shared_ptr<synnax::Synnax> &client,
        const std::vector<synnax::Channel> &read_from
    );

    /// @brief alternative constructor that can be used to stub Synnax for test
    /// cases.
    explicit ChannelReceive(
        const std::shared_ptr<pipeline::StreamerFactory> &factory,
        const std::vector<synnax::Channel> &read_from
    );

    xerrors::Error before_all(lua_State *L) override;

    xerrors::Error after_all(lua_State *L) override;

    xerrors::Error before_next(lua_State *L) override;
};

/// @brief a plugin that binds JSON data as global variables to the sequence.
class JSON final : public Plugin {
    const json data;

public:
    explicit JSON(json source_data);

    xerrors::Error before_all(lua_State *L) override;
};

/// @brief a plugin that adds timing utilities to the sequence.
class Time final : public Plugin {
    /// @brief a function that returns the current time.
    const telem::NowFunc now;
    /// @brief the start time for the sequence.
    telem::TimeStamp start_time;
    /// @brief the total elapsed time since the sequence started.
    telem::TimeSpan elapsed;
    /// @brief the current iteration of the sequence. We use an int64 as it's the
    /// highest precision integer lua supports.
    int64_t iteration;

public:
    explicit Time(telem::NowFunc now = telem::TimeStamp::now):
        now(std::move(now)), start_time(0), elapsed(0), iteration(0) {}

    xerrors::Error before_all(lua_State *L) override;

    xerrors::Error before_next(lua_State *L) override;
};
}
