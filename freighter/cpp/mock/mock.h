// Copyright 2025 Synnax Labs, Inc.
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

template<typename RQ, typename RS>
class MockUnaryClient final : public freighter::UnaryClient<RQ, RS>,
                              freighter::Finalizer<RQ, RS> {
public:
    std::vector<RQ> requests{};
    std::vector<RS> responses{};
    std::vector<xerrors::Error> response_errors{};

    MockUnaryClient() = default;

    MockUnaryClient(
        std::vector<RS> responses,
        std::vector<xerrors::Error> response_errors
    ):
        responses(responses), response_errors(std::move(response_errors)) {}

    MockUnaryClient(RS response, const xerrors::Error &response_error):
        responses({response}), response_errors({response_error}) {}

    void use(std::shared_ptr<freighter::Middleware> middleware) override {
        mw.use(middleware);
    }

    std::pair<RS, xerrors::Error>
    send(const std::string &target, RQ &request) override {
        requests.push_back(request);
        if (responses.empty())
            throw std::runtime_error("mock unary client has no responses left!");
        const auto ctx = freighter::Context(
            "mock",
            url::URL(target),
            freighter::TransportVariant::STREAM
        );
        auto [res, err] = mw.exec(ctx, this, request);
        return {res, err};
    }

    freighter::FinalizerReturn<RS>
    operator()(freighter::Context outboundContext, RQ &req) override {
        auto response_error = response_errors.front();
        response_errors.erase(response_errors.begin());
        auto res = responses.front();
        responses.erase(responses.begin());
        return {outboundContext, response_error, res};
    }

private:
    freighter::MiddlewareCollector<RQ, RS> mw;
};
