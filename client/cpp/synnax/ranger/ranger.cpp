
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
#include "synnax/telem/telem.h"

using namespace synnax;


Range::Range(const std::string &name, synnax::TimeRange time_range) :
        name(name),
        time_range(time_range) {
}


Range::Range(const api::v1::Range &a) :
        key(a.key()),
        name(a.name()),
        time_range(
                synnax::TimeRange(synnax::TimeStamp(a.time_range().start()), synnax::TimeStamp(a.time_range().end()))) {
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

std::pair<Range, freighter::Error> RangeClient::retrieveOne(api::v1::RangeRetrieveRequest &req) const {
    auto [res, err] = retrieve_client->send(RETRIEVE_ENDPOINT, req);
    if (err) return {Range(), err};
    return {Range(res.ranges(0)), err};
}

std::pair<Range, freighter::Error> RangeClient::retrieveByKey(const std::string &key) const {
    auto req = api::v1::RangeRetrieveRequest();
    req.add_keys(key);
    return retrieveOne(req);
}

std::pair<Range, freighter::Error> RangeClient::retrieveByName(const std::string &name) const {
    auto req = api::v1::RangeRetrieveRequest();
    req.add_names(name);
    return retrieveOne(req);
}

std::pair<Range, freighter::Error> RangeClient::activeRange() {
    return {Range(), freighter::NIL};
}


std::pair<std::vector<Range>, freighter::Error> RangeClient::retrieveMany(api::v1::RangeRetrieveRequest &req) const {
    auto [res, err] = retrieve_client->send(RETRIEVE_ENDPOINT, req);
    if (err) return {std::vector<Range>(), err};
    std::vector<Range> ranges = {res.ranges().begin(), res.ranges().end()};
    for (auto &r: ranges) r.kv = RangeKV(r.key, kv_get_client, kv_set_client, kv_delete_client);
    return {ranges, err};
}

std::pair<std::vector<Range>, freighter::Error> RangeClient::retrieveByName(std::vector<std::string> names) const {
    auto req = api::v1::RangeRetrieveRequest();
    for (auto &name: names) req.add_names(name);
    return retrieveMany(req);
}

std::pair<std::vector<Range>, freighter::Error> RangeClient::retrieveByKey(std::vector<std::string> keys) const {
    auto req = api::v1::RangeRetrieveRequest();
    for (auto &key: keys) req.add_keys(key);
    return retrieveMany(req);
}


freighter::Error RangeClient::create(std::vector<Range> &ranges) const {
    auto req = api::v1::RangeCreateRequest();
    req.mutable_ranges()->Reserve(ranges.size());
    for (const auto &range: ranges) range.to_proto(req.add_ranges());
    auto [res, err] = create_client->send(CREATE_ENDPOINT, req);
    if (!err)
        for (auto i = 0; i < res.ranges_size(); i++) {
            ranges[i].key = res.ranges(i).key();
            ranges[i].kv = RangeKV(ranges[i].key, kv_get_client, kv_set_client, kv_delete_client);
        }
    return err;
}


freighter::Error RangeClient::create(Range &range) const {
    auto req = api::v1::RangeCreateRequest();
    range.to_proto(req.add_ranges());
    auto [res, err] = create_client->send(CREATE_ENDPOINT, req);
    if (!err) range.key = res.ranges(0).key();
    return err;
}

std::pair<Range, freighter::Error> RangeClient::create(std::string name, synnax::TimeRange time_range) const {
    auto rng = Range(name, time_range);
    auto err = create(rng);
    return {rng, err};
}

const std::string KV_SET_ENDPOINT = "/range/kv/set";
const std::string KV_GET_ENDPOINT = "/range/kv/get";
const std::string KV_DELETE_ENDPOINT = "/range/kv/delete";


std::pair<std::string, freighter::Error> RangeKV::get(const std::string &key) const {
    auto req = api::v1::RangeKVGetRequest();
    req.add_keys(key);
    auto [res, err] = kv_get_client->send(KV_GET_ENDPOINT, req);
    return {res.pairs().at(key), err};
}

freighter::Error RangeKV::set(const std::string &key, const std::string &value) const {
    auto req = api::v1::RangeKVSetRequest();
    req.set_range_key(range_key);
    (*req.mutable_pairs())[key] = value;
    auto [res, err] = kv_set_client->send(KV_SET_ENDPOINT, req);
    return err;
}

freighter::Error RangeKV::del(const std::string &key) const {
    auto req = api::v1::RangeKVDeleteRequest();
    req.set_range_key(range_key);
    req.add_keys(key);
    auto [res, err] = kv_delete_client->send(KV_DELETE_ENDPOINT, req);
    return err;
}