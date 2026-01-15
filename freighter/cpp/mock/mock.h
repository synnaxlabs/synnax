// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <utility>

#include "freighter/cpp/freighter.h"

namespace freighter::mock {
template<typename RQ, typename RS>
class UnaryClient final : public freighter::UnaryClient<RQ, RS>, Finalizer<RQ, RS> {
public:
    std::vector<RQ> requests{};
    std::vector<RS> responses{};
    std::vector<x::errors::Error> response_errors{};

    UnaryClient() = default;

    UnaryClient(
        std::vector<RS> responses,
        std::vector<x::errors::Error> response_errors
    ):
        responses(responses), response_errors(std::move(response_errors)) {}

    UnaryClient(RS response, const x::errors::Error &response_error):
        responses({response}), response_errors({response_error}) {}

    void use(std::shared_ptr<Middleware> middleware) override { mw.use(middleware); }

    std::pair<RS, x::errors::Error>
    send(const std::string &target, RQ &request) override {
        requests.push_back(request);
        if (responses.empty())
            throw std::runtime_error("mock unary client has no responses left!");
        const auto ctx = Context("mock", x::url::URL(target), TransportVariant::STREAM);
        auto [res, err] = mw.exec(ctx, this, request);
        return {res, err};
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
};
}
