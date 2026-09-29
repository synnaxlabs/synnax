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
#include <vector>

#include "client/cpp/bus/types.gen.h"
#include "client/cpp/synnax.h"
#include "client/cpp/testutil/testutil.h"
#include "x/cpp/test/test.h"
#include "x/cpp/uuid/uuid.h"

#include "driver/task/task.h"

/// @brief fixtures for bus task tests against a live Core.
namespace driver::bus::testutil {
/// @brief a library, device, and channels created in a live Core for one task.
struct Core {
    std::shared_ptr<synnax::Synnax> client = std::make_shared<synnax::Synnax>(
        new_test_client()
    );
    std::shared_ptr<task::MockContext> ctx = std::make_shared<task::MockContext>(
        this->client
    );
    synnax::rack::Rack rack = ASSERT_NIL_P(this->client->racks.create("bus_test"));

    /// @returns the library holding messages, created in the Core.
    synnax::library::Library
    create_library(std::vector<synnax::library::MessageEntry> messages) const {
        synnax::library::Library lib{.name = "bus_test"};
        for (auto &m: messages)
            lib.entries.emplace_back(std::move(m));
        const auto err = this->client->libraries.create(lib);
        EXPECT_FALSE(err) << err;
        return lib;
    }

    /// @returns the key of a device with the given make and properties, created in
    /// the Core.
    std::string
    create_device(const std::string &make, const x::json::json &props) const {
        synnax::device::Device dev{
            .key = x::uuid::create().to_string(),
            .rack = this->rack.key,
            .location = "bus_test",
            .make = make,
            .model = "bus_test",
            .name = "bus_test",
            .properties = props.get<x::json::json::object_t>(),
        };
        const auto err = this->client->devices.create(dev);
        EXPECT_FALSE(err) << err;
        return dev.key;
    }

    /// @returns a read config over every message and field in lib, with an index and
    /// a float64 channel per field created in the Core.
    ::synnax::bus::ReadConfig
    read_config(const synnax::library::Library &lib, const std::string &device) const {
        ::synnax::bus::ReadConfig cfg;
        cfg.device = device;
        cfg.library = lib.key;
        cfg.data_saving_disabled = true;
        for (const auto &e: lib.entries) {
            const auto *m = std::get_if<synnax::library::MessageEntry>(&e);
            if (m == nullptr) continue;
            const auto idx = ASSERT_NIL_P(this->client->channels.create(
                make_unique_channel_name(m->name + "_time"),
                x::telem::TIMESTAMP_T,
                0,
                true
            ));
            ::synnax::bus::ReadMessage rm{.message = m->key, .index = idx.key};
            for (const auto &f: m->fields) {
                const auto &base = std::visit(
                    [](const auto &v) -> const synnax::library::BaseField & {
                        return v;
                    },
                    f
                );
                const auto ch = ASSERT_NIL_P(this->client->channels.create(
                    make_unique_channel_name(m->name + "_" + base.name),
                    x::telem::FLOAT64_T,
                    idx.key,
                    false
                ));
                rm.fields.push_back({.field = base.key, .channel = ch.key});
            }
            cfg.messages.push_back(rm);
        }
        return cfg;
    }

    /// @returns a task of the given type and config on the rack.
    [[nodiscard]] synnax::task::Task
    task(const std::string &type, const x::json::json &config) const {
        return synnax::task::Task{
            .key = x::uuid::create(),
            .rack = this->rack.key,
            .name = "bus_test",
            .type = type,
            .config = config.get<x::json::json::object_t>(),
        };
    }

    /// @returns a streamer of key.
    [[nodiscard]] synnax::framer::Streamer
    stream(const synnax::channel::Key key) const {
        return ASSERT_NIL_P(this->client->telem.open_streamer({.channels = {key}}));
    }
};

/// @brief sends a command of the given type to t.
inline void exec(task::Task &t, const std::string &type) {
    synnax::task::Command cmd{.type = type, .key = type};
    t.exec(cmd);
}
}
