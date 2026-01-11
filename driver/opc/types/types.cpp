// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// std
#include <iomanip>
#include <map>
// Disable GCC 13 false positive warning in <regex> header
#if defined(__GNUC__) && !defined(__clang__)
#pragma GCC diagnostic push
#pragma GCC diagnostic ignored "-Wmaybe-uninitialized"
#endif
#include <regex>
#if defined(__GNUC__) && !defined(__clang__)
#pragma GCC diagnostic pop
#endif
#include <sstream>

/// external
#include "open62541/types.h"

/// internal
#include "driver/opc/telem/telem.h"
#include "driver/opc/types/types.h"

namespace driver::opc {
namespace {
/// @brief Helper function to convert string GUID to UA_Guid
UA_Guid string_to_guid(const std::string &guidStr) {
    UA_Guid guid;
    unsigned int data4[8];
    std::sscanf(
        guidStr.c_str(),
        "%8x-%4hx-%4hx-%2x%2x-%2x%2x%2x%2x%2x%2x",
        &guid.data1,
        &guid.data2,
        &guid.data3,
        &data4[0],
        &data4[1],
        &data4[2],
        &data4[3],
        &data4[4],
        &data4[5],
        &data4[6],
        &data4[7]
    );
    for (int i = 0; i < 8; ++i)
        guid.data4[i] = static_cast<UA_Byte>(data4[i]);
    return guid;
}

/// @brief Helper function to convert a GUID to a string
std::string guid_to_string(const UA_Guid &guid) {
    std::ostringstream stream;
    stream << std::hex << std::setfill('0') << std::setw(8) << guid.data1 << "-"
           << std::setw(4) << guid.data2 << "-" << std::setw(4) << guid.data3 << "-"
           << std::setw(2) << (guid.data4[0] & 0xFF) << std::setw(2)
           << (guid.data4[1] & 0xFF) << "-" << std::setw(2) << (guid.data4[2] & 0xFF)
           << std::setw(2) << (guid.data4[3] & 0xFF) << std::setw(2)
           << (guid.data4[4] & 0xFF) << std::setw(2) << (guid.data4[5] & 0xFF)
           << std::setw(2) << (guid.data4[6] & 0xFF) << std::setw(2)
           << (guid.data4[7] & 0xFF);
    return stream.str();
}
}

NodeId NodeId::parse(const std::string &field_name, x::json::Parser &parser) {
    const std::string nodeIdStr = parser.field<std::string>(field_name);
    if (!parser.ok()) return NodeId();
    auto [node_id, err] = parse(nodeIdStr);
    if (err) {
        parser.field_err(field_name, err.message());
        return NodeId();
    }
    return std::move(node_id);
}

std::pair<NodeId, x::errors::Error> NodeId::parse(const std::string &node_id_str) {
    std::regex regex("NS=(\\d+);(I|S|G|B)=(.+)");
    std::smatch matches;
    if (!std::regex_search(node_id_str, matches, regex))
        return {NodeId(), x::errors::Error(x::errors::VALIDATION, "Invalid NodeId format")};

    int nsIndex = std::stoi(matches[1].str());
    std::string type = matches[2].str();
    std::string identifier = matches[3].str();

    UA_NodeId raw_id = UA_NODEID_NULL;
    if (type == "I")
        raw_id = UA_NODEID_NUMERIC(
            static_cast<UA_UInt16>(nsIndex),
            std::stoul(identifier)
        );
    else if (type == "S")
        raw_id = UA_NODEID_STRING_ALLOC(
            static_cast<UA_UInt16>(nsIndex),
            identifier.c_str()
        );
    else if (type == "G")
        raw_id = UA_NODEID_GUID(
            static_cast<UA_UInt16>(nsIndex),
            string_to_guid(identifier)
        );
    else if (type == "B") {
        size_t len = identifier.length() / 2;
        auto *data = static_cast<UA_Byte *>(UA_malloc(len));
        for (size_t i = 0; i < len; ++i)
            sscanf(&identifier[2 * i], "%2hhx", &data[i]);
        raw_id = UA_NODEID_BYTESTRING(
            static_cast<UA_UInt16>(nsIndex),
            reinterpret_cast<char *>(data)
        );
        UA_free(data);
    }

    // Wrap in RAII type - NodeId constructor will take ownership
    NodeId result(raw_id);
    // Clear the raw_id to prevent double-free (NodeId now owns it)
    UA_NodeId_clear(&raw_id);
    return {std::move(result), x::errors::NIL};
}

std::string NodeId::to_string(const UA_NodeId &node_id) {
    std::ostringstream node_id_str;
    node_id_str << "NS=" << node_id.namespaceIndex << ";";
    switch (node_id.identifierType) {
        case UA_NODEIDTYPE_NUMERIC:
            node_id_str << "I=" << node_id.identifier.numeric;
            break;
        case UA_NODEIDTYPE_STRING:
            node_id_str << "S="
                        << std::string(
                               reinterpret_cast<char *>(node_id.identifier.string.data),
                               node_id.identifier.string.length
                           );
            break;
        case UA_NODEIDTYPE_GUID:
            node_id_str << "G=" << guid_to_string(node_id.identifier.guid);
            break;
        case UA_NODEIDTYPE_BYTESTRING:
            node_id_str << "B=";
            for (std::size_t i = 0; i < node_id.identifier.byteString.length; ++i) {
                node_id_str << std::setfill('0') << std::setw(2) << std::hex
                            << static_cast<int>(node_id.identifier.byteString.data[i]);
            }
            break;
        default:
            node_id_str << "Unknown";
    }
    return node_id_str.str();
}

static const std::map<UA_NodeClass, std::string> NODE_CLASS_MAP = {
    {UA_NODECLASS_OBJECT, "Object"},
    {UA_NODECLASS_VARIABLE, "Variable"},
    {UA_NODECLASS_METHOD, "Method"},
    {UA_NODECLASS_OBJECTTYPE, "ObjectType"},
    {UA_NODECLASS_VARIABLETYPE, "VariableType"},
    {UA_NODECLASS_DATATYPE, "DataType"},
    {UA_NODECLASS_REFERENCETYPE, "ReferenceType"},
    {UA_NODECLASS_VIEW, "View"}
};

std::string node_class_to_string(const UA_NodeClass &node_class) {
    return NODE_CLASS_MAP.at(node_class);
}

x::errors::Error WriteRequestBuilder::add_value(
    const UA_NodeId &node_id,
    const ::x::telem::Series &series
) {
    auto [variant, err] = driver::opc::telem::series_to_variant(series);
    if (err) return err;
    add_value(node_id, variant);
    return x::errors::NIL;
}
}
