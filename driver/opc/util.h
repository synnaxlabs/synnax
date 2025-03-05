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
#include <sstream>
#include <regex>

/// external
#include "open62541/types.h"
#include "open62541/nodeids.h"
#include "open62541/types.h"
#include "open62541/types_generated.h"
#include "open62541/client_config_default.h"
#include "open62541/client.h"
#include "open62541/statuscodes.h"

// module
#include "client/cpp/synnax.h"

/// internal
#include "driver/opc/opc.h"
#include "driver/errors/errors.h"

using json = nlohmann::json;

namespace opc {
/// @brief the configuration for an OPC UA connection.
struct ConnectionConfig {
    /// @brief the endpoint of the OPC UA server.
    std::string endpoint;
    /// @brief the username to use for authentication. Not required.
    std::string username;
    /// @brief the password to use for authentication. Not required.
    std::string password;
    /// @brief the security mode.
    std::string security_mode;
    /// @brief the security policy.
    std::string security_policy;
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

using ClientDeleter = void (*)(UA_Client *);
using VariantDeleter = void (*)(UA_Variant *);


std::pair<std::shared_ptr<UA_Client>, xerrors::Error> connect(
    opc::ConnectionConfig &cfg,
    std::string log_prefix
);

static xerrors::Error refresh_connection(
    const std::shared_ptr<UA_Client> &client,
    const std::string &endpoint
) {
    const UA_StatusCode status = UA_Client_connect(client.get(), endpoint.c_str());
    if (status != UA_STATUSCODE_GOOD) {
        const UA_StatusCode status_retry = UA_Client_connect(
            client.get(), endpoint.c_str());
        if (status_retry != UA_STATUSCODE_GOOD) {
            return xerrors::Error(
                "Failed to connect to OPC UA server: " +
                std::string(UA_StatusCode_name(status)));
        }
    }
    return xerrors::NIL;
}

///@brief Define constants for the conversion
static constexpr int64_t UNIX_EPOCH_START_1601 = 11644473600LL;
// Seconds from 1601 to 1970
static constexpr int64_t HUNDRED_NANOSECOND_INTERVALS_PER_SECOND = 10000000LL;
// 100-nanosecond intervals per second
constexpr int64_t UNIX_EPOCH_START_IN_100_NANO_INTERVALS =
        UNIX_EPOCH_START_1601 * HUNDRED_NANOSECOND_INTERVALS_PER_SECOND;

inline int64_t ua_datetime_to_unix_nano(const UA_DateTime dateTime) {
    return (dateTime - UNIX_EPOCH_START_IN_100_NANO_INTERVALS) * 100;
}

///@brief this function returns the appropriate synnax data type that corresponds to
/// the OPCUA data type
inline std::pair<telem::DataType, bool> variant_data_type(const UA_Variant &val) {
    if (!val.type) {
        LOG(ERROR) << "[opc.scanner] opc node type is null.";
        return {telem::UNKNOWN_T, false};
    }
    if (UA_Variant_hasArrayType(&val, &UA_TYPES[UA_TYPES_FLOAT]))
        return {telem::FLOAT32_T, true};
    if (UA_Variant_hasArrayType(&val, &UA_TYPES[UA_TYPES_DOUBLE]))
        return {telem::FLOAT64_T, true};
    if (UA_Variant_hasArrayType(&val, &UA_TYPES[UA_TYPES_INT16]))
        return {telem::INT16_T, true};
    if (UA_Variant_hasArrayType(&val, &UA_TYPES[UA_TYPES_INT32]))
        return {telem::INT32_T, true};
    if (UA_Variant_hasArrayType(&val, &UA_TYPES[UA_TYPES_INT64]))
        return {telem::INT64_T, true};
    if (UA_Variant_hasArrayType(&val, &UA_TYPES[UA_TYPES_UINT16]))
        return {telem::UINT16_T, true};
    if (UA_Variant_hasArrayType(&val, &UA_TYPES[UA_TYPES_UINT32]))
        return {telem::UINT32_T, true};
    if (UA_Variant_hasArrayType(&val, &UA_TYPES[UA_TYPES_UINT64]))
        return {telem::UINT64_T, true};
    if (UA_Variant_hasArrayType(&val, &UA_TYPES[UA_TYPES_STRING]))
        return {telem::STRING_T, true};
    if (UA_Variant_hasArrayType(&val, &UA_TYPES[UA_TYPES_DATETIME]))
        return {telem::TIMESTAMP_T, true};
    if (UA_Variant_hasArrayType(&val, &UA_TYPES[UA_TYPES_GUID]))
        return {telem::UINT128_T, true};
    if (UA_Variant_hasArrayType(&val, &UA_TYPES[UA_TYPES_BOOLEAN]))
        return {telem::UINT8_T, true};
    if (val.type == &UA_TYPES[UA_TYPES_FLOAT]) return {telem::FLOAT32_T, false};
    if (val.type == &UA_TYPES[UA_TYPES_DOUBLE]) return {telem::FLOAT64_T, false};
    if (val.type == &UA_TYPES[UA_TYPES_SBYTE]) return {telem::INT8_T, false};
    if (val.type == &UA_TYPES[UA_TYPES_INT16]) return {telem::INT16_T, false};
    if (val.type == &UA_TYPES[UA_TYPES_INT32]) return {telem::INT32_T, false};
    if (val.type == &UA_TYPES[UA_TYPES_INT64]) return {telem::INT64_T, false};
    if (val.type == &UA_TYPES[UA_TYPES_BYTE]) return {telem::UINT8_T, false};
    if (val.type == &UA_TYPES[UA_TYPES_UINT16]) return {telem::UINT16_T, false};
    if (val.type == &UA_TYPES[UA_TYPES_UINT32]) return {telem::UINT32_T, false};
    if (val.type == &UA_TYPES[UA_TYPES_UINT64]) return {telem::UINT64_T, false};
    if (val.type == &UA_TYPES[UA_TYPES_STRING]) return {telem::STRING_T, false};
    if (val.type == &UA_TYPES[UA_TYPES_DATETIME]) return {telem::TIMESTAMP_T, false};
    if (val.type == &UA_TYPES[UA_TYPES_GUID]) return {telem::UINT128_T, false};
    if (val.type == &UA_TYPES[UA_TYPES_BOOLEAN]) return {telem::UINT8_T, false};
    LOG(ERROR) << "[opc.scanner] Unknown data type: " << val.type->typeName;
    return {telem::UNKNOWN_T, false};
}

inline xerrors::Error
communicate_response_error(
    const UA_StatusCode &status,
    std::shared_ptr<task::Context> ctx,
    task::State &curr_state) {
    xerrors::Error err;
    if (
        status == UA_STATUSCODE_BADCONNECTIONREJECTED ||
        status == UA_STATUSCODE_BADSECURECHANNELCLOSED
    ) {
        err.type = driver::TEMPORARY_HARDWARE_ERROR.type;
        err.data = "connection rejected";
        curr_state.variant = "warning";
        curr_state.details = json{
            {
                "message",
                "Temporarily unable to reach OPC UA server. Will keep trying."
            },
            {"running", true}
        };
    } else {
        err.type = driver::CRITICAL_HARDWARE_ERROR.type;
        err.data = "failed to execute read: " + std::string(
                       UA_StatusCode_name(status));
        curr_state.variant = "error";
        curr_state.details = json{
            {
                "message", "Failed to read from OPC UA server: " + std::string(
                               UA_StatusCode_name(status))
            },
            {"running", false}
        };
    }
    ctx->set_state(curr_state);
    return err;
}
}

///@brief Helper function to convert string GUID to UA_Guid
inline UA_Guid string_to_guid(const std::string &guidStr) {
    UA_Guid guid;
    unsigned int data4[8];
    std::sscanf(guidStr.c_str(),
                "%8x-%4hx-%4hx-%2x%2x-%2x%2x%2x%2x%2x%2x",
                &guid.data1, &guid.data2, &guid.data3,
                &data4[0], &data4[1], &data4[2], &data4[3],
                &data4[4], &data4[5], &data4[6], &data4[7]);
    for (int i = 0; i < 8; ++i)
        guid.data4[i] = static_cast<UA_Byte>(data4[i]);
    return guid;
}


///@brief Helper function to convert a GUID to a string
inline std::string guid_to_string(const UA_Guid &guid) {
    std::ostringstream stream;
    stream << std::hex << std::setfill('0')
            << std::setw(8) << guid.data1 << "-"
            << std::setw(4) << guid.data2 << "-"
            << std::setw(4) << guid.data3 << "-"
            << std::setw(2) << (guid.data4[0] & 0xFF) << std::setw(2) << (
                guid.data4[1] & 0xFF) << "-"
            << std::setw(2) << (guid.data4[2] & 0xFF) << std::setw(2) << (
                guid.data4[3] & 0xFF)
            << std::setw(2) << (guid.data4[4] & 0xFF) << std::setw(2) << (
                guid.data4[5] & 0xFF)
            << std::setw(2) << (guid.data4[6] & 0xFF) << std::setw(2) << (
                guid.data4[7] & 0xFF);
    return stream.str();
}

///@brief Parses a string NodeId into a UA_NodeId object
inline UA_NodeId parse_node_id(const std::string &path, xjson::Parser &parser) {
    std::regex regex("NS=(\\d+);(I|S|G|B)=(.+)");
    std::smatch matches;
    const std::string nodeIdStr = parser.required<std::string>(path);
    if (!parser.ok()) return UA_NODEID_NULL;

    if (!std::regex_search(nodeIdStr, matches, regex)) {
        parser.field_err(path, "Invalid NodeId format");
        return UA_NODEID_NULL;
    }

    int nsIndex = std::stoi(matches[1].str());
    std::string type = matches[2].str();
    std::string identifier = matches[3].str();

    UA_NodeId nodeId = UA_NODEID_NULL;

    if (type == "I") {
        nodeId = UA_NODEID_NUMERIC(nsIndex, std::stoul(identifier));
    } else if (type == "S") {
        nodeId = UA_NODEID_STRING_ALLOC(nsIndex, identifier.c_str());
    } else if (type == "G") {
        UA_Guid guid = string_to_guid(identifier);
        nodeId = UA_NODEID_GUID(nsIndex, guid);
    } else if (type == "B") {
        size_t len = identifier.length() / 2;
        auto *data = static_cast<UA_Byte *>(UA_malloc(len));
        for (size_t i = 0; i < len; ++i) {
            sscanf(&identifier[2 * i], "%2hhx", &data[i]);
        }
        nodeId = UA_NODEID_BYTESTRING(nsIndex, reinterpret_cast<char *>(data));
        UA_free(data);
    }

    return nodeId;
}


inline std::string node_id_to_string(const UA_NodeId &nodeId) {
    std::ostringstream nodeIdStr;
    nodeIdStr << "NS=" << nodeId.namespaceIndex << ";";
    switch (nodeId.identifierType) {
        case UA_NODEIDTYPE_NUMERIC:
            nodeIdStr << "I=" << nodeId.identifier.numeric;
            break;
        case UA_NODEIDTYPE_STRING:
            nodeIdStr << "S=" << std::string(
                reinterpret_cast<char *>(nodeId.identifier.string.data),
                nodeId.identifier.string.length
            );
            break;
        case UA_NODEIDTYPE_GUID:
            nodeIdStr << "G=" << guid_to_string(nodeId.identifier.guid);
            break;
        case UA_NODEIDTYPE_BYTESTRING:
            // Convert ByteString to a base64 or similar readable format if needed
            nodeIdStr << "B=";
            for (std::size_t i = 0; i < nodeId.identifier.byteString.length; ++i) {
                nodeIdStr << std::setfill('0') << std::setw(2) << std::hex
                        << static_cast<int>(nodeId.identifier.byteString.data[i]);
            }
            break;
        default:
            nodeIdStr << "Unknown";
    }
    return nodeIdStr.str();
}

inline UA_ByteString string_to_ua_byte_string(const std::string &str) {
    const size_t len = str.length();
    const auto strData = reinterpret_cast<const UA_Byte *>(str.data());
    const auto data = static_cast<UA_Byte *>(malloc(len * sizeof(UA_Byte)));

    if (data == nullptr) {
        return UA_BYTESTRING_NULL;
    }

    memcpy(data, strData, len);

    UA_ByteString b = {
        .length = len,
        .data = data
    };

    return b;
}

inline xerrors::Error parse_error(
    const UA_StatusCode &status
) {
    const std::string status_name = UA_StatusCode_name(status);
    return {
        driver::CRITICAL_HARDWARE_ERROR.type,
        status_name
    };
}

inline telem::Series from_ua_array(
    const telem::DataType &data_type,
    const UA_Variant *val
) {
    if (UA_Variant_hasArrayType(val, &UA_TYPES[UA_TYPES_FLOAT])) {
        const auto *data = static_cast<UA_Float *>(val->data);
        return telem::Series::cast(data_type, data, val->arrayLength);
    }
    if (UA_Variant_hasArrayType(val, &UA_TYPES[UA_TYPES_DOUBLE])) {
        const UA_Double *data = static_cast<UA_Double *>(val->data);
        return telem::Series::cast(data_type, data, val->arrayLength);
    }
    if (UA_Variant_hasArrayType(val, &UA_TYPES[UA_TYPES_INT16])) {
        const UA_Int16 *data = static_cast<UA_Int16 *>(val->data);
        return telem::Series::cast(data_type, data, val->arrayLength);
    }
    if (UA_Variant_hasArrayType(val, &UA_TYPES[UA_TYPES_INT32])) {
        const UA_Int32 *data = static_cast<UA_Int32 *>(val->data);
        return telem::Series::cast(data_type, data, val->arrayLength);
    }
    if (UA_Variant_hasArrayType(val, &UA_TYPES[UA_TYPES_INT64])) {
        const UA_Int64 *data = static_cast<UA_Int64 *>(val->data);
        return telem::Series::cast(data_type, data, val->arrayLength);
    }
    if (UA_Variant_hasArrayType(val, &UA_TYPES[UA_TYPES_UINT32])) {
        const UA_UInt32 *data = static_cast<UA_UInt32 *>(val->data);
        return telem::Series::cast(data_type, data, val->arrayLength);
    }
    if (UA_Variant_hasArrayType(val, &UA_TYPES[UA_TYPES_UINT64])) {
        const UA_UInt64 *data = static_cast<UA_UInt64 *>(val->data);
        return telem::Series::cast(data_type, data, val->arrayLength);
    }
    if (UA_Variant_hasArrayType(val, &UA_TYPES[UA_TYPES_BYTE])) {
        const UA_Byte *data = static_cast<UA_Byte *>(val->data);
        return telem::Series::cast(data_type, data, val->arrayLength);
    }
    if (UA_Variant_hasArrayType(val, &UA_TYPES[UA_TYPES_SBYTE])) {
        const UA_SByte *data = static_cast<UA_SByte *>(val->data);
        return telem::Series::cast(data_type, data, val->arrayLength);
    }
    if (UA_Variant_hasArrayType(val, &UA_TYPES[UA_TYPES_BOOLEAN])) {
        const UA_Boolean *data = static_cast<UA_Boolean *>(val->data);
        return telem::Series::cast(data_type, data, val->arrayLength);
    }
    if (UA_Variant_hasArrayType(val, &UA_TYPES[UA_TYPES_DATETIME])) {
        const UA_DateTime *data = static_cast<UA_DateTime *>(val->data);
        auto s = telem::Series(data_type, val->arrayLength);
        size_t acc = 0;
        for (size_t j = 0; j < val->arrayLength; ++j)
            acc += s.write(opc::ua_datetime_to_unix_nano(data[j]));
        return s;
    }
}

inline size_t write_to_series(
    const UA_Variant &val,
    telem::Series &s
) {
    if (val.type == &UA_TYPES[UA_TYPES_FLOAT]) {
        const auto value = *static_cast<UA_Float *>(val.data);
        return s.write(s.data_type().cast(value));
    }
    if (val.type == &UA_TYPES[UA_TYPES_DOUBLE]) {
        const auto value = *static_cast<UA_Double *>(val.data);
        return s.write(s.data_type().cast(value));
    }
    if (val.type == &UA_TYPES[UA_TYPES_INT16]) {
        const auto value = *static_cast<UA_Int16 *>(val.data);
        return s.write(s.data_type().cast(value));
    }
    if (val.type == &UA_TYPES[UA_TYPES_INT32]) {
        const auto value = *static_cast<UA_Int32 *>(val.data);
        return s.write(s.data_type().cast(value));
    }
    if (val.type == &UA_TYPES[UA_TYPES_INT64]) {
        const auto value = *static_cast<UA_Int64 *>(val.data);
        return s.write(s.data_type().cast(value));
    }
    if (val.type == &UA_TYPES[UA_TYPES_UINT32]) {
        const auto value = *static_cast<UA_UInt32 *>(val.data);
        return s.write(s.data_type().cast(value));
    }
    if (val.type == &UA_TYPES[UA_TYPES_UINT64]) {
        const auto value = *static_cast<UA_UInt64 *>(val.data);
        return s.write(s.data_type().cast(value));
    }
    if (val.type == &UA_TYPES[UA_TYPES_BYTE]) {
        const auto value = *static_cast<UA_Byte *>(val.data);
        return s.write(s.data_type().cast(value));
    }
    if (val.type == &UA_TYPES[UA_TYPES_SBYTE]) {
        const auto value = *static_cast<UA_SByte *>(val.data);
        return s.write(s.data_type().cast(value));
    }
    if (val.type == &UA_TYPES[UA_TYPES_BOOLEAN]) {
        const auto value = *static_cast<UA_Boolean *>(val.data);
        return s.write(s.data_type().cast(value));
    }
    if (val.type == &UA_TYPES[UA_TYPES_DATETIME]) {
        const auto value = *static_cast<UA_DateTime *>(val.data);
        return s.write(opc::ua_datetime_to_unix_nano(value));
    }
    return 0;
}


std::pair<std::unique_ptr<UA_Variant>, xerrors::Error> series_to_variant(
    const telem::Series &s) {
    auto variant = std::unique_ptr<UA_Variant, opc::VariantDeleter>(
        UA_Variant_new(), UA_Variant_delete
    );
    auto status = UA_Variant_setScalarCopy(
        variant.get(),
        s.at(-1),
        &UA_TYPES[s.data_type()]
    );
    return {parse_error(status), std::move(variant)};
}
