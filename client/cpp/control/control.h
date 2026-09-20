// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <memory>
#include <string>
#include <utility>
#include <vector>

#include "client/cpp/channel/channel.h"
#include "freighter/cpp/freighter.h"
#include "x/cpp/control/control.h"
#include "x/cpp/errors/errors.h"

#include "core/pkg/transport/grpc/control/control.pb.h"

namespace synnax::control {

/// @brief the free virtual channel carrying control updates for the whole cluster.
const std::string CHANNEL_NAME = "sy_control";

/// @brief the control state of a single channel.
using State = x::control::State<synnax::channel::Key>;

/// @brief Type alias for the transport used to retrieve control state.
using RetrieveClient = freighter::
    UnaryClient<grpc::control::RetrieveRequest, grpc::control::RetrieveResponse>;

/// @brief Client for reading the control state of channels in a Synnax cluster.
class Client {
public:
    Client() = default;

    explicit Client(std::shared_ptr<RetrieveClient> retrieve_client):
        retrieve_client(std::move(retrieve_client)) {}

    /// @brief Retrieves the control state of the given channels.
    /// @param keys The channels to read the state of. An empty set reads every
    /// controlled channel in the cluster.
    /// @returns One state per controlled channel. Channels that no subject controls
    /// are absent from the result.
    [[nodiscard]] std::pair<std::vector<State>, x::errors::Error>
    retrieve(const std::vector<synnax::channel::Key> &keys = {}) const {
        grpc::control::RetrieveRequest req;
        req.mutable_keys()->Reserve(static_cast<int>(keys.size()));
        for (const auto key: keys)
            req.add_keys(key);
        auto [res, err] = this->retrieve_client->send("/control/retrieve", req);
        if (err) return {{}, err};
        std::vector<State> states;
        states.reserve(res.states_size());
        for (const auto &pb: res.states()) {
            auto [subject, subject_err] = x::control::Subject::from_proto(pb.subject());
            if (subject_err) return {{}, subject_err};
            states.push_back(
                State{
                    .subject = subject,
                    .resource = pb.resource(),
                    .authority = static_cast<x::control::Authority>(pb.authority()),
                }
            );
        }
        return {states, x::errors::NIL};
    }

private:
    std::shared_ptr<RetrieveClient> retrieve_client;
};

}
