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
#include <string>
#include <string_view>
#include <vector>

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
    const auto *const end = str.data() + str.size();
    const auto [ptr, ec] = std::from_chars(str.data(), end, value);
    return ec == std::errc() && ptr == end;
}

/// @returns true if str starts with prefix, ignoring the case of letters.
bool starts_with(const std::string_view str, const std::string_view prefix) {
    if (str.size() < prefix.size()) return false;
    for (std::size_t i = 0; i < prefix.size(); ++i)
        if (std::tolower(static_cast<unsigned char>(str[i])) != prefix[i]) return false;
    return true;
}

/// @brief decodes the %XX escapes in a namespace URI.
/// @returns nothing if an escape is not two hex digits.
std::optional<std::string> percent_decode(const std::string_view str) {
    std::string out;
    out.reserve(str.size());
    for (std::size_t i = 0; i < str.size(); ++i) {
        if (str[i] != '%') {
            out.push_back(str[i]);
            continue;
        }
        unsigned char c = 0;
        const auto *const begin = str.data() + i + 1;
        if (i + 2 >= str.size() ||
            std::from_chars(begin, begin + 2, c, 16).ptr != begin + 2)
            return std::nullopt;
        out.push_back(static_cast<char>(c));
        i += 2;
    }
    return out;
}

/// @brief escapes the characters that end or escape a namespace URI.
std::string percent_encode(const std::string &str) {
    std::string out;
    out.reserve(str.size());
    for (const char c: str)
        if (c == ';')
            out += "%3B";
        else if (c == '%')
            out += "%25";
        else
            out.push_back(c);
    return out;
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

/// @returns the identifier of the node ID as `<type>=<identifier>`, with the type in
/// the given case.
std::string body(const UA_NodeId &node_id, const bool upper) {
    const auto type = [upper](const char c) {
        return std::string(1, upper ? c : static_cast<char>(std::tolower(c))) + "=";
    };
    switch (node_id.identifierType) {
        case UA_NODEIDTYPE_NUMERIC:
            return type('I') + std::to_string(node_id.identifier.numeric);
        case UA_NODEIDTYPE_STRING:
            return type('S') +
                   std::string(
                       reinterpret_cast<char *>(node_id.identifier.string.data),
                       node_id.identifier.string.length
                   );
        case UA_NODEIDTYPE_GUID:
            return type('G') + guid_to_string(node_id.identifier.guid);
        case UA_NODEIDTYPE_BYTESTRING: {
            String encoded;
            if (UA_ByteString_toBase64(&node_id.identifier.byteString, encoded.ptr()) !=
                UA_STATUSCODE_GOOD)
                throw std::bad_alloc();
            return type('B') + std::string(
                                   reinterpret_cast<char *>(encoded.get().data),
                                   encoded.get().length
                               );
        }
        default:
            return "Unknown";
    }
}
}

NodeId NodeId::parse(const std::string &field_name, x::json::Parser &parser) {
    const auto nodeIdStr = parser.field<std::string>(field_name);
    if (!parser.ok()) return NodeId();
    auto [node_id, err] = parse(nodeIdStr);
    if (err) {
        parser.field_err(field_name, err.data);
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
    std::string_view body = node_id_str;
    UA_UInt16 ns = 0;
    std::string uri;
    if (starts_with(body, "nsu=")) {
        const auto sep = body.find(';');
        if (sep == std::string_view::npos) return invalid();
        auto decoded = percent_decode(body.substr(4, sep - 4));
        if (!decoded.has_value() || decoded->empty()) return invalid();
        uri = std::move(*decoded);
        body = body.substr(sep + 1);
    } else if (starts_with(body, "ns=")) {
        const auto sep = body.find(';');
        if (sep == std::string_view::npos || !parse_number(body.substr(3, sep - 3), ns))
            return invalid();
        body = body.substr(sep + 1);
    }
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
    result.namespace_uri_ = std::move(uri);
    return {std::move(result), x::errors::NIL};
}

std::optional<UA_NodeId>
NodeId::resolve(const std::vector<std::string> &namespaces) const {
    if (this->namespace_uri_.empty()) return this->id_;
    for (std::size_t i = 0; i < namespaces.size(); ++i)
        if (namespaces[i] == this->namespace_uri_) {
            UA_NodeId resolved = this->id_;
            resolved.namespaceIndex = static_cast<UA_UInt16>(i);
            return resolved;
        }
    return std::nullopt;
}

std::string NodeId::to_string(
    const UA_NodeId &node_id,
    const std::vector<std::string> &namespaces
) {
    const auto ns = node_id.namespaceIndex;
    if (ns == 0) return body(node_id, false);
    if (ns < namespaces.size() && !namespaces[ns].empty())
        return "nsu=" + percent_encode(namespaces[ns]) + ";" + body(node_id, false);
    return "ns=" + std::to_string(ns) + ";" + body(node_id, false);
}

std::string NodeId::to_string() const {
    if (this->namespace_uri_.empty()) return to_string(this->id_);
    return "nsu=" + percent_encode(this->namespace_uri_) + ";" + body(this->id_, false);
}

std::string NodeId::to_string(const UA_NodeId &node_id) {
    return "NS=" + std::to_string(node_id.namespaceIndex) + ";" + body(node_id, true);
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
