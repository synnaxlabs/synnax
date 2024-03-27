//
// Created by Emiliano Bonilla on 3/27/24.
//

#include "nlohmann/json.hpp"

using json = nlohmann::json;

namespace jsonutil {
void field_err(const std::string& path, const std::string& message, json& err) {
    json field;
    field["path"] = path;
    field["message"] = message;
    err["errors"].push_back(field);
}


template <typename T>
T find_required(json j, const std::string& key, json& err, bool& ok) {
    auto iter = j.find(key);
    if (j.find(key) == j.end()) {
        field_err(key, "required", err);
        ok = false;
        return T();
    }
    return iter.value().get<T>();
}

// create a find_optional function that takes a json object, a key, and a default value
// if the key is not found in the json object, return the default value, otherwise,
// return the value associated with the key. Use a template to allow for any type of
// default value to be passed in.
template <typename T>
T find_optional(json j, const std::string& key, T default_value) {
    auto iter = j.find(key);
    if (iter == j.end()) {
        return default_value;
    }
    return iter.value().get<T>();
}
}