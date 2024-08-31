// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "opc.h"
#include "client/cpp/synnax.h"
#include "include/open62541/types.h"
#include "include/open62541/nodeids.h"
#include "include/open62541/types.h"
#include "include/open62541/types_generated.h"
#include "include/open62541/client_config_default.h"
#include "include/open62541/client.h"
#include "include/open62541/statuscodes.h"
#include <string>
#include <sstream>
#include <regex>
#include <iostream>

namespace opc {
using ClientDeleter = void (*)(UA_Client *);


std::pair<std::shared_ptr<UA_Client>, freighter::Error> connect(
    opc::ConnectionConfig &cfg,
    std::string log_prefix
);

// Define constants for the conversion
static const int64_t UNIX_EPOCH_START_1601 = 11644473600LL; // Seconds from 1601 to 1970
static const int64_t HUNDRED_NANOSECOND_INTERVALS_PER_SECOND = 10000000LL;
// 100-nanosecond intervals per second

// Function to convert UA_DateTime to Unix timestamp in nanoseconds
inline int64_t ua_datetime_to_unix_nano(UA_DateTime dateTime) {
    int64_t unixEpochStartIn100NanoIntervals =
            UNIX_EPOCH_START_1601 * HUNDRED_NANOSECOND_INTERVALS_PER_SECOND;
    return (dateTime - unixEpochStartIn100NanoIntervals) * 100;
}

inline synnax::Series val_to_series(UA_Variant *val, synnax::DataType dt) {
    if (val->type == &UA_TYPES[UA_TYPES_FLOAT]) {
        const auto value = *static_cast<UA_Float *>(val->data);
        if (dt == synnax::FLOAT32) return Series(value);
        if (dt == synnax::FLOAT64) return Series(static_cast<double>(value));
    }
    if (val->type == &UA_TYPES[UA_TYPES_DOUBLE]) {
        const auto value = *static_cast<UA_Double *>(val->data);
        // TODO - warn on potential precision drop here
        if (dt == synnax::FLOAT32) return Series(static_cast<float>(value));
        if (dt == synnax::FLOAT64) return Series(value);
    }
    if (val->type == &UA_TYPES[UA_TYPES_INT32]) {
        const auto value = *static_cast<UA_Int32 *>(val->data);
        if (dt == synnax::INT32) return Series(value);
        if (dt == synnax::INT64) return Series(static_cast<int64_t>(value));
        if (dt == synnax::UINT32) return Series(static_cast<uint32_t>(value));
        if (dt == synnax::UINT64) return Series(static_cast<uint64_t>(value));
    }
    if (val->type == &UA_TYPES[UA_TYPES_INT64]) {
        const auto value = *static_cast<UA_Int64 *>(val->data);
        if (dt == synnax::INT32) return Series(static_cast<int32_t>(value));
        if (dt == synnax::INT64) return Series(value);
        if (dt == synnax::UINT32) return Series(static_cast<uint32_t>(value));
        if (dt == synnax::UINT64) return Series(static_cast<uint64_t>(value));
        if (dt == synnax::TIMESTAMP) return Series(static_cast<uint64_t>(value));
    }
    if (val->type == &UA_TYPES[UA_TYPES_UINT32]) {
        const auto value = *static_cast<UA_UInt32 *>(val->data);
        if (dt == synnax::INT32) return synnax::Series(static_cast<int32_t>(value));
        // Potential data loss
        if (dt == synnax::INT64) return synnax::Series(static_cast<int64_t>(value));
        if (dt == synnax::UINT32) return synnax::Series(value);
        if (dt == synnax::UINT64)
            return synnax::Series(static_cast<uint64_t>(value));
    }
    if (val->type == &UA_TYPES[UA_TYPES_UINT64]) {
        const auto value = *static_cast<UA_UInt64 *>(val->data);
        if (dt == synnax::UINT64) return synnax::Series(value);
        if (dt == synnax::INT32) return synnax::Series(static_cast<int32_t>(value));
        // Potential data loss
        if (dt == synnax::INT64) return synnax::Series(static_cast<int64_t>(value));
        if (dt == synnax::UINT32)
            return synnax::Series(static_cast<uint32_t>(value));
        // Potential data loss
        if (dt == synnax::TIMESTAMP)
            return
                    synnax::Series(static_cast<uint64_t>(value));
    }
    if (val->type == &UA_TYPES[UA_TYPES_BYTE]) {
        const auto value = *static_cast<UA_Byte *>(val->data);
        if (dt == synnax::UINT8) return synnax::Series(value);
        if (dt == synnax::UINT16)
            return synnax::Series(static_cast<uint16_t>(value));
        if (dt == synnax::UINT32)
            return synnax::Series(static_cast<uint32_t>(value));
        if (dt == synnax::UINT64)
            return synnax::Series(static_cast<uint64_t>(value));
        if (dt == synnax::INT8) return synnax::Series(static_cast<int8_t>(value));
        if (dt == synnax::INT16) return synnax::Series(static_cast<int16_t>(value));
        if (dt == synnax::INT32) return synnax::Series(static_cast<int32_t>(value));
        if (dt == synnax::INT64) return synnax::Series(static_cast<int64_t>(value));
        if (dt == synnax::FLOAT32) return synnax::Series(static_cast<float>(value));
        if (dt == synnax::FLOAT64)
            return synnax::Series(static_cast<double>(value));
    }
    if (val->type == &UA_TYPES[UA_TYPES_SBYTE]) {
        const auto value = *static_cast<UA_SByte *>(val->data);
        if (dt == synnax::INT8) return synnax::Series(value);
        if (dt == synnax::INT16) return synnax::Series(static_cast<int16_t>(value));
        if (dt == synnax::INT32) return synnax::Series(static_cast<int32_t>(value));
        if (dt == synnax::INT64) return synnax::Series(static_cast<int64_t>(value));
        if (dt == synnax::FLOAT32) return synnax::Series(static_cast<float>(value));
        if (dt == synnax::FLOAT64)
            return synnax::Series(static_cast<double>(value));
    }
    if (val->type == &UA_TYPES[UA_TYPES_BOOLEAN]) {
        const auto value = *static_cast<UA_Boolean *>(val->data);
        if (dt == synnax::UINT8) return synnax::Series(static_cast<uint8_t>(value));
        if (dt == synnax::UINT16)
            return synnax::Series(static_cast<uint16_t>(value));
        if (dt == synnax::UINT32)
            return synnax::Series(static_cast<uint32_t>(value));
        if (dt == synnax::UINT64)
            return synnax::Series(static_cast<uint64_t>(value));
        if (dt == synnax::INT8) return synnax::Series(static_cast<int8_t>(value));
        if (dt == synnax::INT16) return synnax::Series(static_cast<int16_t>(value));
        if (dt == synnax::INT32) return synnax::Series(static_cast<int32_t>(value));
        if (dt == synnax::INT64) return synnax::Series(static_cast<int64_t>(value));
        if (dt == synnax::FLOAT32) return synnax::Series(static_cast<float>(value));
        if (dt == synnax::FLOAT64)
            return synnax::Series(static_cast<double>(value));
    }
    if (val->type == &UA_TYPES[UA_TYPES_DATETIME]) {
        const auto value = *static_cast<UA_DateTime *>(val->data);
        if (dt == synnax::INT64)
            return synnax::Series(ua_datetime_to_unix_nano(value));
        if (dt == synnax::TIMESTAMP)
            return synnax::Series(
                ua_datetime_to_unix_nano(value));
        if (dt == synnax::UINT64)
            return synnax::Series(
                static_cast<uint64_t>(ua_datetime_to_unix_nano(value)));
        if (dt == synnax::FLOAT32) return synnax::Series(static_cast<float>(value));
        if (dt == synnax::FLOAT64)
            return synnax::Series(static_cast<double>(value));
    }
    return Series(1);
}

inline std::pair<synnax::DataType, bool> variant_data_type(const UA_Variant &val) {
    if (UA_Variant_hasArrayType(&val, &UA_TYPES[UA_TYPES_FLOAT]))
        return {
            synnax::FLOAT32, true
        };
    if (UA_Variant_hasArrayType(&val, &UA_TYPES[UA_TYPES_DOUBLE]))
        return {
            synnax::FLOAT64, true
        };
    if (UA_Variant_hasArrayType(&val, &UA_TYPES[UA_TYPES_INT16]))
        return {
            synnax::INT16, true
        };
    if (UA_Variant_hasArrayType(&val, &UA_TYPES[UA_TYPES_INT32]))
        return {
            synnax::INT32, true
        };
    if (UA_Variant_hasArrayType(&val, &UA_TYPES[UA_TYPES_INT64]))
        return {
            synnax::INT64, true
        };
    if (UA_Variant_hasArrayType(&val, &UA_TYPES[UA_TYPES_UINT16]))
        return {
            synnax::UINT16, true
        };
    if (UA_Variant_hasArrayType(&val, &UA_TYPES[UA_TYPES_UINT32]))
        return {
            synnax::UINT32, true
        };
    if (UA_Variant_hasArrayType(&val, &UA_TYPES[UA_TYPES_UINT64]))
        return {
            synnax::UINT64, true
        };
    if (UA_Variant_hasArrayType(&val, &UA_TYPES[UA_TYPES_STRING]))
        return {
            synnax::STRING, true
        };
    if (UA_Variant_hasArrayType(&val, &UA_TYPES[UA_TYPES_DATETIME]))
        return {
            synnax::TIMESTAMP, true
        };
    if (UA_Variant_hasArrayType(&val, &UA_TYPES[UA_TYPES_GUID]))
        return {
            synnax::UINT128, true
        };
    if (val.type == &UA_TYPES[UA_TYPES_FLOAT]) return {synnax::FLOAT32, false};
    if (val.type == &UA_TYPES[UA_TYPES_DOUBLE]) return {synnax::FLOAT64, false};
    if (val.type == &UA_TYPES[UA_TYPES_INT16]) return {synnax::INT16, false};
    if (val.type == &UA_TYPES[UA_TYPES_INT32]) return {synnax::INT32, false};
    if (val.type == &UA_TYPES[UA_TYPES_INT64]) return {synnax::INT64, false};
    if (val.type == &UA_TYPES[UA_TYPES_UINT16]) return {synnax::UINT16, false};
    if (val.type == &UA_TYPES[UA_TYPES_UINT32]) return {synnax::UINT32, false};
    if (val.type == &UA_TYPES[UA_TYPES_UINT64]) return {synnax::UINT64, false};
    if (val.type == &UA_TYPES[UA_TYPES_STRING]) return {synnax::STRING, false};
    if (val.type == &UA_TYPES[UA_TYPES_DATETIME]) return {synnax::TIMESTAMP, false};
    if (val.type == &UA_TYPES[UA_TYPES_GUID]) return {synnax::UINT128, false};
    LOG(ERROR) << "Unknown data type: " << val.type->typeName;
    return {synnax::DATA_TYPE_UNKNOWN, false};
}
}

// Helper function to convert string GUID to UA_Guid
inline UA_Guid stringToGuid(const std::string &guidStr) {
    UA_Guid guid;
    unsigned int data4[8];
    std::sscanf(guidStr.c_str(),
                "%8x-%4hx-%4hx-%2x%2x-%2x%2x%2x%2x%2x%2x",
                &guid.data1, &guid.data2, &guid.data3,
                &data4[0], &data4[1], &data4[2], &data4[3],
                &data4[4], &data4[5], &data4[6], &data4[7]);
    for (int i = 0; i < 8; ++i)
        guid.data4[i] = (UA_Byte) data4[i];
    return guid;
}


// Helper function to convert a GUID to a string
inline std::string guidToString(const UA_Guid &guid) {
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

// Parses a string NodeId into a UA_NodeId object
inline UA_NodeId parseNodeId(const std::string &path, config::Parser &parser) {
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
        UA_Guid guid = stringToGuid(identifier);
        nodeId = UA_NODEID_GUID(nsIndex, guid);
    } else if (type == "B") {
        // Allocate memory for ByteString
        size_t len = identifier.length() / 2;
        auto *data = (UA_Byte *) UA_malloc(len);
        for (size_t i = 0; i < len; ++i) {
            sscanf(&identifier[2 * i], "%2hhx", &data[i]);
        }
        nodeId = UA_NODEID_BYTESTRING(nsIndex, (char *) data);
        UA_free(data); // Free the temporary buffer
    }

    return nodeId;
}


// Function to build a string identifier from a UA_NodeId
inline std::string nodeIdToString(const UA_NodeId &nodeId) {
    std::ostringstream nodeIdStr;
    nodeIdStr << "NS=" << nodeId.namespaceIndex << ";";

    switch (nodeId.identifierType) {
        case UA_NODEIDTYPE_NUMERIC:
            nodeIdStr << "I=" << nodeId.identifier.numeric;
            break;
        case UA_NODEIDTYPE_STRING:
            nodeIdStr << "S=" << std::string((char *) nodeId.identifier.string.data,
                                             nodeId.identifier.string.length);
            break;
        case UA_NODEIDTYPE_GUID:
            nodeIdStr << "G=" << guidToString(nodeId.identifier.guid);
            break;
        case UA_NODEIDTYPE_BYTESTRING:
            // Convert ByteString to a base64 or similar readable format if needed
            nodeIdStr << "B=";
            for (std::size_t i = 0; i < nodeId.identifier.byteString.length; ++i) {
                nodeIdStr << std::setfill('0') << std::setw(2) << std::hex
                        << (int) nodeId.identifier.byteString.data[i];
            }
            break;
        default:
            nodeIdStr << "Unknown";
    }

    return nodeIdStr.str();
}

inline UA_ByteString stringToUAByteString(const std::string &str) {
    size_t len = str.length();
    const UA_Byte *strData = reinterpret_cast<const UA_Byte *>(str.data());

    UA_Byte *data = static_cast<UA_Byte *>(malloc(len * sizeof(UA_Byte)));

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
