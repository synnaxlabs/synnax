#pragma once

/// std.
#include <string>

/// external.
#include "nlohmann/json.hpp"

using json = nlohmann::json;

namespace config {
class Parser {
public:
    std::shared_ptr<std::vector<json>> errors;

    explicit Parser(json config): errors(std::make_shared<std::vector<json>>()),
                                  config(std::move(config)) {
    }

    explicit Parser(const std::string& encoded): errors(
        std::make_shared<std::vector<json>>()) {
        try {
            config = json::parse(encoded);
        } catch (const json::parse_error& e) {
            json field;
            field["path"] = "";
            field["message"] = e.what();
            errors->push_back(field);
        }
    }

    Parser(
        json config,
        std::shared_ptr<std::vector<json>> errors,
        std::string path_prefix
    )
        : errors(std::move(errors)),
          path_prefix(std::move(path_prefix)),
          config(std::move(config)) {
    }

    explicit Parser(bool noop) : noop(noop) {
    }

    Parser() = default;


    /// @brief gets the field at the given path. If the field is not found,
    /// accumulates an error in the builder.
    template<typename T>
    T required(const std::string& path) {
        if (noop) return T();
        const auto iter = config.find(path);
        if (iter == config.end()) {
            field_err(path, "This field is required");
            return T();
        }
        return get<T>(path, iter);
    }

    template<typename T>
    T optional(const std::string& path, T default_value) {
        if (noop) return default_value;
        const auto iter = config.find(path);
        if (iter == config.end()) return default_value;
        return get<T>(path, iter);
    }

    Parser child(const std::string& path) {
        const auto iter = config.find(path);
        if (iter == config.end()) {
            field_err(path, "This field is required");
            return Parser(true);
        }
        return {*iter, errors, path_prefix + path + "."};
    }

    /// @brief Iterates over an array at the given path, executing a function for each element.
    /// If the path does not point to an array, logs an error.
    /// @param path The JSON path to the array.
    /// @param func The function to execute for each element of the array. It should take a Parser as its argument.
    void iter(const std::string& path, const std::function<void(Parser&)>& func) {
        if (noop) return;
        const auto iter = config.find(path);
        if (iter == config.end()) {
            field_err(path, "This field is required");
            return;
        }
        if (!iter->is_array()) {
            field_err(path, "Expected an array");
            return;
        }

        for (size_t i = 0; i < iter->size(); ++i) {
            // Construct a Parser for each element of the array, appending the index to the path
            Parser childParser((*iter)[i], errors,
                               path_prefix + path + "." + std::to_string(i) + ".");
            func(childParser); // Execute the function, providing the child Parser
        }
    }


    void field_err(const std::string& path, const std::string& message) const {
        json field;
        field["path"] = path_prefix + path;
        field["message"] = message;
        errors->push_back(field);
    }

    [[nodiscard]] bool ok() const { return errors->empty(); }

    [[nodiscard]] json error_json() const {
        json err;
        err["errors"] = *errors;
        return err;
    }

private:
    std::string path_prefix;
    bool noop = false;
    json config;

    template<typename T>
    T get(const std::string& path, const nlohmann::basic_json<>::iterator& iter) {
        try {
            return iter->get<T>();
        } catch (const nlohmann::json::type_error& e) {
            // slice the error message from index 32 to remove the library error prefix.
            field_err(path, e.what() + 32);
        }
        return T();
    }
};
}
