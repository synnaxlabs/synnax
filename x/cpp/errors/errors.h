// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <algorithm>
#include <iostream>
#include <string>
#include <vector>

#include "x/go/errors/x/go/errors/errors.pb.h"

namespace x::errors {
const std::string TYPE_NIL = "nil";
const std::string TYPE_UNKNOWN = "unknown";

/// @brief a network transportable error with a type and string encoded data.
class Error {
public:
    /// @brief defines the general class that this particular error belongs to.
    /// Typically used to identify handling logic for errors (especially ones
    /// transported over the network).
    std::string type;
    /// @brief data related to the error. This is typically a message, but can
    /// sometimes be a serialized object.
    std::string data;

    /// @brief constructs the default version fo the error with TYPE_NIL.
    Error(): type(TYPE_NIL) {}

    /// @brief constructs the error from a particular string data and data.
    Error(std::string type, std::string data):
        type(std::move(type)), data(std::move(data)) {}

    /// @brief constructs the error from a particular string freighter:Error and
    /// data.
    Error(const Error &err, std::string data): type(err.type), data(std::move(data)) {}

    /// @brief constructs the provided error from a string. If the string is of the
    /// form "type---data", the type and data will be extracted from the string.
    /// Otherwise, the string is assumed to be the type.
    explicit Error(const std::string &err_or_type): type(err_or_type) {
        const size_t pos = err_or_type.find("---");
        if (pos == std::string::npos) return;
        type = err_or_type.substr(0, pos);
        data = err_or_type.substr(pos + 3);
    }

    /// @brief constructs the error from its protobuf representation.
    explicit Error(const ::errors::PBPayload &err):
        type(err.type()), data(err.data()) {}

    [[nodiscard]] Error sub(const std::string &type_extension) const {
        return Error(type + "." + type_extension);
    }

    [[nodiscard]] Error reparent(const Error &parent) const {
        const auto pos = type.rfind('.');
        if (pos == std::string::npos) return *this;
        return {parent.type + type.substr(pos), this->data};
    }

    /// @brief copy constructor.
    Error(const Error &other) = default;

    /// @returns true if the error if of TYPE_NIL, and false otherwise.
    [[nodiscard]] bool ok() const { return type == TYPE_NIL; }

    /// @returns a string formatted error message.
    [[nodiscard]] std::string message() const { return "[" + type + "] " + data; }

    explicit operator bool() const { return !ok(); }

    friend std::ostream &operator<<(std::ostream &os, const Error &err) {
        os << err.message();
        return os;
    }

    /// @brief checks if the error matches the provided error. The error matches if
    /// the provided type is equal to or is a prefix of this errors type.
    [[nodiscard]] bool matches(const Error &other) const { return matches(other.type); }

    /// @brief checks if the error matches the provided type. The error matches if
    /// the provided type is equal to or is a prefix of this errors type.
    [[nodiscard]] bool matches(const std::string &other) const {
        const auto loc = std::mismatch(other.begin(), other.end(), type.begin()).first;
        return loc == other.end();
    }

    //// @brief checks if any of the provided types match the error. An error
    /// matches if
    /// the provided type is equal to or is a prefix of this errors type.
    [[nodiscard]] bool matches(const std::vector<std::string> &types) const {
        return std::any_of(types.begin(), types.end(), [this](const std::string &t) {
            return matches(t);
        });
    }

    /// @brief checks if any of the provided errors match the error. An error
    /// matches if the provided type is equal to or is a prefix of this errors type.
    [[nodiscard]] bool matches(const std::vector<Error> &errors) const {
        return std::any_of(errors.begin(), errors.end(), [this](const Error &e) {
            return matches(e);
        });
    }

    /// @brief if the error matches the provided error, 'skips' the error by
    /// returning nil, otherwise returns the error.
    [[nodiscard]] Error skip(const Error &ignore) const {
        if (this->matches(ignore)) return {TYPE_NIL, ""};
        return *this;
    }

    /// @brief if the error matches any the provided errors, 'skips' the error by
    /// returning nil, otherwise returns the error.
    [[nodiscard]] Error skip(const std::vector<Error> &ignore) const {
        if (this->matches(ignore)) return {TYPE_NIL, ""};
        return *this;
    }

    /// @brief if the error matches the provided type, 'skips' the error by
    /// returning nil, otherwise returns the error.
    [[nodiscard]] Error skip(const std::string &other) const {
        if (this->matches(other)) return {TYPE_NIL, ""};
        return *this;
    }

    bool operator==(const Error &other) const { return type == other.type; };

    bool operator!=(const Error &other) const { return type != other.type; };

    bool operator==(const std::string &other) const { return type == other; };

    bool operator!=(const std::string &other) const { return type != other; };
};

const Error UNKNOWN = {TYPE_UNKNOWN, ""};
const Error NIL = {TYPE_NIL, ""};
const Error SY("sy");
const Error VALIDATION = SY.sub("validation");
const Error QUERY = SY.sub("query");
const Error MULTIPLE_RESULTS = QUERY.sub("multiple_results");
const Error NOT_FOUND = QUERY.sub("not_found");
const Error NOT_SUPPORTED = SY.sub("not_supported");

const Error INTERNAL = SY.sub("internal");
const Error UNEXPECTED = SY.sub("unexpected");
const Error CONTROL = SY.sub("control");
const Error UNAUTHORIZED = CONTROL.sub("unauthorized");
}
