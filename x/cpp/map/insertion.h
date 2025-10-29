// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#pragma once

#include <algorithm>
#include <string>
#include <vector>

#include "x/cpp/xjson/xjson.h"

namespace map {
/// @brief An insertion-ordered map that maintains the order of key-value pairs
/// based on their insertion order. Provides O(n) lookup but preserves insertion
/// order for iteration. Should only be used for small collections.
template<typename Value>
class Insertion {
private:
    /// @brief The keys in the map. Must have the same size as the values vector.
    std::vector<std::string> keys;
    /// @brief The values in the map. Must have the same size as the keys vector.
    std::vector<Value> values;

public:
    Insertion() = default;

    Insertion(xjson::Parser &p) {
        auto j = p.get_json();
        if (!j.is_object()) {
            p.field_err("", "expected an object");
            return;
        }
        for (auto it = j.begin(); it != j.end(); ++it) {
            const std::string &key = it.key();
            if constexpr (std::is_constructible_v<Value, xjson::Parser &>)
                values.emplace_back(p.child(key));
            else
                values.push_back(p.required<Value>(key));
            keys.push_back(key);
        }
    }

    /// @brief Returns the number of elements in the map.
    size_t count() const { return keys.size(); }

    /// @brief Returns true if the map is empty.
    bool empty() const { return keys.empty(); }

    /// @brief Checks if the map contains a key.
    /// @param name The key to search for.
    /// @return True if the key exists, false otherwise.
    bool contains(const std::string &name) const {
        return std::find(keys.begin(), keys.end(), name) != keys.end();
    }

    /// @brief Gets a pointer to the value associated with a key.
    /// @param name The key to search for.
    /// @return Pointer to the value if found, nullptr otherwise.
    const Value *get(const std::string &name) const {
        auto it = std::find(keys.begin(), keys.end(), name);
        if (it == keys.end()) return nullptr;
        size_t index = std::distance(keys.begin(), it);
        return &values[index];
    }

    /// @brief Gets a mutable pointer to the value associated with a key.
    /// @param name The key to search for.
    /// @return Pointer to the value if found, nullptr otherwise.
    Value *get(const std::string &name) {
        auto it = std::find(keys.begin(), keys.end(), name);
        if (it == keys.end()) return nullptr;
        size_t index = std::distance(keys.begin(), it);
        return &values[index];
    }

    /// @brief Inserts or updates a key-value pair.
    /// @param name The key to insert or update.
    /// @param value The value to associate with the key.
    void set(const std::string &name, const Value &value) {
        auto it = std::find(keys.begin(), keys.end(), name);
        if (it != keys.end()) {
            size_t index = std::distance(keys.begin(), it);
            values[index] = value;
        } else {
            keys.push_back(name);
            values.push_back(value);
        }
    }

    /// @brief Inserts or updates a key-value pair (move semantics).
    /// @param name The key to insert or update.
    /// @param value The value to move into the map.
    void set(const std::string &name, Value &&value) {
        auto it = std::find(keys.begin(), keys.end(), name);
        if (it != keys.end()) {
            size_t index = std::distance(keys.begin(), it);
            values[index] = std::move(value);
        } else {
            keys.push_back(name);
            values.push_back(std::move(value));
        }
    }

    /// @brief Removes a key-value pair from the map.
    /// @param name The key to remove.
    /// @return True if the key was found and removed, false otherwise.
    bool erase(const std::string &name) {
        auto it = std::find(keys.begin(), keys.end(), name);
        if (it == keys.end()) return false;
        size_t index = std::distance(keys.begin(), it);
        keys.erase(keys.begin() + index);
        values.erase(values.begin() + index);
        return true;
    }

    /// @brief Accesses the value at a specific index.
    /// @param index The index to access.
    /// @return Reference to the value at the index.
    Value &at(size_t index) { return values.at(index); }

    /// @brief Accesses the value at a specific index (const).
    /// @param index The index to access.
    /// @return Const reference to the value at the index.
    const Value &at(size_t index) const { return values.at(index); }

    /// @brief Gets the key at a specific index.
    /// @param index The index to access.
    /// @return Reference to the key at the index.
    const std::string &key_at(size_t index) const { return keys.at(index); }

    /// @brief Clears all key-value pairs from the map.
    void clear() {
        keys.clear();
        values.clear();
    }

    /// @brief Reserves space for a specified number of elements.
    /// @param capacity The number of elements to reserve space for.
    void reserve(size_t capacity) {
        keys.reserve(capacity);
        values.reserve(capacity);
    }
};
}
