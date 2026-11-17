// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <algorithm>
#include <cctype>
#include <cstdint>
#include <map>
#include <memory>
#include <mutex>
#include <optional>
#include <string>
#include <utility>
#include <variant>

#include "client/cpp/ranger/ranger.h"
#include "x/cpp/color/color.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/uuid/uuid.h"

#include "arc/cpp/runtime/node/node.h"
#include "arc/cpp/runtime/state/state.h"
#include "arc/cpp/stl/stl.h"
#include "arc/cpp/stl/strings/state.h"
#include "arc/cpp/types/types.h"
#include "driver/arc/reporter.h"
#include "driver/outbox/outbox.h"

namespace driver::arc::ranges {
// Reporter text. Keep in sync with core/pkg/service/arc/ranges/ranges.go.
const std::string CREATE_FAILURE_MSG = "Failed to create range";
const std::string END_FAILURE_MSG = "Failed to end range";

/// @brief the shared state of the ranges module, which outlives a module while the
/// outbox holds its calls. Safe for concurrent use.
class Calls {
    /// @brief held while a report runs, which can wait on the Core.
    std::mutex report_mu;
    Reporter reporter;
    /// @brief the reports of refused calls, as the count and last reason of each
    /// message.
    struct Refused {
        size_t count = 0;
        std::string reason;
    };
    std::mutex refused_mu;
    std::map<std::string, Refused> refused;

public:
    explicit Calls(Reporter reporter): reporter(std::move(reporter)) {}

    /// @brief reports through the reporter until close.
    void report(const std::string &message, const std::string &description) {
        std::lock_guard lock(this->report_mu);
        if (this->reporter != nullptr) this->reporter(message, description);
    }

    /// @brief counts a call the outbox refused, to report later as message, without
    /// waiting on the Core.
    void refuse(const std::string &message, const std::string &reason) {
        std::lock_guard lock(this->refused_mu);
        auto &r = this->refused[message];
        r.count++;
        r.reason = reason;
    }

    /// @brief reports the calls refused since the last call, if any. It can wait on
    /// the Core, so only the outbox thread calls it.
    void report_refused() {
        std::map<std::string, Refused> reports;
        {
            std::lock_guard lock(this->refused_mu);
            std::swap(reports, this->refused);
        }
        for (const auto &[message, r]: reports)
            this->report(
                message,
                r.reason + " Dropped " + std::to_string(r.count) + " calls."
            );
    }

    /// @brief stops reports, and waits for a report in progress, so the reporter can
    /// end.
    void close() {
        std::lock_guard lock(this->report_mu);
        this->reporter = nullptr;
    }
};

/// @brief returns a callback that reports a failed call as message.
inline ::driver::outbox::RangeDone
on_done(const std::shared_ptr<Calls> &calls, const std::string &message) {
    return [weak = std::weak_ptr(calls), message](const x::errors::Error &err) {
        const auto calls = weak.lock();
        if (calls == nullptr) return;
        calls->report_refused();
        if (err) calls->report(message, err.data);
    };
}

/// @brief queues the create of an open range that starts now, and reports failures
/// as warnings. Never waits for the Core.
/// @returns the key of the new range, or "" when the name, color, or parent is not
/// valid or the outbox refuses the create.
inline std::string queue_create(
    ::driver::outbox::Outbox &outbox,
    const std::shared_ptr<Calls> &calls,
    const std::string &name,
    const std::string &parent,
    const std::string &color_hex
) {
    if (std::ranges::all_of(name, [](const unsigned char ch) {
            return std::isspace(ch) != 0;
        })) {
        calls->report(CREATE_FAILURE_MSG, "The range name is empty.");
        return "";
    }
    std::optional<x::color::Color> c;
    if (!color_hex.empty()) {
        auto [parsed, err] = x::color::parse(color_hex);
        if (err) {
            calls->report(
                CREATE_FAILURE_MSG,
                "The color \"" + color_hex + "\" is not valid: " + err.data
            );
            return "";
        }
        c = parsed;
    }
    synnax::ranger::Range r;
    r.key = x::uuid::create();
    r.name = name;
    r.time_range = x::telem::TimeRange{
        x::telem::TimeStamp::now(),
        x::telem::TIME_STAMP_MAX
    };
    r.color = c;
    if (!parent.empty()) {
        auto [uid, err] = x::uuid::UUID::parse(parent);
        if (err) {
            calls->report(
                CREATE_FAILURE_MSG,
                "The parent key \"" + parent + "\" is not a valid range key."
            );
            return "";
        }
        r.parent = uid;
    }
    if (const auto err = outbox.create_range(r, on_done(calls, CREATE_FAILURE_MSG))) {
        calls->refuse(CREATE_FAILURE_MSG, err.data);
        return "";
    }
    return r.key.to_string();
}

/// @brief queues a call that sets the end of the keyed range to now, after the calls
/// queued before it. Never waits for the Core.
/// @returns the key, or "" when the key is not valid or the call fails to queue.
inline std::string queue_end(
    ::driver::outbox::Outbox &outbox,
    const std::shared_ptr<Calls> &calls,
    const std::string &key
) {
    auto [uid, parse_err] = x::uuid::UUID::parse(key);
    if (parse_err) {
        calls->report(
            END_FAILURE_MSG,
            "The key \"" + key + "\" is not a valid range key."
        );
        return "";
    }
    if (const auto err = outbox.end_range(
            uid,
            x::telem::TimeStamp::now(),
            on_done(calls, END_FAILURE_MSG)
        )) {
        calls->refuse(END_FAILURE_MSG, err.data);
        return "";
    }
    return uid.to_string();
}

/// @brief validates that the named param holds a string value.
inline x::errors::Error validate_string(
    const ::arc::types::Params &params,
    const std::string &scope,
    const std::string &name
) {
    const auto &p = params[name];
    // A var-bound param takes its value from the variable at fire time, so the
    // declared initial carries no contract.
    if (p.type.kind == ::arc::types::Kind::VarRef) return x::errors::NIL;
    const auto sv = ::arc::types::to_sample_value(p.value, p.type);
    if (sv.has_value() && std::holds_alternative<std::string>(*sv))
        return x::errors::NIL;
    return x::errors::Error(
        x::errors::VALIDATION,
        scope + " inputs: " + name + " must be a string"
    );
}

/// @brief Flow node for `ranges.create`. Queues an open range on every trigger and
/// emits the new range key on Output(0). Inputs are read at fire time so var-bound
/// params track the referenced variable's latest value.
class CreateRange : public ::arc::runtime::node::Node {
    ::arc::runtime::state::Node state;
    std::shared_ptr<::driver::outbox::Outbox> outbox;
    std::shared_ptr<Calls> calls;

public:
    CreateRange(
        ::arc::runtime::state::Node &&state,
        std::shared_ptr<::driver::outbox::Outbox> outbox,
        std::shared_ptr<Calls> calls
    ):
        state(std::move(state)), outbox(std::move(outbox)), calls(std::move(calls)) {}

    x::errors::Error next(::arc::runtime::node::Context &ctx) override {
        const std::string key = queue_create(
            *this->outbox,
            this->calls,
            this->state.string_input("name"),
            this->state.string_input("parent"),
            this->state.string_input("color")
        );
        *this->state.output(0) = x::telem::Series(key);
        *this->state.output_time(0) = x::telem::Series(ctx.cycle.now);
        this->state.emit(ctx.mark_changed, 0);
        return x::errors::NIL;
    }

    void reset(::arc::runtime::node::Context &) override { this->state.reset(); }

    [[nodiscard]] bool is_output_truthy(size_t output_idx) const override {
        return this->state.is_output_truthy(output_idx);
    }
};

/// @brief Flow node for `ranges.end`. Queues an end of the keyed range at now on
/// every trigger, emitting the key on Output(0). The key is read at fire time so a
/// var-bound param tracks the referenced variable's latest value.
class EndRange : public ::arc::runtime::node::Node {
    ::arc::runtime::state::Node state;
    std::shared_ptr<::driver::outbox::Outbox> outbox;
    std::shared_ptr<Calls> calls;

public:
    EndRange(
        ::arc::runtime::state::Node &&state,
        std::shared_ptr<::driver::outbox::Outbox> outbox,
        std::shared_ptr<Calls> calls
    ):
        state(std::move(state)), outbox(std::move(outbox)), calls(std::move(calls)) {}

    x::errors::Error next(::arc::runtime::node::Context &ctx) override {
        const std::string out = queue_end(
            *this->outbox,
            this->calls,
            this->state.string_input("key")
        );
        *this->state.output(0) = x::telem::Series(out);
        *this->state.output_time(0) = x::telem::Series(ctx.cycle.now);
        this->state.emit(ctx.mark_changed, 0);
        return x::errors::NIL;
    }

    void reset(::arc::runtime::node::Context &) override { this->state.reset(); }

    [[nodiscard]] bool is_output_truthy(size_t output_idx) const override {
        return this->state.is_output_truthy(output_idx);
    }
};

class Module : public ::arc::stl::Module, public ::arc::stl::strings::StateConsumer {
    std::shared_ptr<::driver::outbox::Outbox> outbox;
    std::shared_ptr<Calls> calls;
    std::shared_ptr<::arc::stl::strings::State> str_state;

public:
    /// @param report called from the outbox thread too, until the module ends.
    Module(std::shared_ptr<::driver::outbox::Outbox> outbox, Reporter report):
        outbox(std::move(outbox)), calls(std::make_shared<Calls>(std::move(report))) {}

    Module(const Module &) = delete;
    Module &operator=(const Module &) = delete;

    ~Module() override { this->calls->close(); }

    [[nodiscard]] std::string module_name() const override { return "ranges"; }

    [[nodiscard]] bool handles(const std::string &node_type) const override {
        return node_type == "create" || node_type == "end";
    }

    void set_str_state(std::shared_ptr<::arc::stl::strings::State> ss) override {
        this->str_state = std::move(ss);
    }

    void bind_to(wasmtime::Linker &linker, wasmtime::Store::Context) override {
        auto outbox = this->outbox;
        auto calls = this->calls;
        auto str_state = this->str_state;
        linker
            .func_wrap(
                "ranges",
                "create",
                [outbox, calls, str_state](
                    uint32_t name_h,
                    uint32_t parent_h,
                    uint32_t color_h
                ) -> uint32_t {
                    return str_state->create(queue_create(
                        *outbox,
                        calls,
                        str_state->get(name_h),
                        str_state->get(parent_h),
                        str_state->get(color_h)
                    ));
                }
            )
            .unwrap();
        linker
            .func_wrap(
                "ranges",
                "end",
                [outbox, calls, str_state](uint32_t key_h) -> uint32_t {
                    return str_state->create(
                        queue_end(*outbox, calls, str_state->get(key_h))
                    );
                }
            )
            .unwrap();
    }

    std::pair<std::unique_ptr<::arc::runtime::node::Node>, x::errors::Error>
    create(::arc::runtime::node::Config &&cfg) override {
        if (!this->handles(cfg.node.type)) return {nullptr, x::errors::NOT_FOUND};
        if (cfg.node.type == "create") {
            for (const auto *name: {"name", "parent", "color"})
                if (const auto
                        err = validate_string(cfg.node.inputs, "ranges.create", name);
                    err)
                    return {nullptr, err};
            return {
                std::make_unique<CreateRange>(
                    std::move(cfg.state),
                    this->outbox,
                    this->calls
                ),
                x::errors::NIL
            };
        }
        if (const auto err = validate_string(cfg.node.inputs, "ranges.end", "key"); err)
            return {nullptr, err};
        return {
            std::make_unique<EndRange>(std::move(cfg.state), this->outbox, this->calls),
            x::errors::NIL
        };
    }
};

}
