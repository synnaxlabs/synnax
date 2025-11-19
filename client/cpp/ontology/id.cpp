// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <sstream>

#include "client/cpp/ontology/id.h"

namespace synnax::ontology {
ID::ID(std::string type, std::string key): type(std::move(type)), key(std::move(key)) {}

std::string ID::string() const {
    return type + ":" + key;
}

std::pair<ID, xerrors::Error> ID::parse(const std::string &s) {
    const auto colon_pos = s.find(':');
    if (colon_pos == std::string::npos) {
        return {
            ID{},
            xerrors::Error(
                xerrors::VALIDATION,
                "[ontology] - failed to parse id '" + s +
                    "': expected format 'type:key'"
            )
        };
    }

    const auto type = s.substr(0, colon_pos);
    const auto key = s.substr(colon_pos + 1);

    ID id{type, key};
    const auto err = id.validate();
    if (!err.ok()) return {ID{}, err};

    return {id, xerrors::Error{}};
}

xerrors::Error ID::validate() const {
    if (key.empty()) {
        return xerrors::Error(xerrors::VALIDATION, "[ontology] - key is required");
    }
    if (type.empty()) {
        return xerrors::Error(xerrors::VALIDATION, "[ontology] - type is required");
    }
    return xerrors::Error{};
}

bool ID::operator==(const ID &other) const {
    return type == other.type && key == other.key;
}

bool ID::operator!=(const ID &other) const {
    return !(*this == other);
}

void to_json(json &j, const ID &id) {
    j = json{{"type", id.type}, {"key", id.key}};
}

void from_json(const json &j, ID &id) {
    j.at("type").get_to(id.type);
    j.at("key").get_to(id.key);
}

std::pair<std::vector<ID>, xerrors::Error>
parse_ids(const std::vector<std::string> &strs) {
    std::vector<ID> ids;
    ids.reserve(strs.size());

    for (const auto &s: strs) {
        auto [id, err] = ID::parse(s);
        if (!err.ok()) return {std::vector<ID>{}, err};
        ids.push_back(std::move(id));
    }

    return {ids, xerrors::Error{}};
}

std::vector<std::string> ids_to_strings(const std::vector<ID> &ids) {
    std::vector<std::string> strs;
    strs.reserve(ids.size());
    for (const auto &id: ids)
        strs.push_back(id.string());
    return strs;
}
}
