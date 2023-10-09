#include "synnax/telem.h"
#include <string>
#include <utility>
#include "v1/channel.pb.h"
#include "freighter/gRPC/client.h"
#include <grpcpp/grpcpp.h>

using ChannelKey = std::uint32_t;


typedef Client<
        api::v1::ChannelRetrieveResponse,
        api::v1::ChannelRetrieveRequest,
        gRPCStreamer<api::v1::ChannelRetrieveRequest, api::v1::ChannelRetrieveResponse, grpc::Status, api::v1::Channel>,
        grpc::Status> RetrieveClient;

typedef Client<
        api::v1::ChannelCreateResponse,
        api::v1::ChannelCreateRequest,
        gRPCStreamer<api::v1::ChannelCreateRequest, api::v1::ChannelCreateResponse, grpc::Status, api::v1::Channel>,
        grpc::Status> CreateClient;

class Channel {
public:
    DataType dataType;
    std::string name;
    ChannelKey key;
    ChannelKey index;
    Rate rate;
    bool is_index;
    std::uint32_t leaseholder;

    Channel(
            std::string name,
            DataType dataType,
            Rate rate = Rate(0),
            bool is_index = false,
            std::uint32_t leaseholder = 0,
            ChannelKey index = 0,
            ChannelKey key = 0
    );
};

class ChannelClient {
private:
    RetrieveClient *retrieve_client;
    CreateClient *create_client;

    ChannelClient(RetrieveClient *retrieve_client, CreateClient *create_client) :
            retrieve_client(retrieve_client),
            create_client(create_client) {}

public:
    Channel retrieve(const std::string &name);

    Channel retrieve(std::uint32_t key);

    std::vector<Channel> retrieve(const std::vector<std::string> &names);

    std::vector<Channel> retrieve(const std::vector<std::uint32_t> &keys);

    void create(std::vector<Channel> &channels);

    Channel create(std::string name, DataType data_type, Rate rate, ChannelKey index, bool is_index);
};
