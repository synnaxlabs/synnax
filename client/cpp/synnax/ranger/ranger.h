// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <vector>
#include <memory>

#include "synnax/telem/telem.h"
#include "v1/ranger.pb.h"
#include "freighter/freighter.h"
#include <grpcpp/grpcpp.h>
#include "google/protobuf/empty.pb.h"

typedef std::string Key;

using namespace Synnax;

namespace Ranger {
    typedef Freighter::UnaryClient<
            api::v1::RangeRetrieveResponse,
            api::v1::RangeRetrieveRequest,
            grpc::Status
    > RetrieveClient;

    typedef Freighter::UnaryClient<
            api::v1::RangeCreateResponse,
            api::v1::RangeCreateRequest,
            grpc::Status
    > CreateClient;


    typedef Freighter::UnaryClient<
            api::v1::RangeKVGetResponse,
            api::v1::RangeKVGetRequest,
            grpc::Status
    > KVGetClient;

    typedef Freighter::UnaryClient<
            google::protobuf::Empty,
            api::v1::RangeKVSetRequest,
            grpc::Status> KVSetClient;

    typedef Freighter::UnaryClient<
            google::protobuf::Empty,
            api::v1::RangeKVDeleteRequest,
            grpc::Status> KVDeleteClient;


    class KV {
    private:
        std::string range_key;
        std::unique_ptr<KVGetClient> kv_get_client;
        std::unique_ptr<KVSetClient> kv_set_client;
        std::unique_ptr<KVDeleteClient> kv_delete_client;
    public:
        KV(
                std::string range_key,
                KVGetClient *kv_get_client,
                KVSetClient *kv_set_client,
                KVDeleteClient *kv_delete_client
        ) : range_key(range_key), kv_get_client(kv_get_client),
            kv_set_client(kv_set_client), kv_delete_client(kv_delete_client) {}


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

        Range(const std::string &name, Telem::TimeRange time_range);

        Range(const Key &key, const std::string &name, Telem::TimeRange time_range);

        Range(const Key &key, const std::string &name, Telem::TimeRange time_range, KV *kv);
    };

    class Client {
    private:
        RetrieveClient *retrieve_client;
        CreateClient *create_client;
        KVGetClient *kv_get_client;
        KVSetClient *kv_set_client;
        KVDeleteClient *kv_delete_client;
    public:
        Client(RetrieveClient *retrieve_client, CreateClient *create_client, KVGetClient *kv_get_client,
               KVSetClient *kv_set_client, KVDeleteClient *kv_delete_client) :
                retrieve_client(retrieve_client),
                create_client(create_client),
                kv_get_client(kv_get_client),
                kv_set_client(kv_set_client),
                kv_delete_client(kv_delete_client) {}


        Range retrieve_by_key(std::string key);

        Range retrieve_by_name(std::string name);

        std::vector<Range> retrieve_by_key(std::vector<std::string> keys);

        std::vector<Range> retrieve_by_name(std::vector<std::string> names);

        void create(std::vector<Range> &ranges);

        void create(Range &range);

        Range create(std::string name, Telem::TimeRange time_range);
    };

}