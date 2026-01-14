// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "client/cpp/errors/errors.h"
#include "client/cpp/ranger/ranger.h"

namespace synnax::ranger {

std::pair<Range, x::errors::Error>
Client::retrieve_by_key(const std::string &key) const {
    auto req = grpc::ranger::RetrieveRequest();
    req.add_keys(key);
    auto [res, err] = retrieve_client->send("/range/retrieve", req);
    if (err) return {Range(), err};
    if (res.ranges_size() == 0)
        return {Range(), not_found_error("range", "key " + key)};
    auto [rng, proto_err] = Range::from_proto(res.ranges(0));
    if (proto_err) return {Range(), proto_err};
    rng.kv = this->kv.scope_to_range(rng.key);
    return {rng, x::errors::NIL};
}

std::pair<Range, x::errors::Error>
Client::retrieve_by_name(const std::string &name) const {
    auto req = grpc::ranger::RetrieveRequest();
    req.add_names(name);
    auto [res, err] = retrieve_client->send("/range/retrieve", req);
    if (err) return {Range(), err};
    if (res.ranges_size() == 0)
        return {Range(), not_found_error("range", "name " + name)};
    if (res.ranges_size() > 1)
        return {Range(), multiple_found_error("ranges", "name " + name)};
    auto [rng, proto_err] = Range::from_proto(res.ranges(0));
    if (proto_err) return {Range(), proto_err};
    rng.kv = this->kv.scope_to_range(rng.key);
    return {rng, x::errors::NIL};
}

std::pair<std::vector<Range>, x::errors::Error>
Client::retrieve_many(grpc::ranger::RetrieveRequest &req) const {
    auto [res, err] = retrieve_client->send("/range/retrieve", req);
    if (err) return {std::vector<Range>(), err};
    std::vector<Range> ranges;
    ranges.reserve(res.ranges_size());
    for (int i = 0; i < res.ranges_size(); i++) {
        auto [rng, proto_err] = Range::from_proto(res.ranges(i));
        if (proto_err) return {std::vector<Range>(), proto_err};
        rng.kv = this->kv.scope_to_range(rng.key);
        ranges.push_back(rng);
    }
    return {ranges, x::errors::NIL};
}

std::pair<std::vector<Range>, x::errors::Error>
Client::retrieve_by_name(const std::vector<std::string> &names) const {
    auto req = grpc::ranger::RetrieveRequest();
    for (auto &name: names)
        req.add_names(name);
    return retrieve_many(req);
}

std::pair<std::vector<Range>, x::errors::Error>
Client::retrieve_by_key(const std::vector<std::string> &keys) const {
    auto req = grpc::ranger::RetrieveRequest();
    for (auto &key: keys)
        req.add_keys(key);
    return retrieve_many(req);
}

x::errors::Error Client::create(std::vector<Range> &ranges) const {
    auto req = grpc::ranger::CreateRequest();
    req.mutable_ranges()->Reserve(ranges.size());
    for (const auto &range: ranges) {
        auto proto_range = range.to_proto();
        *req.add_ranges() = proto_range;
    }
    auto [res, err] = create_client->send("/range/create", req);
    if (err) return err;
    for (auto i = 0; i < res.ranges_size(); i++) {
        auto [rng, proto_err] = Range::from_proto(res.ranges(i));
        if (proto_err) return proto_err;
        ranges[i].key = rng.key;
        ranges[i].kv = this->kv.scope_to_range(ranges[i].key);
    }
    return x::errors::NIL;
}

x::errors::Error Client::create(Range &range) const {
    auto req = grpc::ranger::CreateRequest();
    auto proto_range = range.to_proto();
    *req.add_ranges() = proto_range;
    auto [res, err] = create_client->send("/range/create", req);
    if (err) return err;
    if (res.ranges_size() == 0) return unexpected_missing_error("range");
    auto [rng, proto_err] = Range::from_proto(res.ranges(0));
    if (proto_err) return proto_err;
    range.key = rng.key;
    range.kv = this->kv.scope_to_range(range.key);
    return x::errors::NIL;
}

std::pair<Range, x::errors::Error>
Client::create(const std::string &name, x::telem::TimeRange time_range) const {
    Range rng;
    rng.name = name;
    rng.time_range = time_range;
    auto err = create(rng);
    return {rng, err};
}

}
