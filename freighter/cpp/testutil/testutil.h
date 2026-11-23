// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <chrono>
#include <optional>
#include <stdexcept>
#include <string>
#include <utility>
#include <vector>

#include "freighter/cpp/freighter.h"

namespace freighter::testutil {
template<typename RQ, typename RS>
class UnaryClient final : public freighter::UnaryClient<RQ, RS>, Finalizer<RQ, RS> {
public:
    std::vector<RQ> requests{};
    std::vector<RS> responses{};
    std::vector<x::errors::Error> response_errors;

    UnaryClient() = default;

    UnaryClient(
        std::vector<RS> responses,
        std::vector<x::errors::Error> response_errors
    ):
        responses(responses), response_errors(std::move(response_errors)) {}

    UnaryClient(RS response, const x::errors::Error &response_error):
        responses({response}), response_errors({response_error}) {}

    void use(std::shared_ptr<Middleware> middleware) override { mw.use(middleware); }

    /// @brief the deadline of each request, or std::nullopt for a request with none.
    std::vector<std::optional<std::chrono::system_clock::time_point>> deadlines;

    std::pair<RS, x::errors::Error>
    send(const std::string &target, RQ &request) override {
        return this->exec(target, request, std::nullopt);
    }

    std::pair<RS, x::errors::Error> send(
        const std::string &target,
        RQ &request,
        const std::chrono::system_clock::time_point deadline
    ) override {
        return this->exec(target, request, deadline);
    }

    FinalizerReturn<RS> operator()(Context outboundContext, RQ &req) override {
        auto response_error = response_errors.front();
        response_errors.erase(response_errors.begin());
        auto res = responses.front();
        responses.erase(responses.begin());
        return {outboundContext, response_error, res};
    }

private:
    MiddlewareCollector<RQ, RS> mw;

    std::pair<RS, x::errors::Error> exec(
        const std::string &target,
        RQ &request,
        const std::optional<std::chrono::system_clock::time_point> deadline
    ) {
        requests.push_back(request);
        deadlines.push_back(deadline);
        if (responses.empty())
            throw std::runtime_error("mock unary client has no responses left!");
        auto ctx = Context("mock", x::url::URL(target), TransportVariant::STREAM);
        ctx.deadline = deadline;
        return mw.exec(ctx, this, request);
    }
};
}
