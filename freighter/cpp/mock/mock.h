// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//
// Created by Emiliano Bonilla on 3/31/24.
//

#pragma once

#include "freighter/cpp/freighter.h"

template<typename RQ, typename RS>
class MockUnaryClient final : public freighter::UnaryClient<RQ, RS>,
                              freighter::Finalizer {
public:
    std::vector<RQ> requests{};
    std::vector<RS> responses{};
    std::vector<freighter::Error> response_errors{};


    MockUnaryClient(
        std::vector<RS> responses,
        std::vector<freighter::Error> response_errors
    ): responses(responses), response_errors(response_errors) {
    }

    MockUnaryClient(
        RS response,
        freighter::Error response_error
    ): responses({response}), response_errors({response_error}) {
    }

    void use(std::shared_ptr<freighter::Middleware> middleware) override { mw.use(middleware); }

    std::pair<RS, freighter::Error> send(const std::string &target, RQ &request) override {
        requests.push_back(request);
        if (responses.empty()) throw std::runtime_error("mock unary client has no responses left!");
        const auto ctx = freighter::Context("mock", target);
        auto [_, err] = mw.exec(ctx, this);
        auto res = responses.front();
        responses.erase(responses.begin());
        return {res, err};
    }

    std::pair<freighter::Context, freighter::Error> operator()(freighter::Context outboundContext) override {
        auto response_error = response_errors.front();
        response_errors.erase(response_errors.begin());
        return {outboundContext, response_error};
    }

private:
    freighter::MiddlewareCollector mw;
};
