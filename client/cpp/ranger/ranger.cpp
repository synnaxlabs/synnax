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
#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"

#include "core/pkg/api/grpc/kv/kv.pb.h"
#include "core/pkg/api/grpc/ranger/ranger.pb.h"
#include "core/pkg/api/ranger/pb/range.pb.h"
#include "x/go/telem/telem.pb.h"

namespace synnax::range {
Range::Range(std::string name, x::telem::TimeRange time_range):
    name(std::move(name)), time_range(time_range) {}

Range::Range(const api::range::pb::Range &rng):
    key(rng.key()),
    name(rng.name()),
    time_range(x::telem::TimeRange(rng.time_range().start(), rng.time_range().end())) {}

void Range::to_proto(api::range::pb::Range *rng) const {
    rng->set_name(name);
    rng->set_key(key);
    auto tr = ::telem::PBTimeRange();
    rng->mutable_time_range()->set_start(time_range.start.nanoseconds());
    rng->mutable_time_range()->set_end(time_range.end.nanoseconds());
}

std::pair<Range, x::errors::Error>
Client::retrieve_by_key(const std::string &key) const {
    auto req = grpc::ranger::RetrieveRequest();
    req.add_keys(key);
    auto [res, err] = retrieve_client->send("/range/retrieve", req);
    if (err) return {Range(), err};
    if (res.ranges_size() == 0)
        return {Range(), not_found_error("range", "key " + key)};
    auto rng = Range(res.ranges(0));
    rng.kv = this->kv.scope_to_range(rng.key);
    return {rng, err};
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
    auto rng = Range(res.ranges(0));
    rng.kv = this->kv.scope_to_range(rng.key);
    return {rng, err};
}

std::pair<std::vector<Range>, x::errors::Error>
Client::retrieve_many(grpc::ranger::RetrieveRequest &req) const {
    auto [res, err] = retrieve_client->send("/range/retrieve", req);
    if (err) return {std::vector<Range>(), err};
    std::vector<Range> ranges = {res.ranges().begin(), res.ranges().end()};
    for (auto &r: ranges)
        r.kv = this->kv.scope_to_range(r.key);
    return {ranges, err};
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
    for (const auto &range: ranges)
        range.to_proto(req.add_ranges());
    auto [res, err] = create_client->send("/range/create", req);
    if (err) return err;
    for (auto i = 0; i < res.ranges_size(); i++) {
        ranges[i].key = res.ranges(i).key();
        ranges[i].kv = this->kv.scope_to_range(ranges[i].key);
    }
    return x::errors::NIL;
}

x::errors::Error Client::create(Range &range) const {
    auto req = grpc::ranger::CreateRequest();
    range.to_proto(req.add_ranges());
    auto [res, err] = create_client->send("/range/create", req);
    if (err) return err;
    if (res.ranges_size() == 0) return unexpected_missing_error("range");
    const auto rng = res.ranges(0);
    range.key = rng.key();
    range.kv = this->kv;
    return err;
}

std::pair<Range, x::errors::Error>
Client::create(const std::string &name, x::telem::TimeRange time_range) const {
    auto rng = Range(name, time_range);
    auto err = create(rng);
    return {rng, err};
}

}
