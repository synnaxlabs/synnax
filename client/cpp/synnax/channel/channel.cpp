// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <string>
#include <utility>
#include <vector>

#include "synnax/channel/channel.h"
#include <grpcpp/grpcpp.h>
#include "v1/channel.pb.h"
#include "synnax/exceptions.h"
#include "synnax/telem/telem.h"

using namespace Synnax;

std::string RETRIEVE_ENDPOINT = "/channel/retrieve";
std::string CREATE_ENDPOINT = "/channel/create";


Channel::Channel(const api::v1::Channel &ch) {
    name = ch.name();
    key = ch.key();
    index = ch.index();
    rate = Telem::Rate(ch.rate());
    is_index = ch.is_index();
    leaseholder = ch.leaseholder();
    dataType = Telem::DataType(ch.data_type());
}

Channel::Channel(
        std::string name,
        Telem::DataType dataType,
        Telem::Rate rate,
        bool is_index,
        std::uint32_t leaseholder,
        ChannelKey index,
        ChannelKey key
) : dataType(dataType),
    name(name),
    key(key),
    index(index),
    rate(rate),
    is_index(is_index),
    leaseholder(leaseholder) {}


void Channel::to_proto(api::v1::Channel *a) const {
    a->set_name(name);
    a->set_data_type(dataType.value);
    a->set_rate(rate.value);
    a->set_is_index(is_index);
    a->set_leaseholder(leaseholder);
    a->set_index(index);
    a->set_key(key);
}


Channel ChannelClient::create(
        std::string name,
        Telem::DataType data_type,
        Telem::Rate rate,
        ChannelKey index,
        bool is_index
) {
    auto req = api::v1::ChannelCreateRequest();
    auto a = req.add_channels();
    a->set_name(name);
    a->set_index(index);
    a->set_is_index(is_index);
    a->set_data_type(data_type.value);
    a->set_rate(rate.value);
    auto response = create_client->send(CREATE_ENDPOINT, req);
    return {response.first.channels(0)};
}

void ChannelClient::create(std::vector<Channel> &channels) {
    auto req = api::v1::ChannelCreateRequest();
    for (const auto &ch: channels) {
        auto a = req.add_channels();
        ch.to_proto(a);
    }
    auto response = create_client->send(CREATE_ENDPOINT, req);
    for (auto i = 0; i < response.first.channels_size(); i++)
        channels[i] = Channel(response.first.channels(i));
}


Channel ChannelClient::retrieve(ChannelKey key) {
    auto req = api::v1::ChannelRetrieveRequest();
    req.add_keys(key);
    std::pair<api::v1::ChannelRetrieveResponse, grpc::Status> response =
            retrieve_client->send(RETRIEVE_ENDPOINT, req);
    if (response.first.channels_size() == 0)
        throw QueryError("No channel found with key " + key);
    return Channel(response.first.channels(0));
}

Channel ChannelClient::retrieve(const std::string &name) {
    auto payload = api::v1::ChannelRetrieveRequest();
    payload.add_names(name);
    retrieve_client->send(RETRIEVE_ENDPOINT, payload);
}

std::vector<Channel> ChannelClient::retrieve(const std::vector<ChannelKey> &keys) {
    auto req = api::v1::ChannelRetrieveRequest();
    for (auto key: keys)
        req.add_keys(key);
    std::pair<api::v1::ChannelRetrieveResponse, grpc::Status> response =
            retrieve_client->send(RETRIEVE_ENDPOINT, req);
    std::vector<Channel> channels;
    for (auto i = 0; i < response.first.channels_size(); i++)
        channels.emplace_back(response.first.channels(i));
    return channels;
}

std::vector<Channel> ChannelClient::retrieve(const std::vector<std::string> &names) {
    auto req = api::v1::ChannelRetrieveRequest();
    for (const auto &name: names)
        req.add_names(name);
    std::pair<api::v1::ChannelRetrieveResponse, grpc::Status> response =
            retrieve_client->send(RETRIEVE_ENDPOINT, req);
    std::vector<Channel> channels;
    for (auto i = 0; i < response.first.channels_size(); i++)
        channels.emplace_back(response.first.channels(i));
    return channels;
}
