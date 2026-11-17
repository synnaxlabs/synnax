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
#include <mutex>
#include <string>
#include <unordered_map>
#include <utility>
#include <variant>

#include "absl/log/log.h"

#include "client/cpp/status/status.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"

#include "arc/cpp/runtime/node/node.h"
#include "arc/cpp/runtime/state/state.h"
#include "arc/cpp/stl/stl.h"
#include "arc/cpp/stl/strings/state.h"
#include "arc/cpp/types/types.h"
#include "driver/arc/reporter.h"
#include "driver/outbox/outbox.h"

namespace driver::arc::status {

// Reporter text. Keep in sync with core/pkg/service/arc/status/status.go.
const std::string SET_FAILURE_MSG = "Failed to set status";
const std::string MULTI_MATCH_MSG = "Found duplicate status names";
/// @brief returns the description for a set that matched more than one status.
inline std::string multi_match_description(
    const std::string &key_or_name,
    const std::string &resolved_key
) {
    return "Several statuses are named \"" + key_or_name +
           "\". Updated the first match, " + resolved_key +
           ". Give each status a unique name, or set it by key.";
}

/// @brief the shared state of the status module, which outlives a module while the
/// outbox holds its sets. Safe for concurrent use.
class Sets {
    /// @brief held while a report runs, which can wait on the Core.
    std::mutex report_mu;
    Reporter reporter;
    std::mutex keys_mu;
    std::unordered_map<std::string, std::string> keys;
    size_t refused_sets = 0;
    std::string refused_reason;

public:
    explicit Sets(Reporter reporter): reporter(std::move(reporter)) {}

    /// @brief reports through the reporter until close.
    void report(const std::string &message, const std::string &description) {
        std::lock_guard lock(this->report_mu);
        if (this->reporter != nullptr) this->reporter(message, description);
    }

    /// @returns the key of the last resolved set of key_or_name, or "" before the
    /// first.
    std::string key(const std::string &key_or_name) {
        std::lock_guard lock(this->keys_mu);
        const auto it = this->keys.find(key_or_name);
        return it == this->keys.end() ? "" : it->second;
    }

    /// @brief records key as the last resolved key of key_or_name.
    void resolve(const std::string &key_or_name, const std::string &key) {
        std::lock_guard lock(this->keys_mu);
        this->keys[key_or_name] = key;
    }

    /// @brief counts a set the outbox refused, without waiting on the Core.
    void refuse(const std::string &reason) {
        std::lock_guard lock(this->keys_mu);
        this->refused_sets++;
        this->refused_reason = reason;
    }

    /// @brief reports the sets refused since the last call, if any. It can wait on the
    /// Core, so only the outbox thread calls it.
    void report_refused() {
        size_t count = 0;
        std::string reason;
        {
            std::lock_guard lock(this->keys_mu);
            count = std::exchange(this->refused_sets, 0);
            reason = this->refused_reason;
        }
        if (count == 0) return;
        this->report(
            SET_FAILURE_MSG,
            reason + " Dropped " + std::to_string(count) + " sets."
        );
    }

    /// @brief stops reports, and waits for a report in progress, so the reporter can
    /// end.
    void close() {
        std::lock_guard lock(this->report_mu);
        this->reporter = nullptr;
    }
};

/// @brief queues a status set stamped with the current time, and reports a failure
/// through sets. Never waits for the Core.
/// @returns the key of the last resolved set of key_or_name, or "" before the first.
inline std::string queue_set(
    ::driver::outbox::Outbox &outbox,
    const std::shared_ptr<Sets> &sets,
    const std::string &key_or_name,
    const std::string &message,
    const std::string &variant
) {
    ::grpc::status::SetByKeyOrNameRequest req;
    req.set_key_or_name(key_or_name);
    req.set_message(message);
    req.set_variant(variant);
    req.set_time(x::telem::TimeStamp::now().nanoseconds());
    const auto err = outbox.set_status(
        req,
        [weak = std::weak_ptr(sets), key_or_name](
            const synnax::status::SetByKeyOrNameResult &res,
            const x::errors::Error &err
        ) {
            const auto sets = weak.lock();
            if (sets == nullptr) return;
            sets->report_refused();
            if (err) {
                LOG(ERROR) << "status.set failed: key_or_name=" << key_or_name
                           << " error=" << err.data;
                sets->report(SET_FAILURE_MSG, err.data);
                return;
            }
            if (res.multiple_matches)
                sets->report(
                    MULTI_MATCH_MSG,
                    multi_match_description(key_or_name, res.key)
                );
            sets->resolve(key_or_name, res.key);
        }
    );
    if (err) sets->refuse(err.data);
    return sets->key(key_or_name);
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

/// @brief Flow node for `status.set`. Queues a set on every trigger and emits the
/// last resolved key on Output(0). Inputs are read at fire time so
/// var-bound params track the referenced variable's latest value. Its output is truthy
/// once it queues a set, so a sequence step moves on before the set resolves.
class SetStatus : public ::arc::runtime::node::Node {
    ::arc::runtime::state::Node state;
    std::shared_ptr<::driver::outbox::Outbox> outbox;
    std::shared_ptr<Sets> sets;
    bool queued = false;

public:
    SetStatus(
        ::arc::runtime::state::Node &&state,
        std::shared_ptr<::driver::outbox::Outbox> outbox,
        std::shared_ptr<Sets> sets
    ):
        state(std::move(state)), outbox(std::move(outbox)), sets(std::move(sets)) {}

    x::errors::Error next(::arc::runtime::node::Context &ctx) override {
        const std::string resolved_key = queue_set(
            *this->outbox,
            this->sets,
            this->state.string_input("key_or_name"),
            this->state.string_input("message"),
            this->state.string_input("variant")
        );
        *this->state.output(0) = x::telem::Series(resolved_key);
        *this->state.output_time(0) = x::telem::Series(ctx.cycle.now);
        this->queued = true;
        this->state.emit(ctx.mark_changed, 0);
        return x::errors::NIL;
    }

    void reset(::arc::runtime::node::Context &) override {
        this->state.reset();
        this->queued = false;
    }

    [[nodiscard]] bool is_output_truthy(size_t) const override { return this->queued; }
};

class Module : public ::arc::stl::Module, public ::arc::stl::strings::StateConsumer {
    std::shared_ptr<::driver::outbox::Outbox> outbox;
    std::shared_ptr<Sets> sets;
    std::shared_ptr<::arc::stl::strings::State> str_state;

public:
    /// @param report called from the outbox thread too, until the module ends.
    Module(std::shared_ptr<::driver::outbox::Outbox> outbox, Reporter report):
        outbox(std::move(outbox)), sets(std::make_shared<Sets>(std::move(report))) {}

    Module(const Module &) = delete;
    Module &operator=(const Module &) = delete;

    ~Module() override { this->sets->close(); }

    [[nodiscard]] std::string module_name() const override { return "status"; }

    [[nodiscard]] bool handles(const std::string &node_type) const override {
        return node_type == "set";
    }

    void set_str_state(std::shared_ptr<::arc::stl::strings::State> ss) override {
        this->str_state = std::move(ss);
    }

    void bind_to(wasmtime::Linker &linker, wasmtime::Store::Context) override {
        auto outbox = this->outbox;
        auto sets = this->sets;
        auto str_state = this->str_state;
        linker
            .func_wrap(
                "status",
                "set",
                [outbox, sets, str_state](
                    uint32_t key_or_name_h,
                    uint32_t msg_h,
                    uint32_t variant_h
                ) -> uint32_t {
                    return str_state->create(queue_set(
                        *outbox,
                        sets,
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
            std::make_unique<SetStatus>(std::move(cfg.state), this->outbox, this->sets),
            x::errors::NIL
        };
    }
};

}
