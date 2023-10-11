// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// Local headers.

// std.
#include <string>
#include <utility>

/// @brief Interface for client.
template <typename response_t, typename request_t, typename stream_t, typename err_t, typename rpc_t>
class Client
{
public:
    /// @brief Interface for unary send.
    virtual std::pair<response_t, err_t> send(std::string target, request_t &request) = 0;

    /// @brief Interface for stream.
    virtual stream_t stream(std::string target) = 0;
};

/// @brief Interface for Streamer.
template <typename response_t, typename request_t, typename err_t, typename rpc_t>
class Streamer
{
public:
    /// @brief Interface for streamer send.
    virtual err_t send(request_t &request) = 0;

    /// @brief Interface for streamer receive.
    virtual std::pair<response_t, err_t> receive() = 0;

    /// @brief Closes the sending direction of the stream.
    virtual err_t closeSend() = 0;
};