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
#include <variant>

#include "absl/log/log.h"

#include "client/cpp/status/status.h"
#include "client/cpp/synnax.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"

#include "arc/cpp/runtime/node/node.h"
#include "arc/cpp/runtime/state/state.h"
#include "arc/cpp/stl/stl.h"
#include "arc/cpp/stl/strings/state.h"
#include "arc/cpp/types/types.h"
#include "driver/arc/reporter.h"

namespace driver::arc::status {

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
        report(set_failure_msg(err.data));
        return "";
    }
    if (res.multiple_matches) report(set_multi_match_msg(key_or_name, res.key));
    return res.key;
}

/// @brief validates that the named param holds a string value.
inline x::errors::Error
validate_string(const ::arc::types::Params &params, const std::string &name) {
    const auto &p = params[name];
    // A var-bound param takes its value from the variable at fire time, so the
    // declared initial carries no contract.
    if (p.type.kind == ::arc::types::Kind::VarRef) return x::errors::NIL;
    const auto sv = ::arc::types::to_sample_value(p.value, p.type);
    if (sv.has_value() && std::holds_alternative<std::string>(*sv))
        return x::errors::NIL;
    return x::errors::Error(
        x::errors::VALIDATION,
        "status.set inputs: " + name + " must be a string"
    );
}

/// @brief Flow node for `status.set`. Calls dispatch_set on every trigger and
/// emits the resolved key on Output(0). Inputs are read at fire time so
/// var-bound params track the referenced variable's latest value.
class SetStatus : public ::arc::runtime::node::Node {
    ::arc::runtime::state::Node state;
    std::shared_ptr<synnax::Synnax> client;
    Reporter report;

public:
    SetStatus(
        ::arc::runtime::state::Node &&state,
        std::shared_ptr<synnax::Synnax> client,
        Reporter report
    ):
        state(std::move(state)), client(std::move(client)), report(std::move(report)) {}

    x::errors::Error next(::arc::runtime::node::Context &ctx) override {
        const std::string resolved_key = dispatch_set(
            this->client,
            this->report,
            this->state.string_input("key_or_name"),
            this->state.string_input("message"),
            this->state.string_input("variant")
        );
        *this->state.output(0) = x::telem::Series(resolved_key);
        *this->state.output_time(0) = x::telem::Series(x::telem::TimeStamp::now());
        ctx.mark_changed(0);
        return x::errors::NIL;
    }

    void reset() override { this->state.reset(); }

    [[nodiscard]] bool is_output_truthy(size_t output_idx) const override {
        return this->state.is_output_truthy(output_idx);
    }
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
        return node_type == "set";
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
    }

    std::pair<std::unique_ptr<::arc::runtime::node::Node>, x::errors::Error>
    create(::arc::runtime::node::Config &&cfg) override {
        if (!this->handles(cfg.node.type)) return {nullptr, x::errors::NOT_FOUND};
        for (const auto *name: {"key_or_name", "message", "variant"})
            if (const auto err = validate_string(cfg.node.inputs, name); err)
                return {nullptr, err};
        return {
            std::make_unique<SetStatus>(
                std::move(cfg.state),
                this->client,
                this->report
            ),
            x::errors::NIL
        };
    }
};

}
