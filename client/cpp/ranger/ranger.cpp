// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// module
#include "x/cpp/telem/telem.h"
#include "x/cpp/xerrors/errors.h"
#include "x/go/telem/x/go/telem/telem.pb.h"

/// internal
#include "client/cpp/errors/errors.h"
#include "client/cpp/ranger/ranger.h"

/// protos
#include "synnax/pkg/api/grpc/v1/synnax/pkg/api/grpc/v1/ranger.pb.h"

using namespace synnax;

Range::Range(std::string name, telem::TimeRange time_range)
    : name(std::move(name)),
      time_range(time_range) {
}

Range::Range(const api::v1::Range &rng)
    : key(rng.key()),
      name(rng.name()),
      time_range(telem::TimeRange(rng.time_range().start(), rng.time_range().end())) {
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

auto RangeClient::retrieve_by_key(const std::string &key) const
    -> std::pair<Range, xerrors::Error> {
    auto req = api::v1::RangeRetrieveRequest();
    req.add_keys(key);
    auto [res, err] = retrieve_client->send(RETRIEVE_ENDPOINT, req);
    if (err)
        return { Range(), err };
    if (res.ranges_size() == 0)
        return {
            Range(),
            xerrors::Error(xerrors::NOT_FOUND, "no ranges found matching " + key)
        };
    auto rng = Range(res.ranges(0));
    rng.kv = RangeKV(rng.key, kv_get_client, kv_set_client, kv_delete_client);
    return { rng, err };
}

auto RangeClient::retrieve_by_name(const std::string &name) const
    -> std::pair<Range, xerrors::Error> {
    auto req = api::v1::RangeRetrieveRequest();
    req.add_names(name);
    auto [res, err] = retrieve_client->send(RETRIEVE_ENDPOINT, req);
    if (err)
        return { Range(), err };
    if (res.ranges_size() == 0)
        return {
            Range(),
            xerrors::Error(xerrors::NOT_FOUND, "no ranges found matching " + name)
        };
    if (res.ranges_size() > 1)
        return { Range(),
                 xerrors::Error(
                     xerrors::MULTIPLE_RESULTS,
                     "multiple ranges found matching " + name
                 ) };
    auto rng = Range(res.ranges(0));
    rng.kv = RangeKV(rng.key, kv_get_client, kv_set_client, kv_delete_client);
    return { rng, err };
}

auto RangeClient::retrieve_many(api::v1::RangeRetrieveRequest &req) const
    -> std::pair<std::vector<Range>, xerrors::Error> {
    auto [res, err] = retrieve_client->send(RETRIEVE_ENDPOINT, req);
    if (err)
        return { std::vector<Range>(), err };
    std::vector<Range> ranges = { res.ranges().begin(), res.ranges().end() };
    for (auto &r : ranges)
        r.kv = RangeKV(r.key, kv_get_client, kv_set_client, kv_delete_client);
    return { ranges, err };
}

auto RangeClient::retrieve_by_name(const std::vector<std::string> &names) const
    -> std::pair<std::vector<Range>, xerrors::Error> {
    auto req = api::v1::RangeRetrieveRequest();
    for (const auto &name : names)
        req.add_names(name);
    return retrieve_many(req);
}

auto RangeClient::retrieve_by_key(const std::vector<std::string> &keys) const
    -> std::pair<std::vector<Range>, xerrors::Error> {
    auto req = api::v1::RangeRetrieveRequest();
    for (const auto &key : keys)
        req.add_keys(key);
    return retrieve_many(req);
}

auto RangeClient::create(std::vector<Range> &ranges) const -> xerrors::Error {
    auto req = api::v1::RangeCreateRequest();
    req.mutable_ranges()->Reserve(static_cast<int64_t>(ranges.size()));
    for (const auto &range : ranges)
        range.to_proto(req.add_ranges());
    auto [res, err] = create_client->send(CREATE_ENDPOINT, req);
    if (err)
        return err;
    for (size_t i = 0; i < static_cast<size_t>(res.ranges_size()); i++) {
        ranges[i].key = res.ranges(i).key();
        ranges[i].kv =
            RangeKV(ranges[i].key, kv_get_client, kv_set_client, kv_delete_client);
    }
    return xerrors::NIL;
}

auto RangeClient::create(Range &range) const -> xerrors::Error {
    auto req = api::v1::RangeCreateRequest();
    range.to_proto(req.add_ranges());
    auto [res, err] = create_client->send(CREATE_ENDPOINT, req);
    if (err)
        return err;
    if (res.ranges_size() == 0)
        return unexpected_missing("range");
    const auto rng = res.ranges(0);
    range.key = rng.key();
    range.kv = RangeKV(rng.key(), kv_get_client, kv_set_client, kv_delete_client);
    return err;
}

auto RangeClient::create(const std::string &name, telem::TimeRange time_range) const
    -> std::pair<Range, xerrors::Error> {
    auto rng = Range(name, time_range);
    auto err = create(rng);
    return { rng, err };
}

const std::string KV_SET_ENDPOINT = "/range/kv/set";
const std::string KV_GET_ENDPOINT = "/range/kv/get";
const std::string KV_DELETE_ENDPOINT = "/range/kv/delete";

auto RangeKV::get(const std::string &key) const
    -> std::pair<std::string, xerrors::Error> {
    auto req = api::v1::RangeKVGetRequest();
    req.add_keys(key);
    req.set_range_key(range_key);
    auto [res, err] = kv_get_client->send(KV_GET_ENDPOINT, req);
    if (err)
        return { "", err };
    if (res.pairs_size() == 0)
        return { "", xerrors::Error(xerrors::NOT_FOUND, "key not found") };
    return { res.pairs().at(0).value(), err };
}

auto RangeKV::set(const std::string &key, const std::string &value) const
    -> xerrors::Error {
    auto req = api::v1::RangeKVSetRequest();
    req.set_range_key(range_key);
    auto *const pair = req.add_pairs();
    pair->set_key(key);
    pair->set_value(value);
    auto [res, err] = kv_set_client->send(KV_SET_ENDPOINT, req);
    return err;
}

auto RangeKV::del(const std::string &key) const -> xerrors::Error {
    auto req = api::v1::RangeKVDeleteRequest();
    req.set_range_key(range_key);
    req.add_keys(key);
    auto [res, err] = kv_delete_client->send(KV_DELETE_ENDPOINT, req);
    return err;
}
