// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "client/cpp/channel/channel.h"

#include <vector>

#include "client/cpp/errors/errors.h"
#include "freighter/cpp/freighter.h"

using namespace synnax;

const std::string CREATE_ENDPOINT = "/api/v1/channel/create";
const std::string RETRIEVE_ENDPOINT = "/api/v1/channel/retrieve";

/// @brief proto ctor.
Channel::Channel(
    const api::v1::Channel &ch
) : name(ch.name()),
    data_type(
        synnax::DataType(ch.data_type())),
    key(ch.key()),
    index(ch.index()),
    rate(synnax::Rate(ch.rate())),
    is_index(ch.is_index()),
    leaseholder(ch.leaseholder()) {
}

/// @brief rate based ctor.
Channel::Channel(
    const std::string &name,
    synnax::DataType data_type,
    synnax::Rate rate
) : name(name), data_type(data_type), rate(rate) {
}

/// @brief index based ctor.
Channel::Channel(
    const std::string &name,
    synnax::DataType data_type,
    ChannelKey index,
    bool is_index
) : name(name), data_type(data_type), index(index), is_index(is_index) {
}

Channel::Channel(
    const std::string &name,
    synnax::DataType data_type,
    bool is_virtual
): name(name), data_type(data_type), is_virtual(is_virtual) {
}


/// @brief binds to the given proto.
void Channel::to_proto(api::v1::Channel *a) const {
    a->set_name(name);
    a->set_data_type(data_type.name());
    a->set_rate(rate.value);
    a->set_is_index(is_index);
    a->set_leaseholder(leaseholder);
    a->set_index(index);
    a->set_key(key);
    a->set_is_virtual(is_virtual);
}

/// @brief create from channel.
freighter::Error ChannelClient::create(synnax::Channel &channel) const {
    auto req = api::v1::ChannelCreateRequest();
    channel.to_proto(req.add_channels());
    auto [res, exc] = create_client->send(CREATE_ENDPOINT, req);
    if (!exc) {
        if (res.channels_size() == 0)
            return freighter::Error(synnax::UNEXPECTED_ERROR,
                                    "no channels returned from server on create. please report this issue to the synnax team");
        auto first = res.channels(0);
        channel.key = first.key();
        channel.name = first.name();
        channel.data_type = synnax::DataType(first.data_type());
        channel.rate = synnax::Rate(first.rate());
        channel.is_index = first.is_index();
        channel.leaseholder = first.leaseholder();
        channel.index = first.index();
        channel.internal = first.internal();
    }
    return exc;
}

/// @brief index based create.
std::pair<Channel, freighter::Error> ChannelClient::create(
    const std::string &name,
    synnax::DataType data_type,
    ChannelKey index,
    bool is_index) const {
    auto ch = Channel(name, data_type, index, is_index);
    auto err = create(ch);
    return {ch, err};
}

/// @brief rate based create.
std::pair<Channel, freighter::Error> ChannelClient::create(
    const std::string &name,
    synnax::DataType data_type,
    synnax::Rate rate) const {
    auto ch = Channel(name, data_type, rate);
    auto err = create(ch);
    return {ch, err};
}

/// @brief multiple channel create.
freighter::Error ChannelClient::create(std::vector<Channel> &channels) const {
    auto req = api::v1::ChannelCreateRequest();
    req.mutable_channels()->Reserve(int(channels.size()));
    for (const auto &ch: channels)
        ch.to_proto(req.add_channels());
    auto [res, exc] = create_client->send(CREATE_ENDPOINT, req);
    for (auto i = 0; i < res.channels_size(); i++)
        channels[i] = Channel(res.channels(i));
    return exc;
}

/// @brief key based retrieve.
std::pair<Channel, freighter::Error> ChannelClient::retrieve(ChannelKey key) const {
    auto req = api::v1::ChannelRetrieveRequest();
    req.add_keys(key);
    auto [res, err] = retrieve_client->send(RETRIEVE_ENDPOINT, req);
    if (err) return {Channel(), err};
    if (res.channels_size() == 0)
        return {
            Channel(),
            freighter::Error(synnax::NOT_FOUND,
                             "no channels found matching key " + std::to_string(key))
        };
    return {Channel(res.channels(0)), err};
}

/// @brief name based retrieve.
std::pair<Channel, freighter::Error> ChannelClient::retrieve(
    const std::string &name) const {
    auto payload = api::v1::ChannelRetrieveRequest();
    payload.add_names(name);
    auto [res, err] = retrieve_client->send(RETRIEVE_ENDPOINT, payload);
    if (err)
        return {Channel(), err};
    if (res.channels_size() == 0)
        return {
            Channel(),
            freighter::Error(synnax::NOT_FOUND,
                             "no channels found matching name " + name)
        };
    if (res.channels_size() > 1)
        return {
            Channel(),
            freighter::Error(synnax::QUERY_ERROR,
                             "multiple channels found matching name " + name)
        };
    return {Channel(res.channels(0)), err};
}

/// @brief multiple key based retrieve.
std::pair<std::vector<Channel>, freighter::Error>
ChannelClient::retrieve(const std::vector<ChannelKey> &keys) const {
    auto req = api::v1::ChannelRetrieveRequest();
    req.mutable_keys()->Add(keys.begin(), keys.end());
    auto [res, exc] = retrieve_client->send(RETRIEVE_ENDPOINT, req);
    std::vector<Channel> channels = {res.channels().begin(), res.channels().end()};
    return {channels, exc};
}

/// @brief multiple name based retrieve.
std::pair<std::vector<Channel>, freighter::Error>
ChannelClient::retrieve(const std::vector<std::string> &names) const {
    auto req = api::v1::ChannelRetrieveRequest();
    req.mutable_names()->Add(names.begin(), names.end());
    auto [res, err] = retrieve_client->send(RETRIEVE_ENDPOINT, req);
    std::vector<Channel> channels = {res.channels().begin(), res.channels().end()};
    return {channels, err};
}
