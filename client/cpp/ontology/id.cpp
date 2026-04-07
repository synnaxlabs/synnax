// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "client/cpp/ontology/id.h"
#include "client/cpp/ontology/proto.gen.h"

namespace synnax::ontology {
std::string ID::string() const {
    return this->type + ":" + this->key;
}

std::pair<ID, x::errors::Error> ID::parse(const std::string &s) {
    if (s.empty())
        return {
            ID{},
            x::errors::Error(
                x::errors::VALIDATION,
                "[ontology] - cannot parse empty id"
            )
        };
    const auto colon_pos = s.find(':');
    if (colon_pos == std::string::npos)
        return {
            ID{},
            x::errors::Error(
                x::errors::VALIDATION,
                "[ontology] - failed to parse id '" + s +
                    "': expected format 'type:key'"
            )
        };
    if (colon_pos == 0)
        return {
            ID{},
            x::errors::Error(
                x::errors::VALIDATION,
                "[ontology] - failed to parse id '" + s + "': type is empty"
            )
        };
    return {
        ID{.type = s.substr(0, colon_pos), .key = s.substr(colon_pos + 1)},
        x::errors::NIL
    };
}

ID ID::parse(x::json::Parser parser) {
    return ID{
        .type = parser.field<std::string>("type"),
        .key = parser.field<std::string>("key"),
    };
}

x::json::json ID::to_json() const {
    x::json::json j;
    j["type"] = this->type;
    j["key"] = this->key;
    return j;
}

std::pair<ID, x::errors::Error>
ID::from_proto(const ::distribution::ontology::pb::ID &pb) {
    auto [type_str, err] = resource_type_from_pb(pb.type());
    if (!err.ok()) return {ID{}, err};
    return {ID{.type = type_str, .key = pb.key()}, x::errors::NIL};
}

std::pair<::distribution::ontology::pb::ID, x::errors::Error> ID::to_proto() const {
    auto [rt, err] = resource_type_to_pb(this->type);
    if (!err.ok()) return {{}, err};
    ::distribution::ontology::pb::ID pb;
    pb.set_type(rt);
    pb.set_key(this->key);
    return {pb, x::errors::NIL};
}

bool ID::operator==(const ID &other) const {
    return this->type == other.type && this->key == other.key;
}

bool ID::operator!=(const ID &other) const {
    return !(*this == other);
}

std::pair<std::vector<ID>, x::errors::Error>
parse_ids(const std::vector<std::string> &strs) {
    std::vector<ID> ids;
    ids.reserve(strs.size());
    for (const auto &s: strs) {
        auto [id, err] = ID::parse(s);
        if (!err.ok()) return {std::vector<ID>{}, err};
        ids.push_back(std::move(id));
    }
    return {ids, x::errors::NIL};
}

std::vector<std::string> ids_to_strings(const std::vector<ID> &ids) {
    std::vector<std::string> strs;
    strs.reserve(ids.size());
    for (const auto &id: ids)
        strs.push_back(id.string());
    return strs;
}
}
