// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// std
#include <string>
#include <regex>

/// external
#include "open62541/types.h"
#include "open62541/client.h"
#include "nlohmann/json.hpp"

/// module
#include "x/cpp/telem/telem.h"
#include "x/cpp/telem/series.h"
#include "x/cpp/xjson/xjson.h"

/// internal
#include "driver/errors/errors.h"

using json = nlohmann::json;

namespace util {
/// @brief the configuration for an OPC UA connection.
struct ConnectionConfig {
    /// @brief the endpoint of the OPC UA server.
    std::string endpoint;
    /// @brief the username to use for authentication. Not required.
    std::string username;
    /// @brief the password to use for authentication. Not required.
    std::string password;
    /// @brief the security mode.
    std::string security_mode = "None";
    /// @brief the security policy.
    std::string security_policy = "None";
    /// @brief the client certificate used to sign and encrypt messages. Only required
    /// if the security policy is not "None".
    std::string client_cert;
    /// @brief the client private key used to sign and encrypt messages. Only required
    /// if the security policy is not "None".
    std::string client_private_key;
    /// @brief a trusted server certificate. Only req
    std::string server_cert;

    ConnectionConfig() = default;

    explicit ConnectionConfig(
        xjson::Parser parser
    ) : endpoint(parser.required<std::string>("endpoint")),
        username(parser.optional<std::string>("username", "")),
        password(parser.optional<std::string>("password", "")),
        security_mode(parser.optional<std::string>("security_mode", "None")),
        security_policy(parser.optional<std::string>("security_policy", "None")),
        client_cert(parser.optional<std::string>("client_certificate", "")),
        client_private_key(parser.optional<std::string>("client_private_key", "")),
        server_cert(parser.optional<std::string>("server_certificate", "")) {
    }

    json to_json() const {
        return {
            {"endpoint", endpoint},
            {"username", username},
            {"password", password},
            {"security_mode", security_mode},
            {"security_policy", security_policy},
            {"client_certificate", client_cert},
            {"client_private_key", client_private_key}
        };
    }
};

struct NodeProperties {
    telem::DataType data_type;
    std::string node_class;
    std::string name;
    std::string node_id;
    bool is_array;

    NodeProperties(
        const telem::DataType &data_type,
        const std::string &name,
        const std::string &node_id,
        const std::string &node_class,
        const bool is_array
    ) : data_type(data_type),
        node_class(node_class),
        name(name),
        node_id(node_id),
        is_array(is_array) {
    }

    explicit NodeProperties(
        xjson::Parser &p
    ) : data_type(telem::DataType(p.required<std::string>("data_type"))),
        name(p.required<std::string>("name")),
        node_id(p.required<std::string>("node_id")),
        is_array(p.optional<bool>("is_array", false)) {
    }

    json to_json() const {
        return {
            {"data_type", data_type.name()},
            {"name", name},
            {"node_id", node_id},
            {"node_class", node_class},
            {"is_array", is_array}
        };
    }
};

struct DeviceProperties {
    ConnectionConfig connection;
    std::vector<NodeProperties> channels;

    DeviceProperties(
        const ConnectionConfig &connection,
        const std::vector<NodeProperties> &channels
    ) : connection(connection), channels(channels) {
    }

    explicit DeviceProperties(
        const xjson::Parser &parser
    ) : connection(parser.child("connection")) {
        parser.iter("channels", [&](xjson::Parser &cb) {
            channels.emplace_back(cb);
        });
    }

    json to_json() const {
        json j;
        j["connection"] = connection.to_json();
        j["channels"] = json::array();
        for (const auto &ch: channels)
            j["channels"].push_back(ch.to_json());
        return j;
    }
};

std::pair<std::shared_ptr<UA_Client>, xerrors::Error> connect(
    const ConnectionConfig &cfg,
    std::string log_prefix
);

xerrors::Error reconnect(
    const std::shared_ptr<UA_Client> &client,
    const std::string &endpoint
);

const xerrors::Error CRITICAL_ERROR = driver::CRITICAL_HARDWARE_ERROR.sub("opc");
const xerrors::Error TEMPORARY_ERROR = driver::TEMPORARY_HARDWARE_ERROR.sub("opc");
const auto CONN_REJECTED = xerrors::Error(
    TEMPORARY_ERROR.sub("unreachable"),
    "unable to reach OPC UA server"
);


inline xerrors::Error parse_error(
    const UA_StatusCode &status
) {
    if (status == UA_STATUSCODE_GOOD) return xerrors::NIL;
    if (
        status == UA_STATUSCODE_BADCONNECTIONREJECTED ||
        status == UA_STATUSCODE_BADSECURECHANNELCLOSED
    )
        return CONN_REJECTED;
    const std::string status_name = UA_StatusCode_name(status);
    return {
        CRITICAL_ERROR,
        status_name
    };
};

telem::DataType ua_to_data_type(const UA_DataType *value);

UA_DataType *data_type_to_ua(const telem::DataType &data_type);

size_t write_to_series(telem::Series &s, const UA_Variant &v);

std::pair<UA_Variant, xerrors::Error> series_to_variant(const telem::Series &s);

std::pair<telem::Series, xerrors::Error> ua_array_to_series(
    const telem::DataType &target_type,
    const UA_Variant *val,
    size_t target_size,
    const std::string &name = ""
);

UA_NodeId parse_node_id(const std::string &path, xjson::Parser &parser);
std::pair<UA_NodeId, xerrors::Error> parse_node_id(const std::string &node_id_str);
std::string node_id_to_string(const UA_NodeId &node_id);
std::string node_class_to_string(const UA_NodeClass &node_class);

std::pair<telem::Series, xerrors::Error> simple_read(
    std::shared_ptr<UA_Client> client,
    const std::string &node_id
);
}
