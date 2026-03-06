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

#include "glog/logging.h"

#include "x/cpp/errors/errors.h"

#include "arc/cpp/runtime/node/factory.h"
#include "arc/cpp/runtime/node/node.h"

namespace arc::runtime::status {

/// @brief data for a status notification.
struct Info {
    std::string key;
    std::string name;
    std::string variant;
    std::string message;
};

/// @brief Callback for delivering status notifications to the cluster.
using Setter = std::function<x::errors::Error(const Info &)>;

/// @brief No-op setter used in tests or when notifications are unavailable.
inline Setter noop_setter = [](const Info &) { return x::errors::NIL; };

/// @brief Sets a status notification each time it is executed by the scheduler.
class SetStatus : public node::Node {
    Info info;
    Setter setter;

public:
    SetStatus(Info info, Setter setter):
        info(std::move(info)), setter(std::move(setter)) {}

    x::errors::Error next(node::Context & /*ctx*/) override {
        auto err = this->setter(this->info);
        if (err) LOG(ERROR) << "[arc] set_status: " << err.message();
        return x::errors::NIL;
    }

    [[nodiscard]] bool is_output_truthy(
        const std::string & /*param_name*/
    ) const override {
        return false;
    }
};

class Factory : public node::Factory {
    Setter setter;

public:
    explicit Factory(Setter setter = noop_setter): setter(std::move(setter)) {}

    bool handles(const std::string &node_type) const override {
        return node_type == "set_status";
    }

    std::pair<std::unique_ptr<node::Node>, x::errors::Error>
    create(node::Config &&cfg) override {
        if (!this->handles(cfg.node.type)) return {nullptr, x::errors::NOT_FOUND};
        Info info{
            .key = cfg.node.config["status_key"].get<std::string>(),
            .name = cfg.node.config["name"].get<std::string>(),
            .variant = cfg.node.config["variant"].get<std::string>(),
            .message = cfg.node.config["message"].get<std::string>(),
        };
        return {
            std::make_unique<SetStatus>(std::move(info), this->setter),
            x::errors::NIL
        };
    }
};

}
