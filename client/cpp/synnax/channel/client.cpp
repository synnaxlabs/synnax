//
// Created by Emiliano Bonilla on 10/8/23.
//

#include <string>
#include <utility>
#include <exception>
#include <vector>

#include "channel.h"
#include "grpc.h"

std::string RETRIEVE_ENDPOINT = "/channel/retrieve";
std::string CREATE_ENDPOINT = "/channel/create";

typedef Client<
        api::v1::ChannelRetrieveRequest,
        api::v1::ChannelRetrieveResponse,
        gRPCStreamer<api::v1::ChannelRetrieveRequest, api::v1::ChannelRetrieveResponse, grpc::Status, api::v1::Channel>,
        grpc::Status,
> RetrieveClient;

typedef Client<
        api::v1::ChannelCreateRequest,
        api::v1::ChannelCreateResponse,
        gRPCStreamer<api::v1::ChannelCreateRequest, api::v1::ChannelCreateResponse, grpc::Status, api::v1::Channel>,
        grpc::Status,
> CreateClient;


Channel translate_channel_forward(api::v1::Channel ch) {
    return Channel(
            ch.name(),
            DataType(ch.data_type()),
            Rate(ch.rate()),
            ch.is_index(),
            ch.leaseholder(),
            ch.index(),
            ch.key()
    );
}

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

class ChannelClient {
private:
    RetrieveClient retrieve_client;
    CreateClient create_client;
public:
    ChannelClient(RetrieveClient retrieve_client, CreateClient create_client) :
            retrieve_client(retrieve_client),
            create_client(create_client) {}

    Channel create(
            std::string name,
            DataType data_type,
            Rate rate,
            ChannelKey index,
            bool is_index
    ) {
        auto req_pld = api::v1::ChannelCreateRequest();
        auto a = req_pld.add_channels();
        a->set_name(name);
        a->set_index(index);
        a->set_is_index(is_index);
        a->set_data_type(data_type.value);
        a->set_rate(rate.value);
        auto response = create_client.send(CREATE_ENDPOINT, req_pld);
        return translate_channel_forward(response.first.channels(0));
    }

    void create(std::vector <Channel> &channels) {
        auto req_pld = api::v1::ChannelCreateRequest();
        for (auto ch: channels) {
            auto a = req_pld.add_channels();
            translate_channel_backward(ch, a);
        }
        auto response = create_client.send(CREATE_ENDPOINT, req_pld);
        for (auto i = 0; i < response.first.channels_size(); i++)
            channels[i] = translate_channel_forward(response.first.channels(i));
    }


    Channel retrieve(ChannelKey key) {
        auto payload = api::v1::ChannelRetrieveRequest();
        payload.add_keys(key);
        std::pair <api::v1::ChannelRetrieveResponse, grpc::Status> response =
                retrieve_client.send(RETRIEVE_ENDPOINT, payload);
        if (response.first.channels_size() == 0)
            throw QueryError("No channel found with key " + key);
        return translate_channel_forward(response.first.channels(0));
    }

    Channel retrieve(std::string name) {
        auto payload = api::v1::ChannelRetrieveRequest();
        payload.add_names(name);
        retrieve_client.send(RETRIEVE_ENDPOINT, payload);
    }

    std::vector<Channel> retrieve(std::vector<ChannelKey> keys) {
        auto payload = api::v1::ChannelRetrieveRequest();
        for (auto key: keys)
            payload.add_keys(key);
        std::pair <api::v1::ChannelRetrieveResponse, grpc::Status> response =
                retrieve_client.send(RETRIEVE_ENDPOINT, payload);
        std::vector<Channel> channels;
        for (auto i = 0; i < response.first.channels_size(); i++)
            channels.push_back(translate_channel_forward(response.first.channels(i)));
        return channels;
    }

    std::vector<Channel> retrieve(std::vector<std::string> names) {
        auto payload = api::v1::ChannelRetrieveRequest();
        for (auto name: names)
            payload.add_names(name);
        std::pair <api::v1::ChannelRetrieveResponse, grpc::Status> response =
                retrieve_client.send(RETRIEVE_ENDPOINT, payload);
        std::vector<Channel> channels;
        for (auto i = 0; i < response.first.channels_size(); i++)
            channels.push_back(translate_channel_forward(response.first.channels(i)));
        return channels;
    }
};
