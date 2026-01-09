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
#include "client/cpp/errors/errors.h"
#include "freighter/cpp/freighter.h"
#include "x/cpp/errors/errors.h"

namespace synnax {
Channel::Channel(const api::channel::Channel &ch):
    name(ch.name()),
    data_type(x::telem::DataType(ch.data_type())),
    key(ch.key()),
    index(ch.index()),
    is_index(ch.is_index()),
    leaseholder(ch.leaseholder()),
    is_virtual(ch.virtual_()),
    internal(ch.internal()) {}

Channel::Channel(
    std::string name,
    x::telem::DataType data_type,
    const ChannelKey index,
    const bool is_index
):
    name(std::move(name)),
    data_type(std::move(data_type)),
    index(index),
    is_index(is_index) {}

Channel::Channel(std::string name, x::telem::DataType data_type, const bool is_virtual):
    name(std::move(name)), data_type(std::move(data_type)), is_virtual(is_virtual) {}

void Channel::to_proto(api::channel::Channel *ch) const {
    ch->set_name(name);
    ch->set_data_type(data_type.name());
    ch->set_is_index(is_index);
    ch->set_leaseholder(leaseholder);
    ch->set_index(index);
    ch->set_key(key);
    ch->set_virtual_(is_virtual);
}

x::errors::Error ChannelClient::create(synnax::Channel &channel) const {
    auto req = grpc::channel::CreateRequest();
    channel.to_proto(req.add_channels());
    auto [res, err] = create_client->send("/channel/create", req);
    if (err) return err;
    if (res.channels_size() == 0) return unexpected_missing_error("channel");
    const auto first = res.channels(0);
    channel.key = first.key();
    channel.name = first.name();
    channel.data_type = x::telem::DataType(first.data_type());
    channel.is_index = first.is_index();
    channel.leaseholder = first.leaseholder();
    channel.index = first.index();
    channel.internal = first.internal();
    return x::errors::NIL;
}

std::pair<Channel, x::errors::Error> ChannelClient::create(
    const std::string &name,
    const x::telem::DataType &data_type,
    const ChannelKey index,
    const bool is_index
) const {
    auto ch = Channel(name, data_type, index, is_index);
    auto err = create(ch);
    return {ch, err};
}

std::pair<Channel, x::errors::Error> ChannelClient::create(
    const std::string &name,
    const x::telem::DataType &data_type,
    const bool is_virtual
) const {
    auto ch = Channel(name, data_type, is_virtual);
    auto err = create(ch);
    return {ch, err};
}

x::errors::Error ChannelClient::create(std::vector<Channel> &channels) const {
    auto req = grpc::channel::CreateRequest();
    req.mutable_channels()->Reserve(static_cast<int>(channels.size()));
    for (const auto &ch: channels)
        ch.to_proto(req.add_channels());
    auto [res, exc] = create_client->send("/channel/create", req);
    for (auto i = 0; i < res.channels_size(); i++)
        channels[i] = Channel(res.channels(i));
    return exc;
}

std::pair<Channel, x::errors::Error>
ChannelClient::retrieve(const ChannelKey key) const {
    auto req = grpc::channel::RetrieveRequest();
    req.add_keys(key);
    auto [res, err] = retrieve_client->send("/channel/retrieve", req);
    if (err) return {Channel(), err};
    if (res.channels_size() == 0)
        return {Channel(), not_found_error("channel", "key " + std::to_string(key))};
    return {Channel(res.channels(0)), err};
}

std::pair<Channel, x::errors::Error>
ChannelClient::retrieve(const std::string &name) const {
    auto payload = grpc::channel::RetrieveRequest();
    payload.add_names(name);
    auto [res, err] = retrieve_client->send("/channel/retrieve", payload);
    if (err) return {Channel(), err};
    if (res.channels_size() == 0)
        return {Channel(), not_found_error("channel", "name " + name)};
    if (res.channels_size() > 1)
        return {Channel(), multiple_found_error("channels", "name " + name)};
    return {Channel(res.channels(0)), err};
}

std::pair<std::vector<Channel>, x::errors::Error>
ChannelClient::retrieve(const std::vector<ChannelKey> &keys) const {
    grpc::channel::RetrieveRequest req;
    req.mutable_keys()->Add(keys.begin(), keys.end());
    auto [res, exc] = this->retrieve_client->send("/channel/retrieve", req);
    std::vector<Channel> channels = {res.channels().begin(), res.channels().end()};
    return {channels, exc};
}

std::pair<std::vector<Channel>, x::errors::Error>
ChannelClient::retrieve(const std::vector<std::string> &names) const {
    auto req = grpc::channel::RetrieveRequest();
    req.mutable_names()->Add(names.begin(), names.end());
    auto [res, err] = retrieve_client->send("/channel/retrieve", req);
    std::vector<Channel> channels = {res.channels().begin(), res.channels().end()};
    return {channels, err};
}
}
