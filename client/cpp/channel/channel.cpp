// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <unordered_set>
#include <utility>
#include <vector>

#include "client/cpp/channel/channel.h"
#include "client/cpp/channel/proto.gen.h"
#include "client/cpp/errors/errors.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/strings/strings.h"

namespace synnax::channel {
x::errors::Error Client::create(Channel &channel) const {
    auto req = grpc::channel::CreateRequest();
    auto [pb, pb_err] = channel.to_proto();
    if (pb_err) return pb_err;
    *req.add_channels() = pb;
    auto [res, err] = create_client->send("/channel/create", req);
    if (err) return err;
    if (res.channels_size() == 0) return errors::unexpected_missing_error("channel");
    auto [ch, parse_err] = Channel::from_proto(res.channels(0));
    if (parse_err) return parse_err;
    channel = ch;
    return x::errors::NIL;
}

std::pair<Channel, x::errors::Error> Client::create(
    const std::string &name,
    const x::telem::DataType &data_type,
    const Key index,
    const bool is_index
) const {
    auto ch = Channel{
        .name = name,
        .data_type = data_type,
        .is_index = is_index,
        .index = index,
    };
    auto err = create(ch);
    return {ch, err};
}

std::pair<Channel, x::errors::Error> Client::create(
    const std::string &name,
    const x::telem::DataType &data_type,
    const bool is_virtual
) const {
    auto ch = Channel{
        .name = name,
        .data_type = data_type,
        .is_virtual = is_virtual,
    };
    auto err = create(ch);
    return {ch, err};
}

x::errors::Error Client::create(std::vector<Channel> &channels) const {
    auto req = grpc::channel::CreateRequest();
    req.mutable_channels()->Reserve(static_cast<int>(channels.size()));
    for (const auto &ch: channels) {
        auto [pb, pb_err] = ch.to_proto();
        if (pb_err) return pb_err;
        *req.add_channels() = pb;
    }
    auto [res, exc] = create_client->send("/channel/create", req);
    if (exc) return exc;
    for (auto i = 0; i < res.channels_size(); i++) {
        auto [ch, err] = Channel::from_proto(res.channels(i));
        if (err) return err;
        channels[i] = ch;
    }
    return x::errors::NIL;
}

std::pair<std::vector<Channel>, x::errors::Error>
Client::retrieve(const RetrieveRequest &req) const {
    auto [pb_req, pb_err] = req.to_proto();
    if (pb_err) return {{}, pb_err};
    grpc::channel::RetrieveRequest grpc_req;
    *grpc_req.mutable_request() = pb_req;
    auto [res, err] = this->retrieve_client->send("/channel/retrieve", grpc_req);
    if (err) return {{}, err};
    auto [decoded, proto_err] = RetrieveResponse::from_proto(res);
    if (proto_err) return {{}, proto_err};
    return {std::move(decoded.channels), x::errors::NIL};
}

std::pair<Channel, x::errors::Error>
Client::retrieve_one(const RetrieveRequest &req, const std::string &query) const {
    auto [channels, err] = this->retrieve(req);
    if (err) return {Channel{}, err};
    if (channels.empty()) return {Channel{}, errors::not_found_error("channel", query)};
    if (channels.size() > 1)
        return {Channel{}, errors::multiple_found_error("channels", query)};
    return {std::move(channels[0]), x::errors::NIL};
}

std::pair<std::vector<Channel>, x::errors::Error>
Client::retrieve_keys(const RetrieveRequest &req, const std::vector<Key> &keys) const {
    auto [channels, err] = this->retrieve(req);
    if (err) return {{}, err};
    // Callers bind the channels to keys by position, so a missing key is an error.
    std::unordered_set<Key> found;
    for (const auto &ch: channels)
        found.insert(ch.key);
    std::vector<std::string> missing;
    for (const auto key: keys)
        if (!found.contains(key)) missing.push_back(std::to_string(key));
    if (!missing.empty())
        return {
            {},
            errors::not_found_error(
                "channels",
                "keys " + x::strings::join(missing, ", ")
            )
        };
    return {channels, x::errors::NIL};
}
}
