// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "client/cpp/errors/errors.h"
#include "client/cpp/ranger/ranger.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/xerrors/errors.h"

#include "core/pkg/api/grpc/v1/ranger.pb.h"
#include "x/go/telem/telem.pb.h"

namespace synnax {
Range::Range(std::string name, telem::TimeRange time_range):
    name(std::move(name)), time_range(time_range) {}

Range::Range(const api::v1::Range &rng):
    key(rng.key()),
    name(rng.name()),
    time_range(telem::TimeRange(rng.time_range().start(), rng.time_range().end())) {}

void Range::to_proto(api::v1::Range *rng) const {
    rng->set_name(name);
    rng->set_key(key);
    auto tr = telem::PBTimeRange();
    rng->mutable_time_range()->set_start(time_range.start.nanoseconds());
    rng->mutable_time_range()->set_end(time_range.end.nanoseconds());
}

const std::string RETRIEVE_ENDPOINT = "/range/retrieve";
const std::string CREATE_ENDPOINT = "/range/create";

std::pair<Range, xerrors::Error>
RangeClient::retrieve_by_key(const std::string &key) const {
    auto req = api::v1::RangeRetrieveRequest();
    req.add_keys(key);
    auto [res, err] = retrieve_client->send(RETRIEVE_ENDPOINT, req);
    if (err) return {Range(), err};
    if (res.ranges_size() == 0)
        return {
            Range(),
            xerrors::Error(xerrors::NOT_FOUND, "no ranges found matching " + key)
        };
    auto rng = Range(res.ranges(0));
    rng.kv = RangeKV(rng.key, kv_get_client, kv_set_client, kv_delete_client);
    return {rng, err};
}

std::pair<Range, xerrors::Error>
RangeClient::retrieve_by_name(const std::string &name) const {
    auto req = api::v1::RangeRetrieveRequest();
    req.add_names(name);
    auto [res, err] = retrieve_client->send(RETRIEVE_ENDPOINT, req);
    if (err) return {Range(), err};
    if (res.ranges_size() == 0)
        return {
            Range(),
            xerrors::Error(xerrors::NOT_FOUND, "no ranges found matching " + name)
        };
    if (res.ranges_size() > 1)
        return {
            Range(),
            xerrors::Error(
                xerrors::MULTIPLE_RESULTS,
                "multiple ranges found matching " + name
            )
        };
    auto rng = Range(res.ranges(0));
    rng.kv = RangeKV(rng.key, kv_get_client, kv_set_client, kv_delete_client);
    return {rng, err};
}

std::pair<std::vector<Range>, xerrors::Error>
RangeClient::retrieve_many(api::v1::RangeRetrieveRequest &req) const {
    auto [res, err] = retrieve_client->send(RETRIEVE_ENDPOINT, req);
    if (err) return {std::vector<Range>(), err};
    std::vector<Range> ranges = {res.ranges().begin(), res.ranges().end()};
    for (auto &r: ranges)
        r.kv = RangeKV(r.key, kv_get_client, kv_set_client, kv_delete_client);
    return {ranges, err};
}

std::pair<std::vector<Range>, xerrors::Error>
RangeClient::retrieve_by_name(const std::vector<std::string> &names) const {
    auto req = api::v1::RangeRetrieveRequest();
    for (auto &name: names)
        req.add_names(name);
    return retrieve_many(req);
}

std::pair<std::vector<Range>, xerrors::Error>
RangeClient::retrieve_by_key(const std::vector<std::string> &keys) const {
    auto req = api::v1::RangeRetrieveRequest();
    for (auto &key: keys)
        req.add_keys(key);
    return retrieve_many(req);
}

xerrors::Error RangeClient::create(std::vector<Range> &ranges) const {
    auto req = api::v1::RangeCreateRequest();
    req.mutable_ranges()->Reserve(ranges.size());
    for (const auto &range: ranges)
        range.to_proto(req.add_ranges());
    auto [res, err] = create_client->send(CREATE_ENDPOINT, req);
    if (err) return err;
    for (auto i = 0; i < res.ranges_size(); i++) {
        ranges[i].key = res.ranges(i).key();
        ranges[i].kv = RangeKV(
            ranges[i].key,
            kv_get_client,
            kv_set_client,
            kv_delete_client
        );
    }
    return xerrors::NIL;
}

xerrors::Error RangeClient::create(Range &range) const {
    auto req = api::v1::RangeCreateRequest();
    range.to_proto(req.add_ranges());
    auto [res, err] = create_client->send(CREATE_ENDPOINT, req);
    if (err) return err;
    if (res.ranges_size() == 0) return unexpected_missing("range");
    const auto rng = res.ranges(0);
    range.key = rng.key();
    range.kv = RangeKV(rng.key(), kv_get_client, kv_set_client, kv_delete_client);
    return err;
}

std::pair<Range, xerrors::Error>
RangeClient::create(const std::string &name, telem::TimeRange time_range) const {
    auto rng = Range(name, time_range);
    auto err = create(rng);
    return {rng, err};
}

const std::string KV_SET_ENDPOINT = "/range/kv/set";
const std::string KV_GET_ENDPOINT = "/range/kv/get";
const std::string KV_DELETE_ENDPOINT = "/range/kv/delete";

std::pair<std::string, xerrors::Error> RangeKV::get(const std::string &key) const {
    auto req = api::v1::RangeKVGetRequest();
    req.add_keys(key);
    req.set_range_key(range_key);
    auto [res, err] = kv_get_client->send(KV_GET_ENDPOINT, req);
    if (err) return {"", err};
    if (res.pairs_size() == 0)
        return {"", xerrors::Error(xerrors::NOT_FOUND, "key not found")};
    return {res.pairs().at(0).value(), err};
}

xerrors::Error RangeKV::set(const std::string &key, const std::string &value) const {
    auto req = api::v1::RangeKVSetRequest();
    req.set_range_key(range_key);
    const auto pair = req.add_pairs();
    pair->set_key(key);
    pair->set_value(value);
    auto [res, err] = kv_set_client->send(KV_SET_ENDPOINT, req);
    return err;
}

xerrors::Error RangeKV::del(const std::string &key) const {
    auto req = api::v1::RangeKVDeleteRequest();
    req.set_range_key(range_key);
    req.add_keys(key);
    auto [res, err] = kv_delete_client->send(KV_DELETE_ENDPOINT, req);
    return err;
}
}
