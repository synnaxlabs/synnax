// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstddef>
#include <memory>
#include <string>
#include <unordered_map>
#include <utility>
#include <vector>

#include "absl/log/log.h"
#include "open62541/client.h"
#include "open62541/types.h"

#include "client/cpp/opcua/json.gen.h"
#include "client/cpp/synnax.h"
#include "x/cpp/json/json.h"
#include "x/cpp/strings/strings.h"

#include "driver/common/write_task.h"
#include "driver/errors/errors.h"
#include "driver/opcua/connection/connection.h"
#include "driver/opcua/device/device.h"
#include "driver/opcua/errors/errors.h"
#include "driver/opcua/types/types.h"

namespace driver::opcua {
struct OutputChan {
    /// @brief the OPC UA node id.
    types::NodeId node;
    /// @brief the corresponding channel key to write the variable for the node
    /// from.
    const synnax::channel::Key cmd_channel;
    /// @brief the channel fetched from the Synnax server. This does not need to
    /// be provided via the JSON configuration.
    synnax::channel::Channel ch;

    OutputChan(const ::synnax::opcua::WriteChannel &parsed, x::json::Parser &parser):
        node(types::NodeId::parse("node_id", parser)), cmd_channel(parsed.cmd_channel) {
        if (this->cmd_channel == 0)
            parser.field_err("cmd_channel", "channel must be specified");
    }
};

struct WriteTaskConfig : common::BaseWriteTaskConfig {
    /// @brief the enabled channels to write to the server.
    std::unordered_map<synnax::channel::Key, std::unique_ptr<OutputChan>> channels;

    explicit WriteTaskConfig(
        const std::shared_ptr<synnax::Synnax> &client,
        x::json::Parser &parser
    ):
        common::BaseWriteTaskConfig(parser) {
        parser.iter("channels", [&](x::json::Parser &cp) {
            const auto parsed = ::synnax::opcua::WriteChannel::parse(cp);
            if (parsed.disabled) return;
            channels[parsed.cmd_channel] = std::make_unique<OutputChan>(parsed, cp);
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
            parser.field_err("channels", "failed to retrieve channels: " + ch_err.data);
            return;
        }
        for (const auto &sy_ch: sy_channels) {
            auto it = this->channels.find(sy_ch.key);
            if (it != this->channels.end()) it->second->ch = sy_ch;
        }
        if (const auto err = device::retrieve_connection(*client, this->device).second)
            parser.field_err("device", common::config_err_description(err));
    }

    [[nodiscard]] std::vector<synnax::channel::Key> cmd_keys() const {
        std::vector<synnax::channel::Key> keys;
        keys.reserve(this->channels.size());
        for (const auto &[key, _]: channels)
            keys.push_back(key);
        return keys;
    }

    static std::pair<WriteTaskConfig, x::errors::Error> parse(
        const std::shared_ptr<synnax::Synnax> &client,
        const synnax::task::Task &task
    ) {
        auto parser = x::json::Parser(task.config);
        return {WriteTaskConfig(client, parser), parser.error()};
    }
};

class WriteTaskSink final : public common::Sink {
    const WriteTaskConfig cfg;
    device::Connector connector;
    connection::Pool::Connection connection;
    types::WriteRequestBuilder builder;
    std::vector<synnax::channel::Key> written_keys;

    x::errors::Error connect() {
        auto [conn, err] = this->connector.connect();
        if (err) return err;
        this->connection = std::move(conn);
        return x::errors::NIL;
    }

public:
    WriteTaskSink(
        std::shared_ptr<synnax::Synnax> client,
        std::shared_ptr<connection::Pool> pool,
        WriteTaskConfig cfg
    ):
        Sink(cfg.cmd_keys()),
        cfg(std::move(cfg)),
        connector(std::move(client), std::move(pool), this->cfg.device, "[opc.write] "),
        connection(nullptr, nullptr, "") {}

    /// @brief retrieves the device again, so a changed trusted server certificate takes
    /// effect on restart. A server that is off or refuses Sessions does not stop the
    /// start, since each command connects again.
    x::errors::Error start() override {
        auto [c, err] = this->connector.start();
        if (err.matches(driver::errors::TEMPORARY_HARDWARE_ERROR))
            return x::errors::NIL;
        if (err) return err;
        this->connection = std::move(c);
        return x::errors::NIL;
    }

    x::errors::Error stop() override {
        this->connection = connection::Pool::Connection(nullptr, nullptr, "");
        return x::errors::NIL;
    }

    /// @brief writes the command to the server. A connection that breaks gets one new
    /// connection and one more attempt, so a Session the server closed while idle does
    /// not drop the command.
    x::errors::Error write(::x::telem::Frame &frame) override {
        if (!this->connection)
            if (const auto err = this->connect()) return err;
        auto err = this->perform_write(frame);
        if (!err.matches(errors::UNREACHABLE)) return err;
        LOG(WARNING) << "[opc.write_task] lost the connection, connecting again: "
                     << err;
        this->connection = connection::Pool::Connection(nullptr, nullptr, "");
        if (const auto conn_err = this->connect()) return conn_err;
        return this->perform_write(frame);
    }

private:
    /// @brief sends the built request. When the server rejects the whole request after
    /// the client certificate changed, connects again with the new certificate and
    /// sends it again.
    std::pair<types::WriteResponse, x::errors::Error> send() {
        const auto &req = this->builder.build();
        auto [res, err] = this->connection.write(req);
        if (!err || err.matches(errors::UNREACHABLE)) return {std::move(res), err};
        auto [refreshed, refresh_err] = this->connector.refresh(this->connection);
        if (refresh_err) return {std::move(res), refresh_err};
        if (!refreshed) return {std::move(res), err};
        return this->connection.write(req);
    }

    /// @brief writes the frame's values. A value that the Driver cannot convert, or
    /// that the server rejects, is not written, and the other values still are.
    x::errors::Error perform_write(const ::x::telem::Frame &frame) {
        this->builder.clear();
        this->written_keys.clear();
        std::vector<std::string> problems;
        for (const auto &[key, s]: frame) {
            auto it = this->cfg.channels.find(key);
            if (it == this->cfg.channels.end()) continue;
            if (auto err = this->builder.add_value(it->second->node, s)) {
                problems.push_back(it->second->ch.name + ": " + err.data + ".");
                continue;
            }
            this->written_keys.push_back(key);
        }
        if (!this->builder.empty()) {
            auto [res, send_err] = this->send();
            if (send_err) return send_err;
            for (std::size_t i = 0; i < res.get().resultsSize; ++i) {
                const auto status = res.get().results[i];
                if (!UA_StatusCode_isBad(status)) continue;
                const auto &ch = this->cfg.channels.at(this->written_keys[i]);
                problems.push_back(errors::channel_status(ch->ch.name, status));
            }
        }
        if (problems.empty()) return x::errors::NIL;
        return {errors::COMMAND_REJECTED, x::strings::join(problems, " ")};
    }
};
}
