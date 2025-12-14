// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <fstream>
#include <map>
#include <sstream>
#include <string>
#include <type_traits>
#include <unordered_map>

#include "nlohmann/json.hpp"

#include "x/cpp/xerrors/errors.h"
#include "x/cpp/xpath/xpath.h"

using json = nlohmann::json;

/// @brief general utilities for parsing configurations.
namespace xjson {

/// @brief Type trait to detect std::vector types
template<typename T>
// NOLINTNEXTLINE(readability-identifier-naming) - follows STL naming convention (std::is_same)
struct is_vector : std::false_type {
    using value_type = T;
};

template<typename T>
struct is_vector<std::vector<T>> : std::true_type {
    using value_type = T;
};

template<typename T>
// NOLINTNEXTLINE(readability-identifier-naming) - follows STL naming convention (std::is_same_v)
inline constexpr bool is_vector_v = is_vector<T>::value;

/// @brief Type trait to detect std::map and std::unordered_map types with string or
/// numeric keys
template<typename T>
// NOLINTNEXTLINE(readability-identifier-naming) - follows STL naming convention (std::is_same)
struct is_map : std::false_type {
    using key_type = void;
    using value_type = T;
};

template<typename K, typename V>
struct is_map<std::map<K, V>>
    : std::conditional_t<
          std::is_same_v<K, std::string> || std::is_arithmetic_v<K>,
          std::true_type,
          std::false_type> {
    using key_type = K;
    using value_type = V;
};

template<typename K, typename V>
struct is_map<std::unordered_map<K, V>>
    : std::conditional_t<
          std::is_same_v<K, std::string> || std::is_arithmetic_v<K>,
          std::true_type,
          std::false_type> {
    using key_type = K;
    using value_type = V;
};

template<typename T>
// NOLINTNEXTLINE(readability-identifier-naming) - follows STL naming convention (std::is_same_v)
inline constexpr bool is_map_v = is_map<T>::value;

/// @brief a utility class for improving the experience of parsing JSON-based
/// configurations.
class Parser {
    /// @brief the JSON configuration being parsed.
    json config;
    /// @brief noop means the parser should fail fast.
    bool noop = false;

    Parser(
        json config,
        std::shared_ptr<std::vector<json>> errors,
        std::string path_prefix
    ):
        config(std::move(config)),
        path_prefix(std::move(path_prefix)),
        errors(std::move(errors)) {}

    /// @brief Core parsing logic - handles all type conversions in one place
    template<typename T>
    T parse_value(const std::string &path, const json &j);

    /// @brief Helper to convert JSON key to map key type
    template<typename K>
    std::pair<K, bool>
    convert_key(const std::string &json_key, const std::string &path) {
        if constexpr (std::is_same_v<K, std::string>)
            return {json_key, true};
        else if constexpr (std::is_arithmetic_v<K>) {
            try {
                K map_key;
                if constexpr (std::is_integral_v<K>)
                    map_key = static_cast<K>(std::stoll(json_key));
                else
                    map_key = static_cast<K>(std::stod(json_key));
                return {map_key, true};
            } catch (const std::exception &) {
                field_err(path, "Invalid numeric key: '" + json_key + "'");
                return {K{}, false};
            }
        }
        return {K{}, false};
    }

    /// @brief Wrapper for iterator-based access
    template<typename T>
    T get(const std::string &path, const json::iterator &iter) {
        return parse_value<T>(path, *iter);
    }

    /// @brief Helper method to parse JSON and handle errors
    void parse_with_err_handling(const std::function<json()> &parser) {
        try {
            config = parser();
        } catch (const json::parse_error &e) {
            field_err("", e.what());
            noop = true;
        }
    }

public:
    /// @brief used for tracking the path of a child parser.
    const std::string path_prefix;
    /// @brief the current list of accumulated errors.
    std::shared_ptr<std::vector<json>> errors;

    /// @brief constructs a parser for accessing values on the given JSON
    /// configuration.
    explicit Parser(json config):
        config(std::move(config)), errors(std::make_shared<std::vector<json>>()) {}

    /// @brief constructs a parser for accessing values on the given stringified
    /// JSON configuration. If the string is not valid JSON, immediately binds an
    /// error to the parser.
    explicit Parser(const std::string &encoded):
        errors(std::make_shared<std::vector<json>>()) {
        parse_with_err_handling([&encoded] {
            if (encoded.empty()) return json::object();
            return json::parse(encoded);
        });
    }

    /// @brief constructs a parser from an input stream (e.g., file stream).
    /// If the stream content is not valid JSON, immediately binds an error
    /// to the parser.
    explicit Parser(std::istream &stream):
        errors(std::make_shared<std::vector<json>>()) {
        parse_with_err_handling([&stream] { return json::parse(stream); });
    }

    /// @brief default constructor constructs a parser that will fail fast.
    Parser(): noop(true), errors(std::make_shared<std::vector<json>>()) {}

    /// @brief constructs a valid, empty parser {
    explicit Parser(const bool noop):
        noop(noop), errors(std::make_shared<std::vector<json>>()) {}

    /// @brief parses the parser's current value directly (when no path is specified).
    /// This is used to parse root values or when you have a child parser and want
    /// to parse its entire value.
    /// @returns The parsed value of type T, or default-constructed T if parsing fails.
    template<typename T>
    T field() {
        if (noop) return T();
        return parse_value<T>("", config);
    }

    /// @brief gets the field at the given path. Works for scalars, vectors, and maps.
    /// If the field is not found, accumulates an error in the builder.
    /// Special case: if path is empty string "", parses the root value (same as
    /// field<T>()).
    /// @param path The JSON path to the field.
    /// @returns The value at the given path, or a default-constructed T if not found.
    template<typename T>
    T field(const std::string &path) {
        if (noop) return T();
        if (path.empty()) return field<T>();
        const auto iter = config.find(path);
        if (iter == config.end()) {
            field_err(path, "This field is required");
            return T();
        }
        return get<T>(path, iter);
    }

    /// @brief attempts to pull the value at the provided path. If that path is not
    /// found, returns the default. Works for scalars, vectors, and maps.
    /// Note that this function will still accumulate an error if the path is found
    /// but the value is not of the expected type.
    /// @param path The JSON path to the value.
    /// @param default_value The default value to return if the path is not found.
    /// @returns The value at the path, or default_value if not found.
    template<typename T>
    T field(const std::string &path, const T &default_value) {
        if (noop) return default_value;

        const auto iter = config.find(path);
        if (iter == config.end()) return default_value;
        return get<T>(path, iter);
    }

    /// @brief gets a field by trying multiple paths in order until one is found.
    /// @param paths The paths to try in order.
    /// @returns The value at the first found path, or default-constructed T if none
    /// found.
    template<typename T>
    T field(const std::vector<std::string> &paths) {
        if (noop) return T();
        if (paths.empty()) {
            field_err("", "No paths provided");
            return T();
        }

        for (const auto &path: paths) {
            if (const auto iter = config.find(path); iter != config.end())
                return get<T>(path, iter);
        }

        field_err(paths[0], "this field is required");
        return T();
    }

    /// @brief gets a field by trying multiple paths, with a default fallback.
    /// @param paths The paths to try in order.
    /// @param default_value The default value if no paths are found.
    /// @returns The value at the first found path, or default_value if none found.
    template<typename T>
    T field(const std::vector<std::string> &paths, const T &default_value) {
        if (noop) return default_value;
        if (paths.empty()) return default_value;

        for (const auto &path: paths) {
            if (const auto iter = config.find(path); iter != config.end())
                return get<T>(path, iter);
        }

        return default_value;
    }

    /// @brief gets the field at the given path and creates a new parser just for
    /// that field. The field must be an object or an array. If the field is not of
    /// the expected type, or if the field is not found, accumulates an error in the
    /// parser.
    /// @param path The JSON path to the field.
    [[nodiscard]] Parser child(const std::string &path) const {
        if (noop) return {};
        const auto iter = config.find(path);
        if (iter == config.end()) {
            field_err(path, "this field is required");
            return {};
        }
        if (!iter->is_object() && !iter->is_array()) {
            field_err(path, "expected an object or array");
            return {};
        }
        return {*iter, errors, path_prefix + path + "."};
    }

    /// @brief gets the field at the given path and creates a new parser for that field,
    /// returning a noop parser if the field does not exist. Unlike child(), this method
    /// does not accumulate an error when the field is missing.
    /// @param path The JSON path to the field.
    /// @returns A parser for the child field, or a noop parser if not found.
    [[nodiscard]] Parser optional_child(const std::string &path) const {
        if (noop) return {};
        const auto iter = config.find(path);
        if (iter == config.end()) return {};
        if (!iter->is_object() && !iter->is_array()) {
            field_err(path, "expected an object or array");
            return {};
        }
        return {*iter, errors, path_prefix + path + "."};
    }

    /// @brief checks whether a field exists at the given path.
    /// @param path The JSON path to check.
    /// @returns true if the field exists, false otherwise (including if parser is
    /// noop).
    [[nodiscard]] bool has(const std::string &path) const {
        if (noop) return false;
        const auto iter = config.find(path);
        return iter != config.end();
    }

    /// @brief Iterates over an array at the given path, executing a function for
    /// each element. If the path does not point to an array, logs an error.
    /// @param path The JSON path to the array.
    /// @param func The function to execute for each element of the array. It should
    /// take a Parser as its argument.
    void
    iter(const std::string &path, const std::function<void(Parser &)> &func) const {
        if (noop) return;
        const auto it = config.find(path);
        if (it == config.end()) {
            field_err(path, "this field is required");
            return;
        }
        if (!it->is_array()) {
            field_err(path, "expected an array");
            return;
        }
        for (size_t i = 0; i < it->size(); ++i) {
            const auto child_path = path_prefix + path + "." + std::to_string(i) + ".";
            Parser child_parser((*it)[i], errors, child_path);
            func(child_parser);
        }
    }

    /// @brief Maps over an array at the given path, executing a function for each
    /// element and collecting the results into a vector. If the path does not point
    /// to an array, logs an error and returns an empty vector.
    /// @param path The JSON path to the array.
    /// @param func The function to execute for each element of the array. It should
    /// take a Parser as its argument and return a value of type T.
    /// @return A vector containing the results of applying func to each element.
    template<typename T>
    [[nodiscard]] std::vector<T>
    map(const std::string &path,
        const std::function<std::pair<T, bool>(Parser &)> &func) const {
        if (noop) return {};
        const auto iter = config.find(path);
        if (iter == config.end()) {
            field_err(path, "this field is required");
            return {};
        }
        if (!iter->is_array()) {
            field_err(path, "expected an array");
            return {};
        }
        std::vector<T> results;
        results.reserve(iter->size());
        for (size_t i = 0; i < iter->size(); ++i) {
            const auto child_path = path_prefix + path + "." + std::to_string(i) + ".";
            Parser child_parser((*iter)[i], errors, child_path);
            auto [res, ok] = func(child_parser);
            if (ok) results.push_back(std::move(res));
        }
        return results;
    }

    /// @brief binds a new error to the field at the given path, using the message from
    /// a xerrors::Error.
    /// @param path The JSON path to the field.
    /// @param err The error whose message will be used.
    void field_err(const std::string &path, const xerrors::Error &err) const {
        this->field_err(path, err.message());
    }

    /// @brief binds a new error to the field at the given path.
    /// @param path The JSON path to the field.
    /// @param message The error message to bind.
    void field_err(const std::string &path, const std::string &message) const {
        if (this->noop || this->errors == nullptr) return;
        this->errors->push_back({{"path", path_prefix + path}, {"message", message}});
    }

    /// @returns true if the parser has accumulated no errors, false otherwise.
    [[nodiscard]] bool ok() const {
        if (this->noop) return false;
        return this->errors == nullptr || this->errors->empty();
    }

    /// @returns the parser's errors as a JSON object of the form {"errors":
    /// [ACCUMULATED_ERRORS]}.
    [[nodiscard]] json error_json() const {
        json err;
        err["errors"] = *errors;
        return err;
    }

    /// @brief converts the parser's accumulated errors into a xerrors::Error.
    /// @returns xerrors::NIL if no errors, a simple validation error if there's a
    /// single error with an empty path, or a validation error containing all errors as
    /// JSON.
    [[nodiscard]] xerrors::Error error() const {
        if (this->errors->empty()) return xerrors::NIL;
        if (this->errors->size() == 1) {
            const auto &err = this->errors->at(0);
            if (err["path"].get<std::string>().empty())
                return xerrors::Error{
                    xerrors::VALIDATION,
                    err["message"].get<std::string>()
                };
        }
        return xerrors::Error{xerrors::VALIDATION, error_json().dump()};
    }

    /// @returns the underlying JSON configuration being parsed.
    [[nodiscard]] json get_json() const { return config; }

    /// @brief creates a parser from a file at the given path
    /// @param path The path to the JSON configuration file
    /// @return A parser for the configuration file
    static Parser from_file_path(const std::string &path) {
        std::ifstream file(path);
        if (!file.is_open()) {
            Parser p(false);
            p.field_err("", "failed to open file: " + path);
            return p;
        }
        return Parser(file);
    }
};

/// @brief Type trait to detect if a type can be constructed from a Parser
template<typename T>
inline constexpr bool is_parser_constructible_v =  // NOLINT(readability-identifier-naming) - follows STL naming convention
    std::is_constructible_v<T, Parser> ||
    std::is_constructible_v<T, Parser &> ||
    std::is_constructible_v<T, const Parser &> ||
    std::is_constructible_v<T, Parser &&>;

// Implementation of parse_value - the single source of truth for all type conversions
template<typename T>
T Parser::parse_value(const std::string &path, const json &j) {
    if constexpr (is_map_v<T>) {
        using K = typename is_map<T>::key_type;
        using V = typename is_map<T>::value_type;
        if (!j.is_object()) {
            field_err(path, "expected an object");
            return T();
        }
        T map_result;
        for (const auto &[json_key, value]: j.items()) {
            const auto child_path = xpath::join(".", {path, json_key});
            auto [map_key, ok] = convert_key<K>(json_key, child_path);
            if (!ok) continue;
            map_result[map_key] = parse_value<V>(child_path, value);
        }
        return map_result;
    } else if constexpr (is_vector_v<T>) {
        using U = typename is_vector<T>::value_type;
        if (!j.is_array()) {
            field_err(path, "expected an array");
            return T();
        }
        std::vector<U> values;
        values.reserve(j.size());
        for (size_t i = 0; i < j.size(); ++i) {
            const auto child_path = xpath::join(".", {path, std::to_string(i)});
            values.push_back(parse_value<U>(child_path, j[i]));
        }
        return values;
    } else if constexpr (is_parser_constructible_v<T>) {
        if (!j.is_object() && !j.is_array()) {
            field_err(path, "expected an object or array");
            return T();
        }
        const auto child_prefix = path.empty() ? path_prefix : path_prefix + path + ".";
        Parser child_parser(j, errors, child_prefix);
        return T(child_parser);
    } else {
        try {
            if constexpr (std::is_arithmetic_v<T>) {
                if (j.is_string()) {
                    T value;
                    std::istringstream iss(j.get<std::string>());
                    if (!(iss >> value)) {
                        field_err(
                            path,
                            "expected a number, got '" + j.get<std::string>() + "'"
                        );
                        return T();
                    }
                    return value;
                }
            }
            return j.get<T>();
        } catch (const nlohmann::json::type_error &e) {
            field_err(path, std::string(e.what()).substr(32));
            return T();
        }
    }
}

}
