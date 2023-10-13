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
#include "v1/channel.pb.h"
#include <grpcpp/grpcpp.h>
#include "synnax/telem/telem.h"
#include "freighter/freighter.h"

#pragma once

using namespace Synnax;

using ChannelKey = std::uint32_t;

namespace Channel {
    typedef Freighter::UnaryClient<
            api::v1::ChannelRetrieveResponse,
            api::v1::ChannelRetrieveRequest,
            grpc::Status> RetrieveClient;

    typedef Freighter::UnaryClient<
            api::v1::ChannelCreateResponse,
            api::v1::ChannelCreateRequest,
            grpc::Status> CreateClient;

    class Channel {
    public:
        Telem::DataType dataType;
        std::string name;
        ChannelKey key;
        ChannelKey index;
        Telem::Rate rate;
        bool is_index;
        std::uint32_t leaseholder;

        Channel(
                std::string name,
                Telem::DataType dataType,
                Telem::Rate rate = Telem::Rate(0),
                bool is_index = false,
                std::uint32_t leaseholder = 0,
                ChannelKey index = 0,
                ChannelKey key = 0
        );

        Channel(const api::v1::Channel &ch);

        void to_proto(api::v1::Channel *ch) const;
    };

    class Client {
    public:
        Client(RetrieveClient *retrieve_client, CreateClient *create_client) :
                retrieve_client(retrieve_client),
                create_client(create_client) {}

        Channel retrieve(const std::string &name);

        Channel retrieve(std::uint32_t key);

        std::vector<Channel> retrieve(const std::vector<std::string> &names);

        std::vector<Channel> retrieve(const std::vector<std::uint32_t> &keys);

        void create(std::vector<Channel> &channels);

        Channel create(std::string name, Telem::DataType data_type, Telem::Rate rate, ChannelKey index, bool is_index);

    private:
        RetrieveClient *retrieve_client;
        CreateClient *create_client;
    };
}