// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#pragma once

/// external
#include "open62541/types.h"

namespace opc {
/// @brief RAII wrapper for UA_NodeId that automatically manages memory.
/// This eliminates manual UA_NodeId_clear() calls and prevents double-free bugs.
class NodeId {
    UA_NodeId id_;

public:
    /// @brief Default constructor - creates a null NodeId
    NodeId() { UA_NodeId_init(&id_); }

    /// @brief Construct from a raw UA_NodeId (takes ownership)
    explicit NodeId(const UA_NodeId &id) { UA_NodeId_copy(&id, &id_); }

    /// @brief Destructor - automatically cleans up allocated memory
    ~NodeId() { UA_NodeId_clear(&id_); }

    /// @brief Copy constructor - creates a deep copy
    NodeId(const NodeId &other) { UA_NodeId_copy(&other.id_, &id_); }

    /// @brief Copy assignment - creates a deep copy
    NodeId &operator=(const NodeId &other) {
        if (this != &other) {
            UA_NodeId_clear(&id_);
            UA_NodeId_copy(&other.id_, &id_);
        }
        return *this;
    }

    /// @brief Move constructor - transfers ownership
    NodeId(NodeId &&other) noexcept: id_(other.id_) {
        UA_NodeId_init(&other.id_);
    }

    /// @brief Move assignment - transfers ownership
    NodeId &operator=(NodeId &&other) noexcept {
        if (this != &other) {
            UA_NodeId_clear(&id_);
            id_ = other.id_;
            UA_NodeId_init(&other.id_);
        }
        return *this;
    }

    /// @brief Get const reference to underlying UA_NodeId (for read-only
    /// operations)
    [[nodiscard]] const UA_NodeId &get() const { return id_; }

    /// @brief Get mutable reference to underlying UA_NodeId (use with caution -
    /// caller must not call UA_NodeId_clear on the returned reference)
    UA_NodeId &get_mut() { return id_; }

    /// @brief Check if this is a null NodeId
    [[nodiscard]] bool is_null() const { return UA_NodeId_isNull(&id_); }
};

/// @brief RAII wrapper for UA_Variant that automatically manages memory.
class Variant {
    UA_Variant var_;

public:
    Variant() { UA_Variant_init(&var_); }
    explicit Variant(const UA_Variant &var) { UA_Variant_copy(&var, &var_); }
    ~Variant() { UA_Variant_clear(&var_); }

    Variant(const Variant &other) { UA_Variant_copy(&other.var_, &var_); }
    Variant &operator=(const Variant &other) {
        if (this != &other) {
            UA_Variant_clear(&var_);
            UA_Variant_copy(&other.var_, &var_);
        }
        return *this;
    }

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

    [[nodiscard]] const UA_Variant &get() const { return var_; }
    UA_Variant &get_mut() { return var_; }
    UA_Variant *ptr() { return &var_; }
};

/// @brief RAII wrapper for UA_ReadResponse that automatically manages memory.
class ReadResponse {
    UA_ReadResponse res_;

public:
    ReadResponse() = default;
    explicit ReadResponse(const UA_ReadResponse &res) { res_ = res; }
    ~ReadResponse() { UA_ReadResponse_clear(&res_); }

    ReadResponse(const ReadResponse &) = delete;
    ReadResponse &operator=(const ReadResponse &) = delete;

    ReadResponse(ReadResponse &&other) noexcept: res_(other.res_) {
        UA_ReadResponse_init(&other.res_);
    }
    ReadResponse &operator=(ReadResponse &&other) noexcept {
        if (this != &other) {
            UA_ReadResponse_clear(&res_);
            res_ = other.res_;
            UA_ReadResponse_init(&other.res_);
        }
        return *this;
    }

    [[nodiscard]] const UA_ReadResponse &get() const { return res_; }
    UA_ReadResponse &get_mut() { return res_; }
};

/// @brief RAII wrapper for UA_WriteResponse that automatically manages memory.
class WriteResponse {
    UA_WriteResponse res_;

public:
    WriteResponse() = default;
    explicit WriteResponse(const UA_WriteResponse &res) { res_ = res; }
    ~WriteResponse() { UA_WriteResponse_clear(&res_); }

    WriteResponse(const WriteResponse &) = delete;
    WriteResponse &operator=(const WriteResponse &) = delete;

    WriteResponse(WriteResponse &&other) noexcept: res_(other.res_) {
        UA_WriteResponse_init(&other.res_);
    }
    WriteResponse &operator=(WriteResponse &&other) noexcept {
        if (this != &other) {
            UA_WriteResponse_clear(&res_);
            res_ = other.res_;
            UA_WriteResponse_init(&other.res_);
        }
        return *this;
    }

    [[nodiscard]] const UA_WriteResponse &get() const { return res_; }
    UA_WriteResponse &get_mut() { return res_; }
};

/// @brief RAII wrapper for UA_LocalizedText that automatically manages memory.
class LocalizedText {
    UA_LocalizedText text_;

public:
    LocalizedText() { UA_LocalizedText_init(&text_); }
    explicit LocalizedText(const char *locale, const char *text) {
        text_ = UA_LOCALIZEDTEXT_ALLOC(locale, text);
    }
    ~LocalizedText() { UA_LocalizedText_clear(&text_); }

    LocalizedText(const LocalizedText &other) {
        UA_LocalizedText_copy(&other.text_, &text_);
    }
    LocalizedText &operator=(const LocalizedText &other) {
        if (this != &other) {
            UA_LocalizedText_clear(&text_);
            UA_LocalizedText_copy(&other.text_, &text_);
        }
        return *this;
    }

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

    [[nodiscard]] const UA_LocalizedText &get() const { return text_; }
    UA_LocalizedText &get_mut() { return text_; }
};

/// @brief RAII wrapper for UA_QualifiedName that automatically manages memory.
class QualifiedName {
    UA_QualifiedName name_;

public:
    QualifiedName() { UA_QualifiedName_init(&name_); }
    explicit QualifiedName(uint16_t ns, const char *name) {
        name_ = UA_QUALIFIEDNAME_ALLOC(ns, name);
    }
    ~QualifiedName() { UA_QualifiedName_clear(&name_); }

    QualifiedName(const QualifiedName &other) {
        UA_QualifiedName_copy(&other.name_, &name_);
    }
    QualifiedName &operator=(const QualifiedName &other) {
        if (this != &other) {
            UA_QualifiedName_clear(&name_);
            UA_QualifiedName_copy(&other.name_, &name_);
        }
        return *this;
    }

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

    [[nodiscard]] const UA_QualifiedName &get() const { return name_; }
    UA_QualifiedName &get_mut() { return name_; }
};

/// @brief RAII wrapper for UA_String that automatically manages memory.
class String {
    UA_String str_;

public:
    String() { UA_String_init(&str_); }
    explicit String(const char *s) { str_ = UA_STRING_ALLOC(s); }
    ~String() { UA_String_clear(&str_); }

    String(const String &other) { UA_String_copy(&other.str_, &str_); }
    String &operator=(const String &other) {
        if (this != &other) {
            UA_String_clear(&str_);
            UA_String_copy(&other.str_, &str_);
        }
        return *this;
    }

    String(String &&other) noexcept: str_(other.str_) {
        UA_String_init(&other.str_);
    }
    String &operator=(String &&other) noexcept {
        if (this != &other) {
            UA_String_clear(&str_);
            str_ = other.str_;
            UA_String_init(&other.str_);
        }
        return *this;
    }

    [[nodiscard]] const UA_String &get() const { return str_; }
    UA_String &get_mut() { return str_; }
    UA_String *ptr() { return &str_; }
};

/// @brief RAII wrapper for UA_ByteString that automatically manages memory.
class ByteString {
    UA_ByteString str_;

public:
    ByteString() { UA_ByteString_init(&str_); }
    ~ByteString() { UA_ByteString_clear(&str_); }

    ByteString(const ByteString &other) {
        UA_ByteString_copy(&other.str_, &str_);
    }
    ByteString &operator=(const ByteString &other) {
        if (this != &other) {
            UA_ByteString_clear(&str_);
            UA_ByteString_copy(&other.str_, &str_);
        }
        return *this;
    }

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

    [[nodiscard]] const UA_ByteString &get() const { return str_; }
    UA_ByteString &get_mut() { return str_; }
    UA_ByteString *ptr() { return &str_; }
};
}
