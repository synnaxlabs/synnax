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

namespace Freighter {
    template<typename response_t, typename request_t, typename err_t>
    class UnaryClient {
    public:
        virtual std::pair<response_t, err_t> send(const std::string &target, request_t &request) = 0;
    };

    template<typename response_t, typename request_t, typename err_t>
    class Stream {
    public:
        virtual err_t send(request_t &request) = 0;

        virtual std::pair<response_t, err_t> receive() = 0;

        virtual err_t closeSend() = 0;
    };

    template<typename response_t, typename request_t, typename err_t>
    class StreamClient {
    public:
        virtual Stream<response_t, request_t, err_t>* stream(const std::string &target) = 0;
    };
}
