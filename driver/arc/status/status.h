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
#include "x/cpp/telem/telem.h"

#include "arc/cpp/runtime/node/node.h"
#include "arc/cpp/runtime/state/state.h"
#include "arc/cpp/stl/stl.h"
#include "arc/cpp/stl/strings/state.h"
#include "arc/cpp/types/types.h"

namespace driver::arc::status {

/// @brief Reporter surfaces an stdlib-originated failure as a task-level status.
/// Mirrors the Go-side taskreporter.Reporter so set/delete failures land as
/// visible task statuses (warnings) rather than silent log lines.
using Reporter = std::function<
    void(const std::string &variant, const std::string &message)>;

// Reporter message templates. Keep in sync with
// core/pkg/service/arc/status/status.go.
inline std::string set_failure_msg(const std::string &err) {
    return "status.set: " + err;
}
inline std::string
set_multi_match_msg(const std::string &key_or_name, const std::string &resolved_key) {
    return "status.set: multiple statuses named \"" + key_or_name +
           "\"; updated first match (" + resolved_key + ")";
}
inline std::string delete_failure_msg(const std::string &err) {
    return "status.delete: " + err;
}
inline std::string delete_not_found_msg(const std::string &key_or_name) {
    return "status.delete: no status found \"" + key_or_name + "\"";
}
inline std::string delete_multi_match_msg(const std::string &key_or_name, int count) {
    return "status.delete: multiple statuses named \"" + key_or_name +
           "\"; deleted all (" + std::to_string(count) + ")";
}

/// @brief Upserts a status via the core API and surfaces failures via report.
/// Returns the resolved key on success, "" on failure.
inline std::string dispatch_set(
    const std::shared_ptr<synnax::Synnax> &client,
    const Reporter &report,
    const std::string &key_or_name,
    const std::string &message,
    const std::string &variant
) {
    auto [res, err] = client->statuses
                          .set_by_key_or_name(key_or_name, message, variant);
    if (err) {
        LOG(ERROR) << "status.set failed: key_or_name=" << key_or_name
                   << " error=" << err.data;
        report(x::status::VARIANT_WARNING, set_failure_msg(err.data));
        return "";
    }
    if (res.multiple_matches)
        report(x::status::VARIANT_WARNING, set_multi_match_msg(key_or_name, res.key));
    return res.key;
}

/// @brief Deletes a status via the core API, surfacing failure, not-found, and
/// multi-match as warnings via report.
inline void dispatch_delete(
    const std::shared_ptr<synnax::Synnax> &client,
    const Reporter &report,
    const std::string &key_or_name
) {
    auto [count, err] = client->statuses.delete_by_key_or_name(key_or_name);
    if (err) {
        LOG(ERROR) << "status.delete failed: key_or_name=" << key_or_name
                   << " error=" << err.data;
        report(x::status::VARIANT_WARNING, delete_failure_msg(err.data));
        return;
    }
    if (count == 0) {
        report(x::status::VARIANT_WARNING, delete_not_found_msg(key_or_name));
        return;
    }
    if (count > 1)
        report(x::status::VARIANT_WARNING, delete_multi_match_msg(key_or_name, count));
}

/// @brief Flow node for `status.set`. Calls dispatch_set on every trigger and
/// emits the resolved key on Output(0).
class SetStatus : public ::arc::runtime::node::Node {
    ::arc::runtime::state::Node state;
    std::shared_ptr<synnax::Synnax> client;
    Reporter report;
    std::string key_or_name;
    std::string message;
    std::string variant;

public:
    SetStatus(
        ::arc::runtime::state::Node &&state,
        std::shared_ptr<synnax::Synnax> client,
        Reporter report,
        std::string key_or_name,
        std::string message,
        std::string variant
    ):
        state(std::move(state)),
        client(std::move(client)),
        report(std::move(report)),
        key_or_name(std::move(key_or_name)),
        message(std::move(message)),
        variant(std::move(variant)) {}

    x::errors::Error next(::arc::runtime::node::Context &ctx) override {
        const std::string resolved_key = dispatch_set(
            this->client,
            this->report,
            this->key_or_name,
            this->message,
            this->variant
        );
        *this->state.output(0) = x::telem::Series(resolved_key);
        *this->state.output_time(0) = x::telem::Series(x::telem::TimeStamp::now());
        ctx.mark_changed(0);
        return x::errors::NIL;
    }

    [[nodiscard]] bool is_output_truthy(size_t output_idx) const override {
        return this->state.is_output_truthy(output_idx);
    }
};

/// @brief Flow node for `status.delete`. Calls dispatch_delete on every trigger.
/// Produces no output: delete is fire-and-forget, mirroring every other delete.
class DeleteStatus : public ::arc::runtime::node::Node {
    std::shared_ptr<synnax::Synnax> client;
    Reporter report;
    std::string key_or_name;

public:
    DeleteStatus(
        std::shared_ptr<synnax::Synnax> client,
        Reporter report,
        std::string key_or_name
    ):
        client(std::move(client)),
        report(std::move(report)),
        key_or_name(std::move(key_or_name)) {}

    x::errors::Error next(::arc::runtime::node::Context & /*ctx*/) override {
        dispatch_delete(this->client, this->report, this->key_or_name);
        return x::errors::NIL;
    }

    [[nodiscard]] bool is_output_truthy(size_t) const override { return false; }
};

class Module : public ::arc::stl::Module, public ::arc::stl::strings::StateConsumer {
    std::shared_ptr<synnax::Synnax> client;
    Reporter report;
    std::shared_ptr<::arc::stl::strings::State> str_state;

public:
    Module(std::shared_ptr<synnax::Synnax> client, Reporter report):
        client(std::move(client)), report(std::move(report)) {}

    [[nodiscard]] std::string module_name() const override { return "status"; }

    bool handles(const std::string &node_type) const override {
        return node_type == "set" || node_type == "delete";
    }

    void set_str_state(std::shared_ptr<::arc::stl::strings::State> ss) override {
        this->str_state = std::move(ss);
    }

    void bind_to(wasmtime::Linker &linker, wasmtime::Store::Context) override {
        auto client = this->client;
        auto report = this->report;
        auto str_state = this->str_state;
        linker
            .func_wrap(
                "status",
                "set",
                [client, report, str_state](
                    uint32_t key_or_name_h,
                    uint32_t msg_h,
                    uint32_t variant_h
                ) -> uint32_t {
                    return str_state->create(dispatch_set(
                        client,
                        report,
                        str_state->get(key_or_name_h),
                        str_state->get(msg_h),
                        str_state->get(variant_h)
                    ));
                }
            )
            .unwrap();
        linker
            .func_wrap(
                "status",
                "delete",
                [client, report, str_state](uint32_t key_or_name_h) {
                    dispatch_delete(client, report, str_state->get(key_or_name_h));
                }
            )
            .unwrap();
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
        if (cfg.node.type == "set")
            return {
                std::make_unique<SetStatus>(
                    std::move(cfg.state),
                    this->client,
                    this->report,
                    get_str("key_or_name"),
                    get_str("message"),
                    get_str("variant")
                ),
                x::errors::NIL
            };
        return {
            std::make_unique<DeleteStatus>(
                this->client,
                this->report,
                get_str("key_or_name")
            ),
            x::errors::NIL
        };
    }
};

}
