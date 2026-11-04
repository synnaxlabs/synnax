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
    UA_NodeId id_{};
    /// @brief the namespace of the node when the node ID names it by URI. The namespace
    /// index of id_ is then not used.
    std::string namespace_uri_;

public:
    NodeId() { UA_NodeId_init(&id_); }

    explicit NodeId(const UA_NodeId &id) {
        UA_NodeId_init(&id_);
        UA_NodeId_copy(&id, &id_);
    }

    ~NodeId() { UA_NodeId_clear(&id_); }

    NodeId(const NodeId &) = delete;
    NodeId &operator=(const NodeId &) = delete;

    NodeId(NodeId &&other) noexcept:
        id_(other.id_), namespace_uri_(std::move(other.namespace_uri_)) {
        UA_NodeId_init(&other.id_);
    }

    NodeId &operator=(NodeId &&other) noexcept {
        if (this != &other) {
            UA_NodeId_clear(&id_);
            id_ = other.id_;
            namespace_uri_ = std::move(other.namespace_uri_);
            UA_NodeId_init(&other.id_);
        }
        return *this;
    }

    [[nodiscard]] bool is_null() const { return UA_NodeId_isNull(&id_); }
    [[nodiscard]] const std::string &namespace_uri() const { return namespace_uri_; }

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
    UA_Variant var_{};

public:
    Variant() { UA_Variant_init(&var_); }
    explicit Variant(const UA_Variant &var) {
        UA_Variant_init(&var_);
        UA_Variant_copy(&var, &var_);
    }
    ~Variant() { UA_Variant_clear(&var_); }

    Variant(const Variant &) = delete;
    Variant &operator=(const Variant &) = delete;

    Variant(Variant &&other) noexcept: var_(other.var_) {
        UA_Variant_init(&other.var_);
    }
    Variant &operator=(Variant &&other) noexcept {
        if (this != &other) {
            UA_Variant_clear(&var_);
            var_ = other.var_;
            UA_Variant_init(&other.var_);
        }
        return *this;
    }

    operator const UA_Variant &() const { return var_; }
    [[nodiscard]] const UA_Variant &get() const { return var_; }
    UA_Variant *ptr() { return &var_; }
};

/// @brief owns a response of the client, whose type is UA_TYPES[Index], and clears it
/// on destruction.
template<typename T, std::size_t Index>
class Response {
    T res_{};

public:
    Response() { UA_init(&res_, &UA_TYPES[Index]); }
    explicit Response(const T &res): res_(res) {}
    ~Response() { UA_clear(&res_, &UA_TYPES[Index]); }

    Response(const Response &) = delete;
    Response &operator=(const Response &) = delete;

    Response(Response &&other) noexcept: res_(other.res_) {
        UA_init(&other.res_, &UA_TYPES[Index]);
    }
    Response &operator=(Response &&other) noexcept {
        if (this != &other) {
            UA_clear(&res_, &UA_TYPES[Index]);
            res_ = other.res_;
            UA_init(&other.res_, &UA_TYPES[Index]);
        }
        return *this;
    }

    operator const T &() const { return res_; }
    [[nodiscard]] const T &get() const { return res_; }
    T *ptr() { return &res_; }
};

using ReadResponse = Response<UA_ReadResponse, UA_TYPES_READRESPONSE>;
using WriteResponse = Response<UA_WriteResponse, UA_TYPES_WRITERESPONSE>;
using BrowseResponse = Response<UA_BrowseResponse, UA_TYPES_BROWSERESPONSE>;
using BrowseNextResponse = Response<UA_BrowseNextResponse, UA_TYPES_BROWSENEXTRESPONSE>;

class WriteRequestBuilder {
    std::vector<UA_Variant> owned_variants_;
    std::vector<UA_WriteValue> values_;

public:
    WriteRequestBuilder() = default;
    ~WriteRequestBuilder() {
        for (auto &variant: owned_variants_) {
            UA_Variant_clear(&variant);
        }
    }

    WriteRequestBuilder(const WriteRequestBuilder &) = delete;
    WriteRequestBuilder &operator=(const WriteRequestBuilder &) = delete;
    WriteRequestBuilder(WriteRequestBuilder &&) = delete;
    WriteRequestBuilder &operator=(WriteRequestBuilder &&) = delete;

    void clear() {
        for (auto &variant: owned_variants_) {
            UA_Variant_clear(&variant);
        }
        owned_variants_.clear();
        values_.clear();
    }

    WriteRequestBuilder &add_value(const UA_NodeId &node_id, UA_Variant variant) {
        owned_variants_.push_back(variant);
        UA_Variant_init(&variant);

        UA_WriteValue wv;
        UA_WriteValue_init(&wv);
        wv.nodeId = node_id;
        wv.attributeId = UA_ATTRIBUTEID_VALUE;
        wv.value.hasValue = true;
        wv.value.value = owned_variants_.back();
        values_.push_back(wv);
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
        req.nodesToWrite = const_cast<UA_WriteValue *>(values_.data());
        req.nodesToWriteSize = values_.size();
        return req;
    }

    [[nodiscard]] size_t size() const { return values_.size(); }
    [[nodiscard]] bool empty() const { return values_.empty(); }
};

class ReadRequestBuilder {
    std::vector<UA_ReadValueId> ids_;

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
        ids_.push_back(rvid);
        return *this;
    }

    [[nodiscard]] UA_ReadRequest build() const {
        UA_ReadRequest req;
        UA_ReadRequest_init(&req);
        req.nodesToRead = const_cast<UA_ReadValueId *>(ids_.data());
        req.nodesToReadSize = ids_.size();
        return req;
    }

    [[nodiscard]] size_t size() const { return ids_.size(); }
    [[nodiscard]] bool empty() const { return ids_.empty(); }
};

class LocalizedText {
    UA_LocalizedText text_{};

public:
    LocalizedText() { UA_LocalizedText_init(&text_); }
    explicit LocalizedText(const char *locale, const char *text) {
        text_ = UA_LOCALIZEDTEXT_ALLOC(locale, text);
    }
    ~LocalizedText() { UA_LocalizedText_clear(&text_); }

    LocalizedText(const LocalizedText &) = delete;
    LocalizedText &operator=(const LocalizedText &) = delete;

    LocalizedText(LocalizedText &&other) noexcept: text_(other.text_) {
        UA_LocalizedText_init(&other.text_);
    }
    LocalizedText &operator=(LocalizedText &&other) noexcept {
        if (this != &other) {
            UA_LocalizedText_clear(&text_);
            text_ = other.text_;
            UA_LocalizedText_init(&other.text_);
        }
        return *this;
    }

    operator const UA_LocalizedText &() const { return text_; }
    [[nodiscard]] const UA_LocalizedText &get() const { return text_; }
};

class QualifiedName {
    UA_QualifiedName name_{};

public:
    QualifiedName() { UA_QualifiedName_init(&name_); }
    explicit QualifiedName(uint16_t ns, const char *name) {
        name_ = UA_QUALIFIEDNAME_ALLOC(ns, name);
    }
    ~QualifiedName() { UA_QualifiedName_clear(&name_); }

    QualifiedName(const QualifiedName &) = delete;
    QualifiedName &operator=(const QualifiedName &) = delete;

    QualifiedName(QualifiedName &&other) noexcept: name_(other.name_) {
        UA_QualifiedName_init(&other.name_);
    }
    QualifiedName &operator=(QualifiedName &&other) noexcept {
        if (this != &other) {
            UA_QualifiedName_clear(&name_);
            name_ = other.name_;
            UA_QualifiedName_init(&other.name_);
        }
        return *this;
    }

    operator const UA_QualifiedName &() const { return name_; }
    [[nodiscard]] const UA_QualifiedName &get() const { return name_; }
};

/// @brief RAII wrapper for UA_String that automatically manages memory.
/// This class is move-only to prevent expensive copies.
class String {
    UA_String str_{};

public:
    String() { UA_String_init(&str_); }
    explicit String(const char *s) { str_ = UA_STRING_ALLOC(s); }
    ~String() { UA_String_clear(&str_); }

    String(const String &) = delete;
    String &operator=(const String &) = delete;

    String(String &&other) noexcept: str_(other.str_) { UA_String_init(&other.str_); }
    String &operator=(String &&other) noexcept {
        if (this != &other) {
            UA_String_clear(&str_);
            str_ = other.str_;
            UA_String_init(&other.str_);
        }
        return *this;
    }

    operator const UA_String &() const { return str_; }
    [[nodiscard]] const UA_String &get() const { return str_; }
    /// @brief Get mutable pointer for output parameters
    UA_String *ptr() { return &str_; }
};

/// @brief RAII wrapper for UA_ByteString that automatically manages memory.
/// This class is move-only to prevent expensive copies.
class ByteString {
    UA_ByteString str_{};

public:
    ByteString() { UA_ByteString_init(&str_); }
    ~ByteString() { UA_ByteString_clear(&str_); }

    ByteString(const ByteString &) = delete;
    ByteString &operator=(const ByteString &) = delete;

    ByteString(ByteString &&other) noexcept: str_(other.str_) {
        UA_ByteString_init(&other.str_);
    }
    ByteString &operator=(ByteString &&other) noexcept {
        if (this != &other) {
            UA_ByteString_clear(&str_);
            str_ = other.str_;
            UA_ByteString_init(&other.str_);
        }
        return *this;
    }

    operator const UA_ByteString &() const { return str_; }
    [[nodiscard]] const UA_ByteString &get() const { return str_; }
    /// @brief Get mutable pointer for output parameters
    UA_ByteString *ptr() { return &str_; }
};
// NOLINTEND(misc-explicit-constructor)
}
