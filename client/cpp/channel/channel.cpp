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
Channel Channel::from_proto(const api::v1::Channel &ch) {
    return Channel{
        .name = ch.name(),
        .data_type = x::telem::DataType(ch.data_type()),
        .key = ch.key(),
        .index = ch.index(),
        .is_index = ch.is_index(),
        .leaseholder = ch.leaseholder(),
        .is_virtual = ch.is_virtual(),
        .internal = ch.internal(),
    };
}

void Channel::to_proto(api::v1::Channel *ch) const {
    ch->set_name(name);
    ch->set_data_type(data_type.name());
    ch->set_is_index(is_index);
    ch->set_leaseholder(leaseholder);
    ch->set_index(index);
    ch->set_key(key);
    ch->set_is_virtual(is_virtual);
}

x::errors::Error Client::create(synnax::channel::Channel &channel) const {
    auto req = api::v1::ChannelCreateRequest();
    channel.to_proto(req.add_channels());
    auto [res, err] = create_client->send("/channel/create", req);
    if (err) return err;
    if (res.channels_size() == 0) return errors::unexpected_missing_error("channel");
    auto created = Channel::from_proto(res.channels(0));
    channel = created;
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
        .index = index,
        .is_index = is_index,
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
    auto req = api::v1::ChannelCreateRequest();
    req.mutable_channels()->Reserve(static_cast<int>(channels.size()));
    for (const auto &ch: channels)
        *req.add_channels() = ch.to_proto();
    auto [res, exc] = create_client->send("/channel/create", req);
    for (auto i = 0; i < res.channels_size(); i++)
        channels[i] = Channel::from_proto(res.channels(i));
    return exc;
}

std::pair<Channel, x::errors::Error> Client::retrieve(const Key key) const {
    auto req = api::v1::ChannelRetrieveRequest();
    req.add_keys(key);
    auto [res, err] = retrieve_client->send("/channel/retrieve", req);
    if (err) return {Channel{}, err};
    if (res.channels_size() == 0)
        return {
            Channel{},
            errors::not_found_error("channel", "key " + std::to_string(key))
        };
    return {Channel::from_proto(res.channels(0)), err};
}

std::pair<Channel, x::errors::Error> Client::retrieve(const std::string &name) const {
    auto payload = api::v1::ChannelRetrieveRequest();
    payload.add_names(name);
    auto [res, err] = retrieve_client->send("/channel/retrieve", payload);
    if (err) return {Channel{}, err};
    if (res.channels_size() == 0)
        return {Channel{}, errors::not_found_error("channel", "name " + name)};
    if (res.channels_size() > 1)
        return {Channel{}, errors::multiple_found_error("channels", "name " + name)};
    return {Channel::from_proto(res.channels(0)), err};
}

std::pair<std::vector<Channel>, x::errors::Error>
Client::retrieve(const std::vector<Key> &keys) const {
    api::v1::ChannelRetrieveRequest req;
    req.mutable_keys()->Add(keys.begin(), keys.end());
    auto [res, exc] = this->retrieve_client->send("/channel/retrieve", req);
    std::vector<Channel> channels;
    channels.reserve(res.channels_size());
    for (const auto &ch: res.channels())
        channels.push_back(Channel::from_proto(ch));
    return {channels, exc};
}

std::pair<std::vector<Channel>, x::errors::Error>
Client::retrieve(const std::vector<std::string> &names) const {
    auto req = api::v1::ChannelRetrieveRequest();
    req.mutable_names()->Add(names.begin(), names.end());
    auto [res, err] = retrieve_client->send("/channel/retrieve", req);
    std::vector<Channel> channels;
    channels.reserve(res.channels_size());
    for (const auto &ch: res.channels())
        channels.push_back(Channel::from_proto(ch));
    return {channels, err};
}
}
