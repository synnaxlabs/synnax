// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// external
#include "open62541/client.h"
#include "open62541/client_config_default.h"
#include "open62541/client_highlevel.h"

/// module
#include "x/cpp/defer/defer.h"
#include "x/cpp/json/json.h"

/// internal
#include "driver/opc/connection/connection.h"
#include "driver/opc/errors/errors.h"
#include "driver/opc/types/types.h"
#include "driver/opc/write_task.h"
#include "driver/pipeline/control.h"
#include "driver/task/common/write_task.h"

namespace driver::opc {
struct OutputChan {
    /// @brief whether output for the channel is enabled.
    const bool enabled;
    /// @brief the OPC UA node id.
    driver::opc::NodeId node;
    /// @brief the corresponding channel key to write the variable for the node
    /// from.
    const synnax::channel::Key cmd_channel;
    /// @brief the channel fetched from the Synnax server. This does not need to
    /// be provided via the JSON configuration.
    synnax::channel::Channel ch;

    explicit OutputChan(x::json::Parser &parser):
        enabled(parser.field<bool>("enabled", true)),
        node(driver::opc::NodeId::parse("node_id", parser)),
        cmd_channel([&parser] {
            auto ch = parser.field<synnax::channel::Key>("cmd_channel", 0);
            if (ch == 0) ch = parser.field<synnax::channel::Key>("channel", 0);
            if (ch == 0) parser.field_err("cmd_channel", "channel must be specified");
            return ch;
        }()) {}
};

struct WriteTaskConfig : driver::task::common::BaseWriteTaskConfig {
    /// @brief the list of channels to read from the server.
    std::unordered_map<synnax::channel::Key, std::unique_ptr<OutputChan>> channels;
    /// @brief the config for connecting to the OPC UA server.
    driver::opc::connection::Config connection;

    explicit WriteTaskConfig(
        const std::shared_ptr<synnax::Synnax> &client,
        x::json::Parser &parser
    ):
        driver::task::common::BaseWriteTaskConfig(parser) {
        parser.iter("channels", [&](x::json::Parser &channel_builder) {
            auto ch = std::make_unique<OutputChan>(channel_builder);
            if (ch->enabled) channels[ch->cmd_channel] = std::move(ch);
        });
        if (this->channels.empty()) {
            parser.field_err("channels", "task must have at least one enabled channel");
            return;
        }

        std::vector<synnax::channel::Key> keys;
        keys.reserve(this->channels.size());
        for (const auto &[key, _]: this->channels)
            keys.push_back(key);
        auto [sy_channels, ch_err] = client->channels.retrieve(keys);
        if (ch_err) {
            parser.field_err(
                "channels",
                "failed to retrieve channels: " + ch_err.message()
            );
            return;
        }
        for (const auto &sy_ch: sy_channels) {
            auto it = this->channels.find(sy_ch.key);
            if (it != this->channels.end()) it->second->ch = sy_ch;
        }
        auto [dev, err] = client->devices.retrieve(this->device_key);
        if (err) {
            parser.field_err("device", "failed to retrieve device: " + err.message());
            return;
        }
        const auto properties = x::json::Parser(dev.properties);
        this->connection = driver::opc::connection::Config(properties.child("connection"));
        if (properties.error())
            parser.field_err("device", properties.error().message());
    }

    [[nodiscard]] std::vector<synnax::channel::Key> cmd_keys() const {
        std::vector<synnax::channel::Key> keys;
        keys.reserve(this->channels.size());
        for (const auto &[key, _]: channels)
            keys.push_back(key);
        return keys;
    }

    static std::pair<WriteTaskConfig, x::errors::Error>
    parse(const std::shared_ptr<synnax::Synnax> &client, const synnax::Task &task) {
        auto parser = x::json::Parser(task.config);
        return {WriteTaskConfig(client, parser), parser.error()};
    }
};

class WriteTaskSink final : public driver::task::common::Sink {
    const WriteTaskConfig cfg;
    std::shared_ptr<driver::opc::connection::Pool> pool;
    driver::opc::connection::Pool::Connection connection;
    driver::opc::WriteRequestBuilder builder;
    std::vector<synnax::channel::Key> written_keys;

public:
    WriteTaskSink(std::shared_ptr<driver::opc::connection::Pool> pool, WriteTaskConfig cfg):
        Sink(cfg.cmd_keys()),
        cfg(std::move(cfg)),
        pool(std::move(pool)),
        connection(nullptr, nullptr, "") {}

    x::errors::Error start() override {
        auto [c, err] = pool->acquire(cfg.connection, "[opc.write] ");
        if (err) return err;
        connection = std::move(c);
        return x::errors::NIL;
    }

    x::errors::Error stop() override {
        connection = driver::opc::connection::Pool::Connection(nullptr, nullptr, "");
        return x::errors::NIL;
    }

    x::errors::Error write(::x::telem::Frame &frame) override {
        auto err = this->perform_write(frame);
        if (!err.matches(driver::opc::errors::UNREACHABLE)) return err;
        LOG(
            WARNING
        ) << "[opc.write_task] connection error detected, attempting reconnect: "
          << err;
        this->connection = driver::opc::connection::Pool::Connection(nullptr, nullptr, "");
        auto [c, conn_err] = this->pool->acquire(this->cfg.connection, "[opc.write] ");
        if (conn_err) {
            LOG(ERROR) << "[opc.write_task] failed to reconnect: " << conn_err;
            return conn_err;
        }
        this->connection = std::move(c);
        LOG(INFO) << "[opc.write_task] reconnected successfully, retrying write";
        return this->perform_write(frame);
    }

private:
    x::errors::Error perform_write(const ::x::telem::Frame &frame) {
        if (!this->connection) return driver::opc::errors::NO_CONNECTION;
        this->builder.clear();
        this->written_keys.clear();
        for (const auto &[key, s]: frame) {
            auto it = this->cfg.channels.find(key);
            if (it == this->cfg.channels.end()) continue;
            if (auto err = this->builder.add_value(it->second->node, s); err) {
                LOG(ERROR) << "[opc.write_task] failed to add value: " << err;
                continue;
            }
            this->written_keys.push_back(key);
        }
        if (this->builder.empty()) return x::errors::NIL;
        driver::opc::WriteResponse res(
            UA_Client_Service_write(this->connection.get(), this->builder.build())
        );
        if (auto err = driver::opc::errors::parse(res.get().responseHeader.serviceResult); err)
            return err;
        for (std::size_t i = 0; i < res.get().resultsSize; ++i) {
            if (auto err = driver::opc::errors::parse(res.get().results[i]); err) {
                const auto &ch = this->cfg.channels.at(this->written_keys[i]);
                return driver::wrap_channel_error(
                    err,
                    ch->ch.name,
                    driver::opc::NodeId::to_string(ch->node.get())
                );
            }
        }
        return x::errors::NIL;
    }
};
}
