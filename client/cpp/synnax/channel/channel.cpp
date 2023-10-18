// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// std
#include <vector>

/// internal
#include "synnax/channel/channel.h"

using namespace Synnax;

const std::string CREATE_ENDPOINT = "/api/v1/channel/create";
const std::string RETRIEVE_ENDPOINT = "/api/v1/channel/retrieve";

/// @brief proto ctor.
Channel::Channel::Channel(const api::v1::Channel &ch) :
        name(ch.name()),
        data_type(Telem::DataType(ch.data_type())),
        key(ch.key()),
        index(ch.index()),
        rate(Telem::Rate(ch.rate())),
        is_index(ch.is_index()),
        leaseholder(ch.leaseholder()) {}

/// @brief rate based ctor.
Channel::Channel::Channel(
        const std::string &name,
        Telem::DataType data_type,
        Telem::Rate rate
) :
        name(name),
        data_type(data_type),
        rate(rate) {}

/// @brief index based ctor.
Channel::Channel::Channel(
        const std::string &name,
        Telem::DataType data_type,
        Key index,
        bool is_index
) : name(name), data_type(data_type), index(index), is_index(is_index) {}

/// @brief binds to the given proto.
void Channel::Channel::to_proto(api::v1::Channel *a) const {
    a->set_name(name);
    a->set_data_type(data_type.value);
    a->set_rate(rate.value);
    a->set_is_index(is_index);
    a->set_leaseholder(leaseholder);
    a->set_index(index);
    a->set_key(key);
}

/// @brief create from channel.
Freighter::Error Channel::Client::create(Synnax::Channel::Channel &channel) const {
    auto req = api::v1::ChannelCreateRequest();
    channel.to_proto(req.add_channels());
    auto [res, exc] = create_client->send(CREATE_ENDPOINT, req);
    if (!exc) {
        auto first = res.channels(0);
        channel.key = first.key();
        channel.name = first.name();
        channel.data_type = Telem::DataType(first.data_type());
        channel.rate = Telem::Rate(first.rate());
        channel.is_index = first.is_index();
        channel.leaseholder = first.leaseholder();
        channel.index = first.index();
    }
    return exc;
}

/// @brief index based create.
std::pair<Channel::Channel, Freighter::Error> Channel::Client::create(
        std::string name,
        Telem::DataType data_type,
        Key index,
        bool is_index
) const {
    auto ch = Channel(name, data_type, index, is_index);
    auto err = create(ch);
    return {ch, err};
}

/// @brief rate based create.
std::pair<Channel::Channel, Freighter::Error> Channel::Client::create(
        std::string name,
        Telem::DataType data_type,
        Telem::Rate rate
) const {
    auto ch = Channel(name, data_type, rate);
    auto err = create(ch);
    return {ch, err};
}

/// @brief multiple channel create.
Freighter::Error Channel::Client::create(std::vector<Channel> &channels) const {
    auto req = api::v1::ChannelCreateRequest();
    req.mutable_channels()->Reserve(int(channels.size()));
    for (const auto &ch: channels) ch.to_proto(req.add_channels());
    auto [res, exc] = create_client->send(CREATE_ENDPOINT, req);
    for (auto i = 0; i < res.channels_size(); i++)
        channels[i] = Channel(res.channels(i));
    return exc;
}


/// @brief key based retrieve.
std::pair<Channel::Channel, Freighter::Error> Channel::Client::retrieve(Key key) const {
    auto req = api::v1::ChannelRetrieveRequest();
    req.add_keys(key);
    auto [res, err] = retrieve_client->send(RETRIEVE_ENDPOINT, req);
    if (err) return {Channel(), err};
    return {Channel(res.channels(0)), err};
}

/// @brief name based retrieve.
std::pair<Channel::Channel, Freighter::Error> Channel::Client::retrieve(const std::string &name) const {
    auto payload = api::v1::ChannelRetrieveRequest();
    payload.add_names(name);
    auto [res, err] = retrieve_client->send(RETRIEVE_ENDPOINT, payload);
    if (err) return {Channel(), err};
    return {Channel(res.channels(0)), err};
}

/// @brief multiple key based retrieve.
std::pair<std::vector<Channel::Channel>, Freighter::Error>
Channel::Client::retrieve(const std::vector<Key> &keys) const {
    auto req = api::v1::ChannelRetrieveRequest();
    req.mutable_keys()->Add(keys.begin(), keys.end());
    auto [res, exc] = retrieve_client->send(RETRIEVE_ENDPOINT, req);
    std::vector<Channel> channels = {res.channels().begin(), res.channels().end()};
    return {channels, exc};
}

/// @brief multiple name based retrieve.
std::pair<std::vector<Channel::Channel>, Freighter::Error>
Channel::Client::retrieve(const std::vector<std::string> &names) const {
    auto req = api::v1::ChannelRetrieveRequest();
    req.mutable_names()->Add(names.begin(), names.end());
    auto [res, err] = retrieve_client->send(RETRIEVE_ENDPOINT, req);
    std::vector<Channel> channels = {res.channels().begin(), res.channels().end()};
    return {channels, err};
}


