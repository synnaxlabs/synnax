// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// std
#include <iomanip>
#include <regex>
#include <sstream>

/// module
#include "x/cpp/xjson/xjson.h"

/// external
#include "open62541/types.h"

/// internal
#include "driver/opc/util/util.h"

namespace util {
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
UA_NodeId parse_node_id(const std::string &path, xjson::Parser &parser) {
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
}