// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "client/cpp/ranger/ranger.h"

#include "client/cpp/errors/errors.h"
#include "client/cpp/telem/telem.h"
#include "synnax/pkg/api/grpc/v1/synnax/pkg/api/grpc/v1/ranger.pb.h"
#include "x/go/telem/x/go/telem/telem.pb.h"

using namespace synnax;

Range::Range(const std::string &name, synnax::TimeRange time_range) : name(name),
    time_range(time_range) {
}

Range::Range(
    const api::v1::Range &rng
) : key(rng.key()),
    name(rng.name()),
    time_range(synnax::TimeRange(rng.time_range().start(), rng.time_range().end())) {
}

void Range::to_proto(api::v1::Range *rng) const {
    rng->set_name(name);
    rng->set_key(key);
    auto tr = telem::PBTimeRange();
    rng->mutable_time_range()->set_start(time_range.start.value);
    rng->mutable_time_range()->set_end(time_range.end.value);
}

const std::string RETRIEVE_ENDPOINT = "/range/retrieve";
const std::string CREATE_ENDPOINT = "/range/create";

std::pair<Range, freighter::Error> RangeClient::retrieveByKey(
    const std::string &key) const {
    auto req = api::v1::RangeRetrieveRequest();
    req.add_keys(key);
    auto [res, err] = retrieve_client->send(RETRIEVE_ENDPOINT, req);
    if (err) return {Range(), err};
    if (res.ranges_size() == 0)
        return {
            Range(),
            freighter::Error(synnax::NOT_FOUND, "no ranges found matching " + key)
        };
    auto rng = Range(res.ranges(0));
    rng.kv = RangeKV(rng.key, kv_get_client, kv_set_client, kv_delete_client);
    return {rng, err};
}

std::pair<Range, freighter::Error> RangeClient::retrieveByName(
    const std::string &name) const {
    auto req = api::v1::RangeRetrieveRequest();
    req.add_names(name);
    auto [res, err] = retrieve_client->send(RETRIEVE_ENDPOINT, req);
    if (err) return {Range(), err};
    if (res.ranges_size() == 0)
        return {
            Range(),
            freighter::Error(synnax::NOT_FOUND, "no ranges found matching " + name)
        };
    if (res.ranges_size() > 1)
        return {
            Range(),
            freighter::Error(synnax::MULTIPLE_RESULTS,
                             "multiple ranges found matching " + name)
        };
    auto rng = Range(res.ranges(0));
    rng.kv = RangeKV(rng.key, kv_get_client, kv_set_client, kv_delete_client);
    return {rng, err};
}

std::pair<Range, freighter::Error> RangeClient::activeRange() {
    return {Range(), freighter::NIL};
}

std::pair<std::vector<Range>, freighter::Error> RangeClient::retrieveMany(
    api::v1::RangeRetrieveRequest &req) const {
    auto [res, err] = retrieve_client->send(RETRIEVE_ENDPOINT, req);
    if (err) return {std::vector<Range>(), err};
    std::vector<Range> ranges = {res.ranges().begin(), res.ranges().end()};
    for (auto &r: ranges)
        r.kv = RangeKV(r.key, kv_get_client, kv_set_client,
                       kv_delete_client);
    return {ranges, err};
}

std::pair<std::vector<Range>, freighter::Error> RangeClient::retrieveByName(
    std::vector<std::string> names) const {
    auto req = api::v1::RangeRetrieveRequest();
    for (auto &name: names) req.add_names(name);
    return retrieveMany(req);
}

std::pair<std::vector<Range>, freighter::Error> RangeClient::retrieveByKey(
    std::vector<std::string> keys) const {
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
            ranges[i].kv = RangeKV(ranges[i].key, kv_get_client, kv_set_client,
                                   kv_delete_client);
        }
    return err;
}

freighter::Error RangeClient::create(Range &range) const {
    auto req = api::v1::RangeCreateRequest();
    range.to_proto(req.add_ranges());
    auto [res, err] = create_client->send(CREATE_ENDPOINT, req);
    if (!err) {
        auto rng = res.ranges(0);
        range.key = rng.key();
        range.kv = RangeKV(rng.key(), kv_get_client, kv_set_client, kv_delete_client);
    }
    return err;
}

std::pair<Range, freighter::Error> RangeClient::create(
    std::string name, synnax::TimeRange time_range) const {
    auto rng = Range(name, time_range);
    auto err = create(rng);
    return {rng, err};
}

const std::string SET_ACTIVE_ENDPOINT = "/range/set-active";
const std::string RETRIEVE_ACTIVE_ENDPOINT = "/range/retrieve-active";
const std::string CLEAR_ACTIVE_ENDPOINT = "/range/clear-active";

freighter::Error RangeClient::setActive(const std::string &key) const {
    auto req = api::v1::RangeSetActiveRequest();
    req.set_range(key);
    auto res = set_active_client->send(SET_ACTIVE_ENDPOINT, req);
    return res.second;
}

std::pair<Range, freighter::Error> RangeClient::retrieveActive() const {
    auto req = google::protobuf::Empty();
    auto [res, err] = retrieve_active_client->send(RETRIEVE_ACTIVE_ENDPOINT, req);
    if (err) return {Range(), err};
    auto rng = Range(res.range());
    rng.kv = RangeKV(rng.key, kv_get_client, kv_set_client, kv_delete_client);
    return {rng, err};
}

freighter::Error RangeClient::clearActive() const {
    auto req = google::protobuf::Empty();
    auto res = clear_active_client->send(CLEAR_ACTIVE_ENDPOINT, req);
    return res.second;
}

const std::string KV_SET_ENDPOINT = "/range/kv/set";
const std::string KV_GET_ENDPOINT = "/range/kv/get";
const std::string KV_DELETE_ENDPOINT = "/range/kv/delete";


std::pair<std::string, freighter::Error> RangeKV::get(const std::string &key) const {
    auto req = api::v1::RangeKVGetRequest();
    req.add_keys(key);
    req.set_range_key(range_key);
    auto [res, err] = kv_get_client->send(KV_GET_ENDPOINT, req);
    if (err) return {"", err};
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
