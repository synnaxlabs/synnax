// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstdlib>
#include <string>
#include <type_traits>
#include <vector>

#include "glog/logging.h"

#include "x/cpp/caseconv/caseconv.h"
#include "x/cpp/errors/errors.h"

namespace x::env {
class Parser {
    /// @brief the environment variable prefix.
    std::string prefix;

    template<typename T>
    static T convert_value(const std::string &value, const T &default_value) {
        if constexpr (std::is_same_v<T, bool>) {
            if (!default_value) return (value == "true" || value == "1");
            return !(value == "false" || value == "0");
        }
        if constexpr (std::is_same_v<T, std::string>) return value;
        if constexpr (std::is_floating_point_v<T>)
            return static_cast<T>(std::stold(value));
        if constexpr (std::is_unsigned_v<T>) return static_cast<T>(std::stoull(value));
        if constexpr (std::is_integral_v<T>) return static_cast<T>(std::stoll(value));
        throw std::runtime_error("Unsupported type");
    }

public:
    /// @brief any errors encountered during parsing.
    std::vector<errors::Error> errors;

    explicit Parser(std::string prefix = ""): prefix(std::move(prefix)) {
        if (!this->prefix.empty() && this->prefix.back() != '_') this->prefix += '_';
    }

    template<typename T>
    T field(const std::string &name, const T &default_value) {
        auto screaming_name = caseconv::snake_to_scream(this->prefix + name);
        const char *value = std::getenv(screaming_name.c_str());
        if (value == nullptr) return default_value;

        VLOG(1) << "Loaded " << screaming_name << " from environment variable.";
        try {
            return convert_value(std::string(value), default_value);
        } catch (const std::exception &e) {
            LOG(WARNING) << "Failed to convert environment variable " << screaming_name
                         << " to type " << typeid(T).name() << ": " << e.what();
            this->field_err(
                name,
                std::string("failed to convert ") + screaming_name + ": " + e.what()
            );
            return default_value;
        }
    }

    /// @brief binds an error to the given field name.
    void field_err(const std::string &name, const std::string &message) {
        this->errors.emplace_back(errors::VALIDATION, name + ": " + message);
    }

    /// @brief binds an error to the given field name from an existing error.
    void field_err(const std::string &name, const errors::Error &err) {
        this->field_err(name, err.data);
    }

    /// @returns true if no errors have been accumulated.
    [[nodiscard]] bool ok() const { return this->errors.empty(); }

    /// @brief returns the first error encountered during parsing, or NIL.
    [[nodiscard]] errors::Error error() const {
        if (this->errors.empty()) return errors::NIL;
        return this->errors.at(0);
    }
};

template<typename T>
T load(const std::string &name, const T &default_value) {
    static Parser default_parser;
    return default_parser.field(name, default_value);
}

inline int set(const std::string &name, const std::string &value) {
#ifdef _WIN32
    return _putenv_s(name.c_str(), value.c_str());
#else
    return setenv(name.c_str(), value.c_str(), 1);
#endif
}

inline int unset(const std::string &name) {
#ifdef _WIN32
    return _putenv_s(name.c_str(), "");
#else
    return unsetenv(name.c_str());
#endif
}
}
