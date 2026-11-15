// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <string>
#include <variant>
#include <vector>

#include "client/cpp/errors/errors.h"
#include "client/cpp/ranger/ranger.h"

namespace synnax::ranger {
namespace {
/// @brief returns an integer timestamp as a string of its nanoseconds. The Core decodes
/// a JSON number in a set as a float, which cannot hold every timestamp exactly.
x::json::json exact(const x::json::json &time) {
    return time.is_number_integer() ? x::json::json(time.dump()) : time;
}

/// @brief returns the JSON form of an action, with the timestamps of a time range set
/// as exact strings.
x::json::json encode(const Action &action) {
    auto j = to_json(action);
    const auto *set = std::get_if<SetPayload>(&action);
    if (set == nullptr || set->path.empty() || set->path[0] != "time_range") return j;
    auto &value = j["set"]["value"];
    if (set->path.size() > 1)
        value = exact(value);
    else if (value.is_object())
        for (auto &time: value)
            time = exact(time);
    return j;
}
}

std::pair<Range, x::errors::Error>
Client::retrieve(const Key &key, const RetrieveOptions &options) const {
    return this->retrieve(Where{.key = key}, x::filter::one, options);
}

std::pair<Range, x::errors::Error>
Client::retrieve(const std::string &name, const RetrieveOptions &options) const {
    return this->retrieve(Where{.name = name}, x::filter::one, options);
}

std::pair<std::vector<Range>, x::errors::Error>
Client::retrieve(const std::vector<Key> &keys, const RetrieveOptions &options) const {
    return this->retrieve(Where{.key = keys}, options);
}

std::pair<std::vector<Range>, x::errors::Error> Client::retrieve(
    const std::vector<std::string> &names,
    const RetrieveOptions &options
) const {
    return this->retrieve(Where{.name = names}, options);
}

std::pair<std::vector<Range>, x::errors::Error>
Client::retrieve(const Where &where, const RetrieveOptions &options) const {
    auto [pb_req, pb_err] = RetrieveRequest::from(where, options).to_proto();
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

std::pair<Range, x::errors::Error> Client::retrieve(
    const Where &where,
    x::filter::One,
    const RetrieveOptions &options
) const {
    if (options.limit.has_value() || options.after.has_value())
        return {Range(), errors::paged_one_error()};
    auto [ranges, err] = this->retrieve(where, options);
    if (err) return {Range(), err};
    if (const auto one_err = errors::check_one("range", where.to_json(), ranges.size()))
        return {Range(), one_err};
    return {std::move(ranges[0]), x::errors::NIL};
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
Client::dispatch(const Key &key, const std::vector<Action> &actions) const {
    grpc::ranger::DispatchRequest req;
    req.set_key(key.to_string());
    for (const auto &action: actions)
        req.add_actions(encode(action).dump());
    return this->dispatch_client->send("/range/dispatch", req).second;
}

x::errors::Error Client::set_end(const Key &key, const x::telem::TimeStamp end) const {
    return this->dispatch(
        key,
        {set({.path = {"time_range", "end"}, .value = end.nanoseconds()})}
    );
}

}
