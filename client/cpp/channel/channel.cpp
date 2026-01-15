// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <utility>
#include <vector>

#include "client/cpp/channel/channel.h"
#include "client/cpp/channel/proto.gen.h"
#include "client/cpp/errors/errors.h"
#include "freighter/cpp/freighter.h"
#include "x/cpp/errors/errors.h"

namespace synnax::channel {
x::errors::Error Client::create(Channel &channel) const {
    auto req = grpc::channel::CreateRequest();
    *req.add_channels() = channel.to_proto();
    auto [res, err] = create_client->send("/channel/create", req);
    if (err) return err;
    if (res.channels_size() == 0) return unexpected_missing_error("channel");
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
    for (const auto &ch: channels)
        *req.add_channels() = ch.to_proto();
    auto [res, exc] = create_client->send("/channel/create", req);
    if (exc) return exc;
    for (auto i = 0; i < res.channels_size(); i++) {
        auto [ch, err] = Channel::from_proto(res.channels(i));
        if (err) return err;
        channels[i] = ch;
    }
    return x::errors::NIL;
}

std::pair<Channel, x::errors::Error> Client::retrieve(const Key key) const {
    auto req = grpc::channel::RetrieveRequest();
    req.add_keys(key);
    auto [res, err] = retrieve_client->send("/channel/retrieve", req);
    if (err) return {Channel{}, err};
    if (res.channels_size() == 0)
        return {Channel{}, not_found_error("channel", "key " + std::to_string(key))};
    return Channel::from_proto(res.channels(0));
}

std::pair<Channel, x::errors::Error> Client::retrieve(const std::string &name) const {
    auto payload = grpc::channel::RetrieveRequest();
    payload.add_names(name);
    auto [res, err] = retrieve_client->send("/channel/retrieve", payload);
    if (err) return {Channel{}, err};
    if (res.channels_size() == 0)
        return {Channel{}, not_found_error("channel", "name " + name)};
    if (res.channels_size() > 1)
        return {Channel{}, multiple_found_error("channels", "name " + name)};
    return Channel::from_proto(res.channels(0));
}

std::pair<std::vector<Channel>, x::errors::Error>
Client::retrieve(const std::vector<Key> &keys) const {
    grpc::channel::RetrieveRequest req;
    req.mutable_keys()->Add(keys.begin(), keys.end());
    auto [res, exc] = this->retrieve_client->send("/channel/retrieve", req);
    if (exc) return {{}, exc};
    std::vector<Channel> channels;
    channels.reserve(res.channels_size());
    for (const auto &pb: res.channels()) {
        auto [ch, err] = Channel::from_proto(pb);
        if (err) return {{}, err};
        channels.push_back(ch);
    }
    return {channels, x::errors::NIL};
}

std::pair<std::vector<Channel>, x::errors::Error>
Client::retrieve(const std::vector<std::string> &names) const {
    auto req = grpc::channel::RetrieveRequest();
    req.mutable_names()->Add(names.begin(), names.end());
    auto [res, err] = retrieve_client->send("/channel/retrieve", req);
    if (err) return {{}, err};
    std::vector<Channel> channels;
    channels.reserve(res.channels_size());
    for (const auto &pb: res.channels()) {
        auto [ch, parse_err] = Channel::from_proto(pb);
        if (parse_err) return {{}, parse_err};
        channels.push_back(ch);
    }
    return {channels, x::errors::NIL};
}
}
