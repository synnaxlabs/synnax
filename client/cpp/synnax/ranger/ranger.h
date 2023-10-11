#include <vector>
#include <memory>

#include "synnax/telem/telem.h"
#include "v1/ranger.pb.h"
#include "freighter/freighter.h"
#include <grpcpp/grpcpp.h>

typedef std::string Key;

using namespace Synnax;

typedef Freighter::UnaryClient<
        api::v1::RangeRetrieveResponse,
        api::v1::RangeRetrieveRequest,
        grpc::Status> RetrieveClient;

typedef Freighter::UnaryClient<
        api::v1::RangeCreateResponse,
        api::v1::RangeCreateRequest,
        grpc::Status> CreateClient;


typedef Freighter::UnaryClient<
        api::v1::RangeKVGetResponse,
        api::v1::RangeKVGetRequest,
        grpc::Status> KVGetClient;

struct Empty {
};

typedef Freighter::UnaryClient<
        Empty,
        api::v1::RangeKVSetRequest,
        grpc::Status> KVSetClient;

typedef Freighter::UnaryClient<
        Empty,
        api::v1::RangeKVDeleteRequest,
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