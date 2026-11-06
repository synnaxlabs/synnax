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
#include <optional>
#include <string>
#include <unordered_map>
#include <utility>
#include <vector>

#include "nlohmann/json.hpp"
#include "open62541/types.h"

#include "client/cpp/synnax.h"
#include "x/cpp/json/json.h"
#include "x/cpp/telem/telem.h"

#include "driver/common/scan_task.h"
#include "driver/opcua/certificate/store.h"
#include "driver/opcua/connection/connection.h"
#include "driver/opcua/opcua.h"
#include "driver/opcua/structure/structure.h"
#include "driver/opcua/types/types.h"
#include "driver/task/task.h"

namespace driver::opcua {
inline const std::string SCAN_LOG_PREFIX = "[" + INTEGRATION_NAME + ".scan_task] ";
/// @brief Configuration for the OPC UA scanner.
struct ScanTaskConfig : common::ScanTaskConfig {
    ScanTaskConfig() = default;
    explicit ScanTaskConfig(x::json::Parser &cfg): common::ScanTaskConfig(cfg) {}
};

///@brief The parameters for connecting to and iterating through nodes in the OPC UA
/// server.
struct ScanCommandArgs {
    connection::Config connection;
    std::string node_id;
    types::NodeId node;
    /// @brief the member of the node's structure value whose members to list, or empty
    /// to list the children of the node.
    structure::Path field;

    explicit ScanCommandArgs(x::json::Parser &parser):
        connection(opcua::connection::Config(parser.child("connection"))),
        node_id(parser.field<std::string>("node_id", "")) {
        if (node_id.empty())
            node = types::NodeId(UA_NODEID_NUMERIC(0, UA_NS0ID_OBJECTSFOLDER));
        else
            node = types::NodeId::parse("node_id", parser);
        const auto field = parser.field<std::string>("field", "");
        if (field.empty()) return;
        auto [path, err] = structure::parse(field);
        if (err) parser.field_err("field", err.data);
        this->field = std::move(path);
    }
};

const std::string BROWSE_CMD_TYPE = "browse";
/// @brief lists the variables under a node at any depth, for adding a folder to a task.
const std::string BROWSE_VARIABLES_CMD_TYPE = "browse_variables";
/// @brief reads the attributes and the value of one node.
const std::string READ_NODE_CMD_TYPE = "read_node";
/// @brief the most variables that a browse_variables command returns.
constexpr size_t MAX_VARIABLES = 1000;
/// @brief the most objects that a browse_variables command browses.
constexpr size_t MAX_OBJECTS = 10000;
const std::string TEST_CONNECTION_CMD_TYPE = "test_connection";
const std::string GET_ENDPOINTS_CMD_TYPE = "get_endpoints";
const std::string GET_CERTIFICATE_CMD_TYPE = "get_certificate";
const std::string REGENERATE_CERTIFICATE_CMD_TYPE = "regenerate_certificate";
/// @brief how often the Scanner offers the next Driver certificate to each server.
inline const x::telem::TimeSpan RENEWAL_ATTEMPT_INTERVAL = x::telem::DAY;

/// @brief OPC UA scanner implementing the common::Scanner interface.
/// Handles device health monitoring, node browsing, server certificate pinning, and
/// renewal of the Driver certificate.
class Scanner final : public common::Scanner {
public:
    Scanner(
        std::shared_ptr<task::Context> ctx,
        synnax::task::Task task,
        std::shared_ptr<connection::Pool> conn_pool,
        std::shared_ptr<certificate::Store> store
    );

    /// @brief Returns scanner configuration for common::ScanTask.
    [[nodiscard]] common::ScannerConfig config() const override;

    /// @brief Periodic scan method - checks health of all tracked devices.
    std::pair<std::vector<synnax::device::Device>, x::errors::Error>
    scan(const common::ScannerContext &scan_ctx) override;

    /// @brief Handle OPC-specific commands (scan nodes, test connection).
    bool exec(
        synnax::task::Command &cmd,
        const synnax::task::Task &task,
        const std::shared_ptr<task::Context> &ctx
    ) override;

private:
    /// @brief the progress of offering one next Driver certificate to the servers.
    struct Renewal {
        /// @brief the thumbprint of the next certificate.
        std::string thumbprint;
        x::telem::TimeStamp last_attempt;
        /// @brief whether each secured device's server accepted the next certificate
        /// on the last attempt, by device key. A device that failed for another reason,
        /// such as an unreachable server, has no entry.
        std::unordered_map<std::string, bool> accepted;
    };

    std::shared_ptr<task::Context> ctx;
    synnax::task::Task task;
    std::shared_ptr<connection::Pool> conn_pool;
    std::shared_ptr<certificate::Store> store;
    ScanTaskConfig cfg;
    /// @brief only scan() reads or writes it, since commands run on another thread.
    Renewal renewal;
    /// @brief the time of the last health check of each device, by device key. Only
    /// scan() reads or writes it.
    std::unordered_map<std::string, x::telem::TimeStamp> checked;

    /// @brief connects to the server that a browse command names and finds its node.
    /// When a step fails, sets the command's status with the failure message.
    /// @returns the connection and the node, which borrows from args, or nothing after
    /// a failure.
    std::optional<std::pair<connection::Pool::Connection, UA_NodeId>> target(
        const x::json::Parser &parser,
        const ScanCommandArgs &args,
        synnax::task::Status &status,
        const std::string &failure
    ) const;

    /// @brief Browse child nodes of a given OPC UA node.
    void browse_nodes(const synnax::task::Command &cmd) const;

    /// @brief replies with the variables under the command's node, each with its browse
    /// path from below the node, and whether the search stopped at a limit.
    void browse_variables(const synnax::task::Command &cmd) const;

    /// @brief replies with the attributes and the value of the command's node, or of
    /// the member of its structure value that the command's field names.
    void read_node(const synnax::task::Command &cmd) const;

    /// @brief Test connection to an OPC UA server. Never trusts a server certificate
    /// that the configuration does not name.
    void test_connection(const synnax::task::Command &cmd) const;

    /// @brief replies with the endpoints of the server that the command's connection
    /// names.
    void endpoints(const synnax::task::Command &cmd) const;

    /// @brief replies with the current and next Driver certificates, replacing both
    /// first when regenerate is true.
    void certificates(const synnax::task::Command &cmd, bool regenerate) const;

    /// @brief checks the health of a device. Connects only when no task received a
    /// reply from the server since the last check, or to pin the server certificate of
    /// a secured device that has none.
    [[nodiscard]] x::errors::Error check_device_health(synnax::device::Device &dev);

    /// @brief issues the next Driver certificate when renewal is due, offers it to
    /// each server once a day, and switches to it when every server accepts it.
    /// @returns errors::DRIVER_CERTIFICATE_FAILED if the certificate files cannot be
    /// read or written.
    [[nodiscard]] x::errors::Error
    renew_certificate(const std::vector<synnax::device::Device> &devices);
};
}
