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
    struct Payload {
        std::string type;
        std::string data;
    };
    Payload payload;

public:
    /// @brief constructs the default version fo the error with TYPE_NIL.
    Error(): payload{.type = TYPE_NIL, .data = {}} {}

    /// @brief constructs the error from a particular string data and data.
    Error(std::string type, std::string data):
        payload{.type = std::move(type), .data = std::move(data)} {}

    /// @brief constructs the error from a particular string freighter:Error and
    /// data.
    Error(const Error &err, std::string data):
        payload{.type = err.type(), .data = std::move(data)} {}

    /// @brief constructs the provided error from a string. If the string is of the
    /// form "type---data", the type and data will be extracted from the string.
    /// Otherwise, the string is assumed to be the type.
    explicit Error(const std::string &err_or_type):
        payload{.type = err_or_type, .data = {}} {
        const size_t pos = err_or_type.find("---");
        if (pos == std::string::npos) return;
        this->payload.type = err_or_type.substr(0, pos);
        this->payload.data = err_or_type.substr(pos + 3);
    }

    /// @brief constructs the error from its protobuf representation.
    explicit Error(const ::errors::PBPayload &err):
        payload{.type = err.type(), .data = err.data()} {}

    /// @brief returns the general class that the error belongs to, which handling
    /// logic branches on, especially for errors sent over the network.
    [[nodiscard]] const std::string &type() const { return this->payload.type; }

    /// @brief returns data about the error: a message, or sometimes a serialized
    /// object.
    [[nodiscard]] const std::string &data() const { return this->payload.data; }

    [[nodiscard]] Error sub(const std::string &type_extension) const {
        return Error(this->type() + "." + type_extension);
    }

    [[nodiscard]] Error reparent(const Error &parent) const {
        const auto pos = this->type().rfind('.');
        if (pos == std::string::npos) return *this;
        return {parent.type() + this->type().substr(pos), this->data()};
    }

    /// @brief returns a copy of the error with context before its data, as
    /// "context: data". Wrapping NIL returns NIL.
    [[nodiscard]] Error wrap(const std::string &context) const {
        if (this->ok()) return *this;
        return {*this, context + ": " + this->data()};
    }

    /// @brief copy constructor.
    Error(const Error &other) = default;

    /// @returns true if the error if of TYPE_NIL, and false otherwise.
    [[nodiscard]] bool ok() const { return this->type() == TYPE_NIL; }

    /// @returns a string formatted error message.
    [[nodiscard]] std::string message() const {
        return "[" + this->type() + "] " + this->data();
    }

    explicit operator bool() const { return !ok(); }

    friend std::ostream &operator<<(std::ostream &os, const Error &err) {
        os << err.message();
        return os;
    }

    /// @brief checks if the error matches the provided error, as the string overload
    /// does.
    [[nodiscard]] bool matches(const Error &other) const {
        return matches(other.type());
    }

    /// @brief checks if the error matches the provided type. The error matches if its
    /// type is equal to the provided type or is a sub-type of it: "a.b" matches "a",
    /// but "ab" does not.
    [[nodiscard]] bool matches(const std::string &other) const {
        const auto &type = this->type();
        if (!type.starts_with(other)) return false;
        return type.size() == other.size() || type[other.size()] == '.';
    }

    /// @brief checks if the error matches any of the provided types.
    [[nodiscard]] bool matches(const std::vector<std::string> &types) const {
        return std::ranges::any_of(types, [this](const std::string &t) {
            return matches(t);
        });
    }

    /// @brief checks if the error matches any of the provided errors.
    [[nodiscard]] bool matches(const std::vector<Error> &errors) const {
        return std::ranges::any_of(errors, [this](const Error &e) {
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

    bool operator==(const Error &other) const { return this->type() == other.type(); };

    bool operator!=(const Error &other) const { return this->type() != other.type(); };

    bool operator==(const std::string &other) const { return this->type() == other; };

    bool operator!=(const std::string &other) const { return this->type() != other; };
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
/// @brief work that stopped because its caller cancelled it.
const Error CANCELLED = SY.sub("cancelled");
const Error CONTROL = SY.sub("control");
const Error UNAUTHORIZED = CONTROL.sub("unauthorized");
}
