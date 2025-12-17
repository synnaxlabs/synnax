// Copyright 2025 Synnax Labs, Inc.
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
#include "x/cpp/xjson/xjson.h"

/// internal
#include "driver/opc/connection/connection.h"
#include "driver/opc/errors/errors.h"
#include "driver/opc/types/types.h"
#include "driver/opc/write_task.h"
#include "driver/pipeline/control.h"
#include "driver/task/common/write_task.h"

namespace opc {
struct OutputChan {
    /// @brief whether output for the channel is enabled.
    const bool enabled;
    /// @brief the OPC UA node id.
    opc::NodeId node;
    /// @brief the corresponding channel key to write the variable for the node
    /// from.
    const synnax::ChannelKey cmd_channel;

    explicit OutputChan(xjson::Parser &parser):
        enabled(parser.field<bool>("enabled", true)),
        node(opc::NodeId::parse("node_id", parser)),
        cmd_channel([&parser] {
            auto ch = parser.field<synnax::ChannelKey>("cmd_channel", 0);
            if (ch == 0) ch = parser.field<synnax::ChannelKey>("channel", 0);
            if (ch == 0) parser.field_err("cmd_channel", "channel must be specified");
            return ch;
        }()) {}
};

struct WriteTaskConfig : common::BaseWriteTaskConfig {
    /// @brief the list of channels to read from the server.
    std::unordered_map<synnax::ChannelKey, std::unique_ptr<OutputChan>> channels;
    /// @brief the config for connecting to the OPC UA server.
    opc::connection::Config connection;

    explicit WriteTaskConfig(
        const std::shared_ptr<synnax::Synnax> &client,
        xjson::Parser &parser
    ):
        common::BaseWriteTaskConfig(parser) {
        parser.iter("channels", [&](xjson::Parser &channel_builder) {
            auto ch = std::make_unique<OutputChan>(channel_builder);
            if (ch->enabled) channels[ch->cmd_channel] = std::move(ch);
        });
        if (this->channels.empty()) {
            parser.field_err("channels", "task must have at least one enabled channel");
            return;
        }
        auto [dev, err] = client->devices.retrieve(this->device_key);
        if (err) {
            parser.field_err("device", "failed to retrieve device: " + err.message());
            return;
        }
        const auto properties = xjson::Parser(dev.properties);
        this->connection = opc::connection::Config(properties.child("connection"));
        if (properties.error())
            parser.field_err("device", properties.error().message());
    }

    [[nodiscard]] std::vector<synnax::ChannelKey> cmd_keys() const {
        std::vector<synnax::ChannelKey> keys;
        keys.reserve(this->channels.size());
        for (const auto &[key, _]: channels)
            keys.push_back(key);
        return keys;
    }

    static std::pair<WriteTaskConfig, xerrors::Error>
    parse(const std::shared_ptr<synnax::Synnax> &client, const synnax::Task &task) {
        auto parser = xjson::Parser(task.config);
        return {WriteTaskConfig(client, parser), parser.error()};
    }
};

class WriteTaskSink final : public common::Sink {
    const WriteTaskConfig cfg;
    std::shared_ptr<opc::connection::Pool> pool;
    opc::connection::Pool::Connection connection;
    opc::WriteRequestBuilder builder;

public:
    WriteTaskSink(std::shared_ptr<opc::connection::Pool> pool, WriteTaskConfig cfg):
        Sink(cfg.cmd_keys()),
        cfg(std::move(cfg)),
        pool(std::move(pool)),
        connection(nullptr, nullptr, "") {}

    xerrors::Error start() override {
        auto [c, err] = pool->acquire(cfg.connection, "[opc.write] ");
        if (err) return err;
        connection = std::move(c);
        return xerrors::NIL;
    }

    xerrors::Error stop() override {
        connection = opc::connection::Pool::Connection(nullptr, nullptr, "");
        return xerrors::NIL;
    }

    xerrors::Error write(const ::telem::Frame &frame) override {
        auto err = this->perform_write(frame);
        if (!err.matches(opc::errors::UNREACHABLE)) return err;
        LOG(
            WARNING
        ) << "[opc.write_task] connection error detected, attempting reconnect: "
          << err;
        this->connection = opc::connection::Pool::Connection(nullptr, nullptr, "");
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
    xerrors::Error perform_write(const ::telem::Frame &frame) {
        if (!this->connection) return opc::errors::NO_CONNECTION;
        this->builder.clear();
        for (const auto &[key, s]: frame) {
            auto it = this->cfg.channels.find(key);
            if (it == this->cfg.channels.end()) continue;
            if (auto err = this->builder.add_value(it->second->node, s); err) {
                LOG(ERROR) << "[opc.write_task] failed to add value: " << err;
                continue;
            }
        }
        if (this->builder.empty()) return xerrors::NIL;
        opc::WriteResponse res(
            UA_Client_Service_write(this->connection.get(), this->builder.build())
        );
        return opc::errors::parse(res.get().responseHeader.serviceResult);
    }
};
}
