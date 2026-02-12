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

namespace synnax::ranger {

namespace synnax::ranger {
std::pair<Range, x::errors::Error> Range::from_proto(const api::v1::Range &rng) {
    auto [key, err] = x::uuid::UUID::parse(rng.key());
    if (err) return {{}, err};
    return {
        Range{
            .key = key,
            .name = rng.name(),
            .time_range = {rng.time_range().start(), rng.time_range().end()},
        },
        x::errors::NIL,
    };
}

void Range::to_proto(api::v1::Range *rng) const {
    rng->set_name(name);
    rng->set_key(this->key.to_string());
    auto tr = ::telem::PBTimeRange{};
    rng->mutable_time_range()->set_start(time_range.start.nanoseconds());
    rng->mutable_time_range()->set_end(time_range.end.nanoseconds());
}

std::pair<Range, x::errors::Error> Client::retrieve_by_key(const Key &key) const {
    auto req = api::v1::RangeRetrieveRequest();
    req.add_keys(key.to_string());
    auto [res, err] = retrieve_client->send("/range/retrieve", req);
    if (err) return {Range{}, err};
    if (res.ranges_size() == 0)
        return {Range{}, errors::not_found_error("range", "key " + key.to_string())};
    auto [rng, parse_err] = Range::from_proto(res.ranges(0));
    if (parse_err) return {Range{}, parse_err};
    rng.kv = this->kv.scope_to_range(rng.key.to_string());
    return {rng, x::errors::NIL};
}

std::pair<Range, x::errors::Error>
Client::retrieve_by_name(const std::string &name) const {
    auto req = api::v1::RangeRetrieveRequest();
    req.add_names(name);
    auto [res, err] = retrieve_client->send("/range/retrieve", req);
    if (err) return {Range{}, err};
    if (res.ranges_size() == 0)
        return {Range{}, errors::not_found_error("range", "name " + name)};
    if (res.ranges_size() > 1)
        return {Range{}, errors::multiple_found_error("ranges", "name " + name)};
    auto [rng, parse_err] = Range::from_proto(res.ranges(0));
    if (parse_err) return {Range{}, parse_err};
    rng.kv = this->kv.scope_to_range(rng.key.to_string());
    return {rng, x::errors::NIL};
}

std::pair<std::vector<Range>, x::errors::Error>
Client::retrieve_many(api::v1::RangeRetrieveRequest &req) const {
    auto [res, err] = retrieve_client->send("/range/retrieve", req);
    if (err) return {{}, err};
    std::vector<Range> ranges;
    ranges.reserve(res.ranges_size());
    for (const auto &pb: res.ranges()) {
        auto [rng, parse_err] = Range::from_proto(pb);
        if (parse_err) return {{}, parse_err};
        rng.kv = this->kv.scope_to_range(rng.key.to_string());
        ranges.push_back(std::move(rng));
    }
    return {ranges, x::errors::NIL};
}

std::pair<std::vector<Range>, x::errors::Error>
Client::retrieve_by_name(const std::vector<std::string> &names) const {
    auto req = api::v1::RangeRetrieveRequest();
    for (auto &name: names)
        req.add_names(name);
    return retrieve_many(req);
}

std::pair<std::vector<Range>, x::errors::Error>
Client::retrieve_by_key(const std::vector<Key> &keys) const {
    auto req = api::v1::RangeRetrieveRequest();
    for (const auto &key: keys)
        req.add_keys(key.to_string());
    return retrieve_many(req);
}

x::errors::Error Client::create(std::vector<Range> &ranges) const {
    auto req = api::v1::RangeCreateRequest();
    req.mutable_ranges()->Reserve(ranges.size());
    for (const auto &range: ranges) {
        auto proto_range = range.to_proto();
        *req.add_ranges() = proto_range;
    }
    auto [res, err] = create_client->send("/range/create", req);
    if (err) return err;
    for (auto i = 0; i < res.ranges_size(); i++) {
        auto [key, parse_err] = x::uuid::UUID::parse(res.ranges(i).key());
        if (parse_err) return parse_err;
        ranges[i].key = key;
        ranges[i].kv = this->kv.scope_to_range(ranges[i].key.to_string());
    }
    return x::errors::NIL;
}

x::errors::Error Client::create(Range &range) const {
    auto req = api::v1::RangeCreateRequest();
    range.to_proto(req.add_ranges());
    auto [res, err] = create_client->send("/range/create", req);
    if (err) return err;
    if (res.ranges_size() == 0) return errors::unexpected_missing_error("range");
    const auto &rng = res.ranges(0);
    auto [key, parse_err] = x::uuid::UUID::parse(rng.key());
    if (parse_err) return parse_err;
    range.key = key;
    range.kv = this->kv.scope_to_range(range.key.to_string());
    return x::errors::NIL;
}

std::pair<Range, x::errors::Error>
Client::create(const std::string &name, x::telem::TimeRange time_range) const {
    auto rng = Range{.name = name, .time_range = time_range};
    auto err = create(rng);
    return {rng, err};
}
}
