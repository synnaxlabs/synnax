// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// std.
#include <string>
#include <vector>
#include <memory>

/// freighter
#include "freighter/freighter.h"

/// api protos
#include "google/protobuf/empty.pb.h"
#include "v1/ranger.pb.h"

/// grpc
#include <grpcpp/grpcpp.h>

/// internal
#include "synnax/telem/telem.h"


typedef std::string Key;

using namespace Synnax;

namespace Synnax::Ranger {
typedef Freighter::UnaryClient<
        api::v1::RangeRetrieveResponse,
        api::v1::RangeRetrieveRequest
> RetrieveClient;

typedef Freighter::UnaryClient<
        api::v1::RangeCreateResponse,
        api::v1::RangeCreateRequest
> CreateClient;


typedef Freighter::UnaryClient<
        api::v1::RangeKVGetResponse,
        api::v1::RangeKVGetRequest
> KVGetClient;

typedef Freighter::UnaryClient<
        google::protobuf::Empty,
        api::v1::RangeKVSetRequest
> KVSetClient;

typedef Freighter::UnaryClient<
        google::protobuf::Empty,
        api::v1::RangeKVDeleteRequest
> KVDeleteClient;


class KV {
private:
    std::string range_key;
    KVGetClient *kv_get_client;
    KVSetClient *kv_set_client;
    KVDeleteClient *kv_delete_client;
public:
    KV(
            std::string range_key,
            KVGetClient *kv_get_client,
            KVSetClient *kv_set_client,
            KVDeleteClient *kv_delete_client
    ) : range_key(range_key), kv_get_client(kv_get_client),
        kv_set_client(kv_set_client), kv_delete_client(kv_delete_client) {}


    [[nodiscard]] std::pair<std::string, Freighter::Error> get(const std::string &key) const;

    [[nodiscard]] Freighter::Error set(const std::string &key, const std::string &value) const;

    [[nodiscard]] Freighter::Error delete_(const std::string &key) const;
};

/// @brief a range is a user-defined region of a cluster's data. It's identified
/// by a name, time range, and uniquely generated. See
/// https://docs.synnaxlabs.com/concepts/read-ranges for an introduction to ranges
/// and how they work.
class Range {
public:
    Key key;
    std::string name;
    Telem::TimeRange time_range{};
    KV *kv = nullptr;

    /// @brief constructs the range. Note that this does not mean the range has been
    /// persisted to the cluster. To persist the range, call create, at which
    /// point a unique key will be generated for the range.
    /// @param name - a human-readable name for the range. Does not need to be
    /// unique, and should represent the data that the range contains i.e.
    /// "Hot fire 1", "Print 22", or "Tank Burst Test".
    /// @param time_range - the time range of the range.
    Range(const std::string &name, Telem::TimeRange time_range);

    /// @brief constructs the range from its protobuf type.
    explicit Range(const api::v1::Range &rng);

private:
    /// @brief binds the range's fields to the given proto.
    void to_proto(api::v1::Range *rng) const;

    // @brief constructs an empty, invalid range.
    Range() = default;

    friend class Client;
};

class Client {

public:
    Client(RetrieveClient *retrieve_client, CreateClient *create_client, KVGetClient *kv_get_client,
           KVSetClient *kv_set_client, KVDeleteClient *kv_delete_client) :
            retrieve_client(retrieve_client),
            create_client(create_client),
            kv_get_client(kv_get_client),
            kv_set_client(kv_set_client),
            kv_delete_client(kv_delete_client) {}


    [[nodiscard]] std::pair<Range, Freighter::Error> retrieveByKey(const std::string &key) const;

    [[nodiscard]] std::pair<Range, Freighter::Error> retrieveByName(const std::string &name) const;

    [[nodiscard]] std::pair<std::vector<Range>, Freighter::Error> retrieveByKey(std::vector<std::string> keys) const;

    [[nodiscard]] std::pair<std::vector<Range>, Freighter::Error>
    retrieveByName(std::vector<std::string> names) const;

    [[nodiscard]] Freighter::Error create(std::vector<Range> &ranges) const;

    [[nodiscard]] Freighter::Error create(Range &range) const;

    [[nodiscard]] std::pair<Range, Freighter::Error> create(std::string name, Telem::TimeRange time_range) const;

private:
    RetrieveClient *retrieve_client;
    CreateClient *create_client;
    KVGetClient *kv_get_client;
    KVSetClient *kv_set_client;
    KVDeleteClient *kv_delete_client;

    std::pair<Range, Freighter::Error> retrieveOne(api::v1::RangeRetrieveRequest &req) const;

    std::pair<std::vector<Range>, Freighter::Error> retrieveMany(api::v1::RangeRetrieveRequest &req) const;

};

}