// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <optional>
#include <string>
#include <utility>
#include <vector>

#include "open62541/types.h"

#include "x/cpp/errors/errors.h"
#include "x/cpp/json/json.h"
#include "x/cpp/telem/series.h"

namespace driver::opcua::types {
// NOLINTBEGIN(misc-explicit-constructor): each wrapper converts to the open62541 type
// it holds.
struct Node {
    x::telem::DataType data_type;
    std::string node_class;
    std::string name;
    std::string node_id;
    bool is_array;
    /// @brief the path to a member of the node's structure value, or empty for the node
    /// itself.
    std::string field;

    Node(
        x::telem::DataType data_type,
        std::string name,
        std::string node_id,
        std::string node_class,
        const bool is_array,
        std::string field = ""
    ):
        data_type(std::move(data_type)),
        node_class(std::move(node_class)),
        name(std::move(name)),
        node_id(std::move(node_id)),
        is_array(is_array),
        field(std::move(field)) {}

    explicit Node(x::json::Parser &p):
        data_type(x::telem::DataType(p.field<std::string>("data_type"))),
        name(p.field<std::string>("name")),
        node_id(p.field<std::string>("node_id")),
        is_array(p.field<bool>("is_array", false)),
        field(p.field<std::string>("field", "")) {}

    [[nodiscard]] x::json::json to_json() const {
        return {
            {"data_type", data_type.name()},
            {"name", name},
            {"node_id", node_id},
            {"node_class", node_class},
            {"is_array", is_array},
            {"field", field}
        };
    }
};

class NodeId {
    UA_NodeId id{};
    /// @brief the namespace of the node when the node ID names it by URI. The namespace
    /// index of id is then not used.
    std::string uri;

public:
    NodeId() { UA_NodeId_init(&this->id); }

    explicit NodeId(const UA_NodeId &id) {
        UA_NodeId_init(&this->id);
        UA_NodeId_copy(&id, &this->id);
    }

    ~NodeId() { UA_NodeId_clear(&this->id); }

    NodeId(const NodeId &) = delete;
    NodeId &operator=(const NodeId &) = delete;

    NodeId(NodeId &&other) noexcept: id(other.id), uri(std::move(other.uri)) {
        UA_NodeId_init(&other.id);
    }

    NodeId &operator=(NodeId &&other) noexcept {
        if (this != &other) {
            UA_NodeId_clear(&this->id);
            this->id = other.id;
            this->uri = std::move(other.uri);
            UA_NodeId_init(&other.id);
        }
        return *this;
    }

    [[nodiscard]] bool is_null() const { return UA_NodeId_isNull(&this->id); }
    [[nodiscard]] const std::string &namespace_uri() const { return this->uri; }

    /// @brief finds the namespace index of the node ID on a server.
    /// @param namespaces the namespace URIs of the server, by index.
    /// @returns the node ID with the index of its namespace URI, which borrows the
    /// identifier of this node ID. Nothing if the server has no such namespace.
    [[nodiscard]] std::optional<UA_NodeId>
    resolve(const std::vector<std::string> &namespaces) const;

    /// @brief parses `nsu=<uri>;<id>`, `ns=<index>;<id>`, or `<id>` in namespace 0,
    /// where `<id>` is `i=`, `s=`, `g=`, or `b=` with its identifier. Prefixes match in
    /// any case.
    static NodeId parse(const std::string &field_name, x::json::Parser &parser);
    static std::pair<NodeId, x::errors::Error> parse(const std::string &node_id_str);
    /// @returns the node ID as `nsu=<uri>;<id>` when it names its namespace by URI, and
    /// as `NS=<index>;<id>` when it does not.
    [[nodiscard]] std::string to_string() const;
    /// @returns the node ID as `NS=<index>;<id>`.
    static std::string to_string(const UA_NodeId &node_id);
    /// @returns the node ID as `nsu=<uri>;<id>`, which stays valid when the server
    /// changes its namespace indexes. A node in namespace 0, or in a namespace that
    /// namespaces lacks, keeps its index.
    /// @param namespaces the namespace URIs of the server, by index.
    static std::string
    to_string(const UA_NodeId &node_id, const std::vector<std::string> &namespaces);
};

std::string node_class_to_string(const UA_NodeClass &node_class);

/// @brief a child that a browse found under a node.
struct Reference {
    /// @brief the node ID, with the namespace index of the connection that browsed.
    NodeId node;
    std::string browse_name;
    UA_NodeClass node_class = UA_NODECLASS_UNSPECIFIED;
};

/// @brief the children of one browsed node.
struct BrowseResult {
    /// @brief the error of the node's browse, such as BadNodeIdUnknown. The children
    /// are then empty.
    x::errors::Error err;
    std::vector<Reference> references;
    /// @brief whether the browse stopped at its limit before the server listed every
    /// child, so references holds only the first ones.
    bool truncated = false;
};

class Variant {
    UA_Variant var{};

public:
    Variant() { UA_Variant_init(&this->var); }
    explicit Variant(const UA_Variant &var) {
        UA_Variant_init(&this->var);
        UA_Variant_copy(&var, &this->var);
    }
    ~Variant() { UA_Variant_clear(&this->var); }

    Variant(const Variant &) = delete;
    Variant &operator=(const Variant &) = delete;

    Variant(Variant &&other) noexcept: var(other.var) { UA_Variant_init(&other.var); }
    Variant &operator=(Variant &&other) noexcept {
        if (this != &other) {
            UA_Variant_clear(&this->var);
            this->var = other.var;
            UA_Variant_init(&other.var);
        }
        return *this;
    }

    operator const UA_Variant &() const { return this->var; }
    [[nodiscard]] const UA_Variant &get() const { return this->var; }
    UA_Variant *ptr() { return &this->var; }
};

/// @brief owns a response of the client, whose type is UA_TYPES[Index], and clears it
/// on destruction.
template<typename T, std::size_t Index>
class Response {
    T res{};

public:
    Response() { UA_init(&this->res, &UA_TYPES[Index]); }
    explicit Response(const T &res): res(res) {}
    ~Response() { UA_clear(&this->res, &UA_TYPES[Index]); }

    Response(const Response &) = delete;
    Response &operator=(const Response &) = delete;

    Response(Response &&other) noexcept: res(other.res) {
        UA_init(&other.res, &UA_TYPES[Index]);
    }
    Response &operator=(Response &&other) noexcept {
        if (this != &other) {
            UA_clear(&this->res, &UA_TYPES[Index]);
            this->res = other.res;
            UA_init(&other.res, &UA_TYPES[Index]);
        }
        return *this;
    }

    operator const T &() const { return this->res; }
    [[nodiscard]] const T &get() const { return this->res; }
    T *ptr() { return &this->res; }
};

using ReadResponse = Response<UA_ReadResponse, UA_TYPES_READRESPONSE>;
using WriteResponse = Response<UA_WriteResponse, UA_TYPES_WRITERESPONSE>;
using BrowseResponse = Response<UA_BrowseResponse, UA_TYPES_BROWSERESPONSE>;
using BrowseNextResponse = Response<UA_BrowseNextResponse, UA_TYPES_BROWSENEXTRESPONSE>;

class WriteRequestBuilder {
    std::vector<UA_Variant> owned_variants;
    std::vector<UA_WriteValue> values;

public:
    WriteRequestBuilder() = default;
    ~WriteRequestBuilder() {
        for (auto &variant: owned_variants) {
            UA_Variant_clear(&variant);
        }
    }

    WriteRequestBuilder(const WriteRequestBuilder &) = delete;
    WriteRequestBuilder &operator=(const WriteRequestBuilder &) = delete;
    WriteRequestBuilder(WriteRequestBuilder &&) = delete;
    WriteRequestBuilder &operator=(WriteRequestBuilder &&) = delete;

    void clear() {
        for (auto &variant: owned_variants) {
            UA_Variant_clear(&variant);
        }
        owned_variants.clear();
        values.clear();
    }

    WriteRequestBuilder &add_value(const UA_NodeId &node_id, UA_Variant variant) {
        owned_variants.push_back(variant);
        UA_Variant_init(&variant);

        UA_WriteValue wv;
        UA_WriteValue_init(&wv);
        wv.nodeId = node_id;
        wv.attributeId = UA_ATTRIBUTEID_VALUE;
        wv.value.hasValue = true;
        wv.value.value = owned_variants.back();
        values.push_back(wv);
        return *this;
    }

    /// @brief adds the last sample of series, converted to target, the data type of the
    /// node. A null target converts to the type that matches the series.
    /// @returns x::errors::VALIDATION if the sample does not convert to target.
    x::errors::Error add_value(
        const UA_NodeId &node_id,
        const x::telem::Series &series,
        const UA_DataType *target
    );

    [[nodiscard]] UA_WriteRequest build() const {
        UA_WriteRequest req;
        UA_WriteRequest_init(&req);
        req.nodesToWrite = const_cast<UA_WriteValue *>(values.data());
        req.nodesToWriteSize = values.size();
        return req;
    }

    [[nodiscard]] size_t size() const { return values.size(); }
    [[nodiscard]] bool empty() const { return values.empty(); }
};

class ReadRequestBuilder {
    std::vector<UA_ReadValueId> ids;

public:
    ReadRequestBuilder() = default;
    ~ReadRequestBuilder() = default;

    ReadRequestBuilder(const ReadRequestBuilder &) = delete;
    ReadRequestBuilder &operator=(const ReadRequestBuilder &) = delete;
    ReadRequestBuilder(ReadRequestBuilder &&) = default;
    ReadRequestBuilder &operator=(ReadRequestBuilder &&) = default;

    ReadRequestBuilder &
    add_node(const UA_NodeId &node_id, UA_AttributeId attr = UA_ATTRIBUTEID_VALUE) {
        UA_ReadValueId rvid;
        UA_ReadValueId_init(&rvid);
        rvid.nodeId = node_id;
        rvid.attributeId = attr;
        ids.push_back(rvid);
        return *this;
    }

    [[nodiscard]] UA_ReadRequest build() const {
        UA_ReadRequest req;
        UA_ReadRequest_init(&req);
        req.nodesToRead = const_cast<UA_ReadValueId *>(ids.data());
        req.nodesToReadSize = ids.size();
        return req;
    }

    [[nodiscard]] size_t size() const { return ids.size(); }
    [[nodiscard]] bool empty() const { return ids.empty(); }
};

class LocalizedText {
    UA_LocalizedText text{};

public:
    LocalizedText() { UA_LocalizedText_init(&this->text); }
    explicit LocalizedText(const char *locale, const char *text) {
        this->text = UA_LOCALIZEDTEXT_ALLOC(locale, text);
    }
    ~LocalizedText() { UA_LocalizedText_clear(&this->text); }

    LocalizedText(const LocalizedText &) = delete;
    LocalizedText &operator=(const LocalizedText &) = delete;

    LocalizedText(LocalizedText &&other) noexcept: text(other.text) {
        UA_LocalizedText_init(&other.text);
    }
    LocalizedText &operator=(LocalizedText &&other) noexcept {
        if (this != &other) {
            UA_LocalizedText_clear(&this->text);
            this->text = other.text;
            UA_LocalizedText_init(&other.text);
        }
        return *this;
    }

    operator const UA_LocalizedText &() const { return this->text; }
    [[nodiscard]] const UA_LocalizedText &get() const { return this->text; }
};

class QualifiedName {
    UA_QualifiedName name{};

public:
    QualifiedName() { UA_QualifiedName_init(&this->name); }
    explicit QualifiedName(uint16_t ns, const char *name) {
        this->name = UA_QUALIFIEDNAME_ALLOC(ns, name);
    }
    ~QualifiedName() { UA_QualifiedName_clear(&this->name); }

    QualifiedName(const QualifiedName &) = delete;
    QualifiedName &operator=(const QualifiedName &) = delete;

    QualifiedName(QualifiedName &&other) noexcept: name(other.name) {
        UA_QualifiedName_init(&other.name);
    }
    QualifiedName &operator=(QualifiedName &&other) noexcept {
        if (this != &other) {
            UA_QualifiedName_clear(&this->name);
            this->name = other.name;
            UA_QualifiedName_init(&other.name);
        }
        return *this;
    }

    operator const UA_QualifiedName &() const { return this->name; }
    [[nodiscard]] const UA_QualifiedName &get() const { return this->name; }
};

/// @brief RAII wrapper for UA_String that automatically manages memory.
/// This class is move-only to prevent expensive copies.
class String {
    UA_String str{};

public:
    String() { UA_String_init(&this->str); }
    explicit String(const char *s) { this->str = UA_STRING_ALLOC(s); }
    ~String() { UA_String_clear(&this->str); }

    String(const String &) = delete;
    String &operator=(const String &) = delete;

    String(String &&other) noexcept: str(other.str) { UA_String_init(&other.str); }
    String &operator=(String &&other) noexcept {
        if (this != &other) {
            UA_String_clear(&this->str);
            this->str = other.str;
            UA_String_init(&other.str);
        }
        return *this;
    }

    operator const UA_String &() const { return this->str; }
    [[nodiscard]] const UA_String &get() const { return this->str; }
    /// @brief Get mutable pointer for output parameters
    UA_String *ptr() { return &this->str; }
};

/// @brief RAII wrapper for UA_ByteString that automatically manages memory.
/// This class is move-only to prevent expensive copies.
class ByteString {
    UA_ByteString str{};

public:
    ByteString() { UA_ByteString_init(&this->str); }
    ~ByteString() { UA_ByteString_clear(&this->str); }

    ByteString(const ByteString &) = delete;
    ByteString &operator=(const ByteString &) = delete;

    ByteString(ByteString &&other) noexcept: str(other.str) {
        UA_ByteString_init(&other.str);
    }
    ByteString &operator=(ByteString &&other) noexcept {
        if (this != &other) {
            UA_ByteString_clear(&this->str);
            this->str = other.str;
            UA_ByteString_init(&other.str);
        }
        return *this;
    }

    operator const UA_ByteString &() const { return this->str; }
    [[nodiscard]] const UA_ByteString &get() const { return this->str; }
    /// @brief Get mutable pointer for output parameters
    UA_ByteString *ptr() { return &this->str; }
};
// NOLINTEND(misc-explicit-constructor)
}
