// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// std
#include <memory>
#include <string>
#include <type_traits>
#include <vector>

/// internal
#include "x/cpp/xerrors/errors.h"

namespace xargs {
class Parser {
    template<typename... Args>
    std::pair<std::string, bool> find_arg(const Args&... names) {
        for (size_t i = 0; i < argv_.size(); i++) {
            if ((... || (argv_[i] == names)))
                if (i + 1 < argv_.size())
                    return {argv_[i + 1], true};
        }
        return {"", false};
    }

    template<typename T>
    T parse_value(const std::string& value, const std::string& name, const char* error_msg) {
        try {
            if constexpr (std::is_same_v<T, std::string>)
                return value;
            else if constexpr (std::is_floating_point_v<T>)
                return static_cast<T>(std::stold(value));
            else if constexpr (std::is_integral_v<T> && !std::is_same_v<T, bool>)
                return static_cast<T>(std::stoll(value));
            else if constexpr (std::is_same_v<T, bool>)
                return value == "true" || value == "1";
            else if constexpr (std::is_same_v<T, const char*>)
                return value.c_str();
            else {
                errors.push_back(xerrors::Error(name, "Unsupported type"));
                return T();
            }
        } catch (const std::exception&) {
            errors.push_back(xerrors::Error(name, error_msg));
            return T();
        }
    }

    template<typename T>
    T handle_required(const std::string& name, const char* error_msg) {
        const auto [value, found] = find_arg(name);
        if (!found) {
            errors.push_back(xerrors::Error(name, "Required argument not found"));
            return T();
        }
        return parse_value<T>(value, name, error_msg);
    }

    // Add a new helper method for flag checking
    template<typename... Args>
    bool has_arg(const Args&... names) {
        for (const auto& arg : argv_) {
            if ((... || (arg == names)))
                return true;
        }
        return false;
    }

public:
    std::vector<std::string> argv_;
    std::vector<xerrors::Error> errors;

    Parser() = default;

    explicit Parser(const int argc, char* argv[]) : 
        argv_(argv, argv + argc) {}
    
    explicit Parser(std::vector<std::string> argv) : 
        argv_(std::move(argv)) {}

    template<typename T>
    T required(const std::string& name) {
        return handle_required<T>(name, "Invalid value");
    }

    template<typename T>
    T optional(const std::string& name, const T& default_value) {
        const auto [value, found] = find_arg(name);
        if (!found) return default_value;
        return parse_value<T>(value, name, "Invalid value");
    }

    std::string optional(const std::string& name, const char* default_value) {
        const auto [value, found] = find_arg(name);
        if (!found) return std::string(default_value);
        return value;
    }

    template<typename... Args>
    [[nodiscard]] bool flag(const Args&... names) {
        // Just check if the flag exists, don't look for a value after it
        return has_arg(names...);
    }

    [[nodiscard]] xerrors::Error error() const {
        if (errors.empty()) return xerrors::NIL;
        return errors.at(0);
    }

    std::string at(const int index, const std::string &error_msg) {
        if (static_cast<size_t>(index) >= argv_.size()) {
            errors.push_back(xerrors::Error("index", error_msg));
            return "";
        }
        return argv_[index];
    };
};

template<>
inline std::string Parser::required<std::string>(const std::string& name) {
    const auto [value, found] = find_arg(name);
    if (!found) {
        errors.push_back(xerrors::Error(name, "Required argument not found"));
        return "";
    }
    return value;
}

template<>
inline bool Parser::required<bool>(const std::string& name) {
    const auto [value, found] = find_arg(name);
    if (!found) {
        errors.push_back(xerrors::Error(name, "Required argument not found"));
        return false;
    }
    return value == "true" || value == "1";
}
}