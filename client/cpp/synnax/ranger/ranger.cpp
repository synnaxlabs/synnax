//
// Created by Emiliano Bonilla on 10/8/23.
//

#include "synnax/ranger/ranger.h"
#include "v1/ranger.pb.h"
#include "telempb/telem.pb.h"
#include <grpcpp/grpcpp.h>
#include "synnax/exceptions.h"

api::v1::Range translate_forward(Range ch, api::v1::Range *a) {
    a->set_name(ch.name);
    a->set_key(ch.key);
    auto tr = telempb::TimeRange();
    tr.set_start(ch.time_range.start.value);
    tr.set_end(ch.time_range.end.value);
    a->set_allocated_time_range(&tr);
    return *a;
}

Range translate_backward(api::v1::Range *a, KV *kv) {
    return Range(
            a->key(),
            a->name(),
            TimeRange(TimeStamp(a->time_range().start()), TimeStamp(a->time_range().end())), kv
    );
}


Range::Range(Key key, std::string name, TimeRange time_range) :
        key(key),
        name(name),
        time_range(time_range) {
    kv = nullptr;
}

Range::Range(Key key, std::string name, TimeRange time_range, KV *kv) :
        key(key),
        name(name),
        time_range(time_range),
        kv(kv) {}

std::string RETRIEVE_ENDPOINT = "/range/retrieve";
std::string CREATE_ENDPOINT = "/range/create";

Range RangeClient::retrieve_by_key(std::string key) {
    auto req = api::v1::RangeRetrieveRequest();
    req.add_keys(key);
    auto [res, err] = retrieve_client->send(RETRIEVE_ENDPOINT, req);
    if (!err.ok())
        throw QueryError(err.error_message());
    if (res.ranges_size() == 0)
        throw QueryError("No range found with key " + key);
    return translate_backward(res.mutable_ranges(0), nullptr);
}

Range RangeClient::retrieve_by_name(std::string name) {
    auto req = api::v1::RangeRetrieveRequest();
    req.add_names(name);
    auto [res, err] = retrieve_client->send(RETRIEVE_ENDPOINT, req);
    if (!err.ok())
        throw QueryError(err.error_message());
    if (res.ranges_size() == 0)
        throw QueryError("No range found with name " + name);
    return translate_backward(res.mutable_ranges(0), nullptr);
}

std::vector<Range> RangeClient::retrieve_by_key(std::vector<std::string> keys) {
    auto req = api::v1::RangeRetrieveRequest();
    for (auto &key: keys)
        req.add_keys(key);
    auto [res, err] = retrieve_client->send(RETRIEVE_ENDPOINT, req);
    if (!err.ok())
        throw QueryError(err.error_message());
    std::vector<Range> ranges;
    ranges.reserve(res.ranges_size());
    for (auto i = 0; i < res.ranges_size(); i++)
        ranges.push_back(translate_backward(res.mutable_ranges(i), nullptr));
    return ranges;
}

std::vector<Range> RangeClient::retrieve_by_name(std::vector<std::string> names) {
    auto req = api::v1::RangeRetrieveRequest();
    for (auto &name: names)
        req.add_names(name);
    auto [res, err] = retrieve_client->send(RETRIEVE_ENDPOINT, req);
    if (!err.ok())
        throw QueryError(err.error_message());
    std::vector<Range> ranges;
    ranges.reserve(res.ranges_size());
    for (auto i = 0; i < res.ranges_size(); i++)
        ranges.push_back(translate_backward(res.mutable_ranges(i), nullptr));
    return ranges;
}

void RangeClient::create(std::vector<Range> &ranges) {
    auto req = api::v1::RangeCreateRequest();
    for (auto &range: ranges) {
        auto rng = req.add_ranges();
        translate_forward(range, rng);
    }
    auto [res, err] = create_client->send(CREATE_ENDPOINT, req);
    if (!err.ok())
        throw QueryError(err.error_message());
    for (auto i = 0; i < res.ranges_size(); i++)
        ranges[i].key = res.ranges(i).key();
}


void RangeClient::create(Range &range) {
    auto req = api::v1::RangeCreateRequest();
    auto rng = req.add_ranges();
    translate_forward(range, rng);
    auto [res, err] = create_client->send(CREATE_ENDPOINT, req);
    if (!err.ok())
        throw QueryError(err.error_message());
    range.key = res.ranges(0).key();
}

Range RangeClient::create(std::string name, TimeRange time_range) {
    auto rng = Range(name, time_range);
    create(rng);
    return rng;
}

std::string KV_SET_ENDPOINT = "/range/kv/set";
std::string KV_GET_ENDPOINT = "/range/kv/get";
std::string KV_DELETE_ENDPOINT = "/range/kv/delete";


std::string KV::get(std::string key) {
    auto req = api::v1::RangeKVGetRequest();
    req.add_keys(key);
    auto [res, err] = kv_get_client->send(KV_GET_ENDPOINT, req);
    if (!err.ok())
        throw QueryError(err.error_message());
    return res.pairs().at(key);
}

void KV::set(std::string key, std::string value) {
    auto req = api::v1::RangeKVSetRequest();
    req.set_range_key(range_key);
    (*req.mutable_pairs())[key] = value;
    auto [res, err] = kv_set_client->send(KV_SET_ENDPOINT, req);
    if (!err.ok())
        throw QueryError(err.error_message());
}

void KV::delete_(std::string key) {
    auto req = api::v1::RangeKVDeleteRequest();
    req.set_range_key(range_key);
    req.add_keys(key);
    auto [res, err] = kv_delete_client->send(KV_DELETE_ENDPOINT, req);
    if (!err.ok())
        throw QueryError(err.error_message());
}