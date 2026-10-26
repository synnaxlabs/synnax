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

std::pair<std::vector<Range>, x::errors::Error>
Client::retrieve(const RetrieveRequest &req) const {
    auto [pb_req, pb_err] = req.to_proto();
    if (pb_err) return {std::vector<Range>(), pb_err};
    grpc::ranger::RetrieveRequest grpc_req;
    *grpc_req.mutable_request() = pb_req;
    auto [res, err] = retrieve_client->send("/range/retrieve", grpc_req);
    if (err) return {std::vector<Range>(), err};
    auto [decoded, proto_err] = RetrieveResponse::from_proto(res);
    if (proto_err) return {std::vector<Range>(), proto_err};
    for (auto &rng: decoded.ranges)
        rng.kv = this->kv.scope_to_range(rng.key);
    return {std::move(decoded.ranges), x::errors::NIL};
}

std::pair<Range, x::errors::Error> Client::retrieve_by_key(const Key &key) const {
    auto [ranges, err] = this->retrieve(
        {.where = Where{.key = x::filter::Set<Key>{.eq = key}}}
    );
    if (err) return {Range(), err};
    if (ranges.empty())
        return {Range(), errors::not_found_error("range", "key " + key.to_string())};
    return {ranges[0], x::errors::NIL};
}

std::pair<Range, x::errors::Error>
Client::retrieve_by_name(const std::string &name) const {
    auto [ranges, err] = this->retrieve(
        {.where = Where{.name = x::filter::String{.eq = name}}}
    );
    if (err) return {Range(), err};
    if (ranges.empty())
        return {Range(), errors::not_found_error("range", "name " + name)};
    if (ranges.size() > 1)
        return {Range(), errors::multiple_found_error("ranges", "name " + name)};
    return {ranges[0], x::errors::NIL};
}

std::pair<std::vector<Range>, x::errors::Error>
Client::retrieve_by_name(const std::vector<std::string> &names) const {
    return this->retrieve({.where = Where{.name = x::filter::String{.in = names}}});
}

std::pair<std::vector<Range>, x::errors::Error>
Client::retrieve_by_key(const std::vector<Key> &keys) const {
    return this->retrieve({.where = Where{.key = x::filter::Set<Key>{.in = keys}}});
}

x::errors::Error Client::create(std::vector<Range> &ranges) const {
    auto req = grpc::ranger::CreateRequest();
    req.mutable_ranges()->Reserve(ranges.size());
    for (const auto &range: ranges) {
        auto [proto_range, proto_err] = range.to_proto();
        if (proto_err) return proto_err;
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
    auto [proto_range, pb_err] = range.to_proto();
    if (pb_err) return pb_err;
    *req.add_ranges() = proto_range;
    auto [res, err] = create_client->send("/range/create", req);
    if (err) return err;
    if (res.ranges_size() == 0) return errors::unexpected_missing_error("range");
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

x::errors::Error
Client::set_end(const x::uuid::UUID &key, const x::telem::TimeStamp end) const {
    auto req = grpc::ranger::SetEndRequest();
    req.set_key(key.to_string());
    req.set_end(end.nanoseconds());
    return set_end_client->send("/range/set-end", req).second;
}

}
