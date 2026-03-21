// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <functional>
#include <memory>
#include <string>
#include <utility>

#include "client/cpp/synnax.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/status/status.h"

#include "arc/cpp/runtime/node/factory.h"
#include "arc/cpp/runtime/node/node.h"
#include "arc/cpp/types/types.h"

namespace driver::arc::status {

/// @brief Callback for delivering status notifications to the cluster.
using Setter = std::function<x::errors::Error(x::status::Status<> &)>;

/// @brief Sets a status notification each time it is executed by the scheduler.
class SetStatus : public ::arc::runtime::node::Node {
    x::status::Status<> info;
    Setter setter;

public:
    SetStatus(x::status::Status<> info, Setter setter):
        info(std::move(info)), setter(std::move(setter)) {}

    x::errors::Error next(::arc::runtime::node::Context &ctx) override {
        this->info.time = x::telem::TimeStamp::now();
        auto err = this->setter(this->info);
        if (err) ctx.report_error(err);
        return x::errors::NIL;
    }

    [[nodiscard]] bool is_output_truthy(const std::string &) const override {
        return false;
    }
};

class Factory : public ::arc::runtime::node::Factory {
    std::shared_ptr<synnax::Synnax> client;

public:
    explicit Factory(std::shared_ptr<synnax::Synnax> client):
        client(std::move(client)) {}

    bool handles(const std::string &node_type) const override {
        return node_type == "set_status";
    }

    std::pair<std::unique_ptr<::arc::runtime::node::Node>, x::errors::Error>
    create(::arc::runtime::node::Config &&cfg) override {
        if (!this->handles(cfg.node.type)) return {nullptr, x::errors::NOT_FOUND};
        const auto get_str = [&](const std::string &key) -> std::string {
            const auto &p = cfg.node.config[key];
            auto sv = ::arc::types::to_sample_value(p.value, p.type);
            if (!sv.has_value()) return "";
            const auto *s = std::get_if<std::string>(&*sv);
            return s != nullptr ? *s : "";
        };
        x::status::Status<> info{
            .key = get_str("status_key"),
            .name = get_str("name"),
            .variant = get_str("variant"),
            .message = get_str("message"),
            .time = x::telem::TimeStamp::now(),
        };
        return {
            std::make_unique<SetStatus>(
                std::move(info),
                [c = this->client](x::status::Status<> &s) {
                    return c->statuses.set(s);
                }
            ),
            x::errors::NIL
        };
    }
};

}
