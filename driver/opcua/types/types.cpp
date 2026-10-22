// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cctype>
#include <charconv>
#include <iomanip>
#include <map>
#include <new>
#include <sstream>
#include <string_view>

#include "open62541/types.h"

#include "driver/opcua/telem/telem.h"
#include "driver/opcua/types/types.h"

namespace driver::opcua::types {
namespace {
/// @brief parses an unsigned decimal number that spans all of str.
/// @param str the digits to parse.
/// @param value set to the parsed number on success.
/// @returns false if str is empty, has a non-digit, or overflows T.
template<typename T>
bool parse_number(const std::string_view str, T &value) {
    const auto end = str.data() + str.size();
    const auto [ptr, ec] = std::from_chars(str.data(), end, value);
    return ec == std::errc() && ptr == end;
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
    const auto invalid = [] {
        return std::pair{
            NodeId(),
            x::errors::Error(x::errors::VALIDATION, "Invalid NodeId format")
        };
    };
    const auto upper = [](const char c) {
        return static_cast<char>(std::toupper(static_cast<unsigned char>(c)));
    };
    const std::string_view str = node_id_str;
    const auto sep = str.find(';');
    if (sep == std::string_view::npos || sep < 3 || upper(str[0]) != 'N' ||
        upper(str[1]) != 'S' || str[2] != '=')
        return invalid();
    UA_UInt16 ns = 0;
    if (!parse_number(str.substr(3, sep - 3), ns)) return invalid();
    const auto body = str.substr(sep + 1);
    if (body.size() < 3 || body[1] != '=') return invalid();
    std::string identifier(body.substr(2));
    const UA_String ua_identifier{
        identifier.size(),
        reinterpret_cast<UA_Byte *>(identifier.data())
    };

    NodeId result;
    UA_NodeId &id = result.id_;
    switch (upper(body[0])) {
        case 'I': {
            UA_UInt32 numeric = 0;
            if (!parse_number(std::string_view(identifier), numeric)) return invalid();
            id = UA_NODEID_NUMERIC(ns, numeric);
            break;
        }
        case 'S':
            id = UA_NODEID_STRING_ALLOC(ns, identifier.c_str());
            break;
        case 'G': {
            UA_Guid guid;
            if (UA_Guid_parse(&guid, ua_identifier) != UA_STATUSCODE_GOOD)
                return invalid();
            id = UA_NODEID_GUID(ns, guid);
            break;
        }
        case 'B': {
            id.namespaceIndex = ns;
            id.identifierType = UA_NODEIDTYPE_BYTESTRING;
            // The decoder is lenient, so only input that re-encodes to itself is valid.
            String canonical;
            if (UA_ByteString_fromBase64(&id.identifier.byteString, &ua_identifier) ==
                    UA_STATUSCODE_GOOD &&
                UA_ByteString_toBase64(&id.identifier.byteString, canonical.ptr()) !=
                    UA_STATUSCODE_GOOD)
                throw std::bad_alloc();
            if (!UA_String_equal(&ua_identifier, &canonical.get()))
                return {
                    NodeId(),
                    x::errors::Error(
                        x::errors::VALIDATION,
                        "Invalid base64 in ByteString identifier: " + identifier
                    )
                };
            break;
        }
        default:
            return invalid();
    }
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
        case UA_NODEIDTYPE_BYTESTRING: {
            String encoded;
            if (UA_ByteString_toBase64(&node_id.identifier.byteString, encoded.ptr()) !=
                UA_STATUSCODE_GOOD)
                throw std::bad_alloc();
            node_id_str << "B="
                        << std::string(
                               reinterpret_cast<char *>(encoded.get().data),
                               encoded.get().length
                           );
            break;
        }
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
    auto [variant, err] = telem::series_to_variant(series);
    if (err) return err;
    add_value(node_id, variant);
    return x::errors::NIL;
}
}
