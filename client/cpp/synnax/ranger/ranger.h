#include <vector>
#include <memory>

#include "synnax/telem/telem.h"
#include "v1/ranger.pb.h"
#include "freighter/gRPC/client.h"

typedef std::string Key;

using namespace Synnax;

typedef Client<
        api::v1::RangeRetrieveResponse,
        api::v1::RangeRetrieveRequest,
        gRPCStreamer<api::v1::RangeRetrieveRequest, api::v1::RangeRetrieveResponse, grpc::Status, api::v1::Range>,
        grpc::Status> RetrieveClient;

typedef Client<
        api::v1::RangeCreateResponse,
        api::v1::RangeCreateRequest,
        gRPCStreamer<api::v1::RangeCreateRequest, api::v1::RangeCreateResponse, grpc::Status, api::v1::Range>,
        grpc::Status> CreateClient;


typedef Client<
        api::v1::RangeKVGetResponse,
        api::v1::RangeKVGetRequest,
        gRPCStreamer<api::v1::RangeKVGetRequest, api::v1::RangeKVGetResponse, grpc::Status, api::v1::Range>,
        grpc::Status> KVGetClient;

struct Empty {
};

typedef Client<
        Empty,
        api::v1::RangeKVSetRequest,
        gRPCStreamer<api::v1::RangeKVSetRequest, Empty, grpc::Status, api::v1::Range>,
        grpc::Status> KVSetClient;

typedef Client<
        Empty,
        api::v1::RangeKVDeleteRequest,
        gRPCStreamer<api::v1::RangeKVDeleteRequest, Empty, grpc::Status, api::v1::Range>,
        grpc::Status> KVDeleteClient;


class KV {
private:
    std::string range_key;
    std::unique_ptr<KVGetClient> kv_get_client;
    std::unique_ptr<KVSetClient> kv_set_client;
    std::unique_ptr<KVDeleteClient> kv_delete_client;
public:
    std::string get(std::string key);

    void set(std::string key, std::string value);

    void delete_(std::string key);
};

class Range {
public:
    Key key;
    std::string name;
    Telem::TimeRange time_range;
    KV *kv;

    Range(std::string name, Telem::TimeRange time_range);

    Range(Key key, std::string name, Telem::TimeRange time_range);

    Range(Key key, std::string name, Telem::TimeRange time_range, KV *kv);
};

class RangeClient {
private:
    RetrieveClient *retrieve_client;
    CreateClient *create_client;
public:
    Range retrieve_by_key(std::string key);

    Range retrieve_by_name(std::string name);

    std::vector<Range> retrieve_by_key(std::vector<std::string> keys);

    std::vector<Range> retrieve_by_name(std::vector<std::string> names);

    void create(std::vector<Range> &ranges);

    void create(Range &range);

    Range create(std::string name, Telem::TimeRange time_range);
};