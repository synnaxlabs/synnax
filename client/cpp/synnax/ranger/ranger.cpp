
// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// protos
#include "v1/ranger.pb.h"
#include "telempb/telem.pb.h"

/// internal
#include "synnax/ranger/ranger.h"
#include "synnax/exceptions.h"
#include "synnax/telem/telem.h"

using namespace Synnax;
using namespace Ranger;


Range::Range(const std::string &name, Telem::TimeRange time_range) :
        name(name),
        time_range(time_range) {
}


Range::Range(const api::v1::Range &a) :
        key(a.key()),
        name(a.name()),
        time_range(
                Telem::TimeRange(Telem::TimeStamp(a.time_range().start()), Telem::TimeStamp(a.time_range().end()))),
        kv(nullptr) {
}

void Range::to_proto(api::v1::Range *rng) const {
    rng->set_name(name);
    rng->set_key(key);
    auto tr = telempb::TimeRange();
    tr.set_start(time_range.start.value);
    tr.set_end(time_range.end.value);
    rng->set_allocated_time_range(&tr);
}


const std::string RETRIEVE_ENDPOINT = "/range/retrieve";
const std::string CREATE_ENDPOINT = "/range/create";

Range Client::retrieve_by_key(const std::string &key) const {
    auto req = api::v1::RangeRetrieveRequest();
    req.add_keys(key);
    auto [res, err] = retrieve_client->send(RETRIEVE_ENDPOINT, req);
    if (!err.ok())
        throw QueryError(err.error_message());
    if (res.ranges_size() == 0)
        throw QueryError("No range found with key " + key);
    return Range(res.ranges(0));
}

Range Client::retrieve_by_name(const std::string &name) const {
    auto req = api::v1::RangeRetrieveRequest();
    req.add_names(name);
    auto [res, err] = retrieve_client->send(RETRIEVE_ENDPOINT, req);
    if (!err.ok())
        throw QueryError(err.error_message());
    if (res.ranges_size() == 0)
        throw QueryError("No range found with name " + name);
    return Range(res.ranges(0));
}

std::vector<Range> Client::retrieve_by_key(std::vector<std::string> keys) const {
    auto req = api::v1::RangeRetrieveRequest();
    for (auto &key: keys)
        req.add_keys(key);
    auto [res, err] = retrieve_client->send(RETRIEVE_ENDPOINT, req);
    if (!err.ok())
        throw QueryError(err.error_message());
    std::vector<Range> ranges = {res.ranges().begin(), res.ranges().end()};
    for (auto &r: ranges) r.kv = nullptr;
    return ranges;
}

std::vector<Range> Client::retrieve_by_name(std::vector<std::string> names) const {
    auto req = api::v1::RangeRetrieveRequest();
    for (auto &name: names)
        req.add_names(name);
    auto [res, err] = retrieve_client->send(RETRIEVE_ENDPOINT, req);
    if (!err.ok())
        throw QueryError(err.error_message());
    std::vector<Range> ranges = {res.ranges().begin(), res.ranges().end()};
    for (auto &r: ranges) r.kv = nullptr;
    return ranges;
}

void Client::create(std::vector<Range> &ranges) const {
    auto req = api::v1::RangeCreateRequest();
    req.mutable_ranges()->Reserve(ranges.size());
    for (const auto &range: ranges) range.to_proto(req.add_ranges());
    auto [res, err] = create_client->send(CREATE_ENDPOINT, req);
    if (!err.ok())
        throw QueryError(err.error_message());
    for (auto i = 0; i < res.ranges_size(); i++) {
        ranges[i].key = res.ranges(i).key();
        ranges[i].kv = nullptr;
    }
}


void Client::create(Range &range) const {
    auto req = api::v1::RangeCreateRequest();
    range.to_proto(req.add_ranges());
    auto [res, err] = create_client->send(CREATE_ENDPOINT, req);
    if (!err.ok())
        throw QueryError(err.error_message());
    range.key = res.ranges(0).key();
}

Range Client::create(std::string name, Telem::TimeRange time_range) const {
    auto rng = Range(name, time_range);
    create(rng);
    return rng;
}

const std::string KV_SET_ENDPOINT = "/range/kv/set";
const std::string KV_GET_ENDPOINT = "/range/kv/get";
const std::string KV_DELETE_ENDPOINT = "/range/kv/delete";


std::string KV::get(const std::string &key) const {
    auto req = api::v1::RangeKVGetRequest();
    req.add_keys(key);
    auto [res, err] = kv_get_client->send(KV_GET_ENDPOINT, req);
    if (!err.ok())
        throw QueryError(err.error_message());
    return res.pairs().at(key);
}

void KV::set(const std::string &key, const std::string &value) const {
    auto req = api::v1::RangeKVSetRequest();
    req.set_range_key(range_key);
    (*req.mutable_pairs())[key] = value;
    auto [res, err] = kv_set_client->send(KV_SET_ENDPOINT, req);
    if (!err.ok())
        throw QueryError(err.error_message());
}

void KV::delete_(const std::string &key) const {
    auto req = api::v1::RangeKVDeleteRequest();
    req.set_range_key(range_key);
    req.add_keys(key);
    auto [res, err] = kv_delete_client->send(KV_DELETE_ENDPOINT, req);
    if (!err.ok())
        throw QueryError(err.error_message());
}