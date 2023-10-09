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


Channel translate_channel_forward(const api::v1::Channel &ch) {
    return {
            ch.name(),
            Telem::DataType(ch.data_type()),
            Telem::Rate(ch.rate()),
            ch.is_index(),
            ch.leaseholder(),
            ch.index(),
            ch.key()
    };
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


api::v1::Channel translate_channel_backward(Channel ch, api::v1::Channel *a) {
    a->set_name(ch.name);
    a->set_data_type(ch.dataType.value);
    a->set_rate(ch.rate.value);
    a->set_is_index(ch.is_index);
    a->set_leaseholder(ch.leaseholder);
    a->set_index(ch.index);
    a->set_key(ch.key);
    return *a;
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
    return translate_channel_forward(response.first.channels(0));
}

void ChannelClient::create(std::vector<Channel> &channels) {
    auto req = api::v1::ChannelCreateRequest();
    for (const auto &ch: channels) {
        auto a = req.add_channels();
        translate_channel_backward(ch, a);
    }
    auto response = create_client->send(CREATE_ENDPOINT, req);
    for (auto i = 0; i < response.first.channels_size(); i++)
        channels[i] = translate_channel_forward(response.first.channels(i));
}


Channel ChannelClient::retrieve(ChannelKey key) {
    auto req = api::v1::ChannelRetrieveRequest();
    req.add_keys(key);
    std::pair<api::v1::ChannelRetrieveResponse, grpc::Status> response =
            retrieve_client->send(RETRIEVE_ENDPOINT, req);
    if (response.first.channels_size() == 0)
        throw QueryError("No channel found with key " + key);
    return translate_channel_forward(response.first.channels(0));
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
        channels.push_back(translate_channel_forward(response.first.channels(i)));
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
        channels.push_back(translate_channel_forward(response.first.channels(i)));
    return channels;
}
