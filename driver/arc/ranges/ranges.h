// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstdint>
#include <functional>
#include <memory>
#include <optional>
#include <string>
#include <utility>
#include <variant>

#include "client/cpp/status/status.h"
#include "client/cpp/synnax.h"
#include "x/cpp/color/color.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/mem/indirect.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/uuid/uuid.h"

#include "arc/cpp/runtime/node/node.h"
#include "arc/cpp/runtime/state/state.h"
#include "arc/cpp/stl/stl.h"
#include "arc/cpp/stl/strings/state.h"
#include "arc/cpp/types/types.h"

namespace driver::arc::ranges {

/// @brief Reporter surfaces a stdlib-originated failure as a task-level status,
/// mirroring the Go-side taskreporter.Reporter so failures land as warnings.
using Reporter = std::function<
    void(const std::string &variant, const std::string &message)>;

/// @brief dispatch_create creates an open range that starts now, parsing the color
/// and parent and reporting failures as warnings. Returns the new key or "".
inline std::string dispatch_create(
    const std::shared_ptr<synnax::Synnax> &client,
    const Reporter &report,
    const std::string &name,
    const std::string &color_hex,
    const std::string &parent
) {
    std::optional<x::color::Color> c;
    if (!color_hex.empty()) {
        auto [parsed, err] = x::color::from_css(color_hex);
        if (err)
            report(
                synnax::status::VARIANT_WARNING,
                "ranges.create: invalid color \"" + color_hex + "\": " + err.message()
            );
        else
            c = parsed;
    }
    synnax::ranger::Range r;
    r.name = name;
    r.time_range = x::telem::TimeRange{
        x::telem::TimeStamp::now(),
        x::telem::TimeStamp::max()
    };
    r.color = c;
    if (!parent.empty()) {
        auto [uid, err] = x::uuid::UUID::parse(parent);
        if (err) {
            report(
                synnax::status::VARIANT_WARNING,
                "ranges.create: invalid parent key \"" + parent + "\""
            );
            return "";
        }
        synnax::ranger::Range parent_range;
        parent_range.key = uid;
        r.parent = x::mem::make_indirect<synnax::ranger::Range>(parent_range);
    }
    if (const auto err = client->ranges.create(r)) {
        report(synnax::status::VARIANT_WARNING, "ranges.create: " + err.message());
        return "";
    }
    return r.key.to_string();
}

/// @brief dispatch_end sets the end bound on the range identified by key to now
/// via retrieve-modify-upsert. Returns the key or "".
inline std::string dispatch_end(
    const std::shared_ptr<synnax::Synnax> &client,
    const Reporter &report,
    const std::string &key
) {
    auto [uid, parse_err] = x::uuid::UUID::parse(key);
    if (parse_err) {
        report(
            synnax::status::VARIANT_WARNING,
            "ranges.end: invalid range key \"" + key + "\""
        );
        return "";
    }
    auto [r, retrieve_err] = client->ranges.retrieve_by_key(uid);
    if (retrieve_err) {
        report(
            synnax::status::VARIANT_WARNING,
            "ranges.end: " + retrieve_err.message()
        );
        return "";
    }
    r.time_range.end = x::telem::TimeStamp::now();
    if (const auto err = client->ranges.create(r)) {
        report(synnax::status::VARIANT_WARNING, "ranges.end: " + err.message());
        return "";
    }
    return uid.to_string();
}

/// @brief Flow node for `ranges.create`. Creates an open range on every trigger
/// and emits the new range key on Output(0).
class CreateRange : public ::arc::runtime::node::Node {
    ::arc::runtime::state::Node state;
    std::shared_ptr<synnax::Synnax> client;
    Reporter report;
    std::string name;
    std::string color;
    std::string parent;

public:
    CreateRange(
        ::arc::runtime::state::Node &&state,
        std::shared_ptr<synnax::Synnax> client,
        Reporter report,
        std::string name,
        std::string color,
        std::string parent
    ):
        state(std::move(state)),
        client(std::move(client)),
        report(std::move(report)),
        name(std::move(name)),
        color(std::move(color)),
        parent(std::move(parent)) {}

    x::errors::Error next(::arc::runtime::node::Context &ctx) override {
        const std::string key = dispatch_create(
            this->client,
            this->report,
            this->name,
            this->color,
            this->parent
        );
        *this->state.output(0) = x::telem::Series(key);
        *this->state.output_time(0) = x::telem::Series(x::telem::TimeStamp::now());
        ctx.mark_changed(0);
        return x::errors::NIL;
    }

    [[nodiscard]] bool is_output_truthy(size_t output_idx) const override {
        return this->state.is_output_truthy(output_idx);
    }
};

/// @brief Flow node for `ranges.end`. Sets the end bound on the keyed range to now
/// on every trigger, emitting the key on Output(0).
class EndRange : public ::arc::runtime::node::Node {
    ::arc::runtime::state::Node state;
    std::shared_ptr<synnax::Synnax> client;
    Reporter report;
    std::string key;

public:
    EndRange(
        ::arc::runtime::state::Node &&state,
        std::shared_ptr<synnax::Synnax> client,
        Reporter report,
        std::string key
    ):
        state(std::move(state)),
        client(std::move(client)),
        report(std::move(report)),
        key(std::move(key)) {}

    x::errors::Error next(::arc::runtime::node::Context &ctx) override {
        const std::string out = dispatch_end(this->client, this->report, this->key);
        *this->state.output(0) = x::telem::Series(out);
        *this->state.output_time(0) = x::telem::Series(x::telem::TimeStamp::now());
        ctx.mark_changed(0);
        return x::errors::NIL;
    }

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

    [[nodiscard]] std::string module_name() const override { return "ranges"; }

    bool handles(const std::string &node_type) const override {
        return node_type == "create" || node_type == "end";
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
                "ranges",
                "create",
                [client, report, str_state](
                    uint32_t name_h,
                    uint32_t color_h,
                    uint32_t parent_h
                ) -> uint32_t {
                    return str_state->create(dispatch_create(
                        client,
                        report,
                        str_state->get(name_h),
                        str_state->get(color_h),
                        str_state->get(parent_h)
                    ));
                }
            )
            .unwrap();
        linker
            .func_wrap(
                "ranges",
                "end",
                [client, report, str_state](uint32_t key_h) -> uint32_t {
                    return str_state->create(
                        dispatch_end(client, report, str_state->get(key_h))
                    );
                }
            )
            .unwrap();
    }

    std::pair<std::unique_ptr<::arc::runtime::node::Node>, x::errors::Error>
    create(::arc::runtime::node::Config &&cfg) override {
        if (!this->handles(cfg.node.type)) return {nullptr, x::errors::NOT_FOUND};
        const auto get_str = [&](const std::string &key) -> std::string {
            const auto &p = cfg.node.inputs[key];
            auto sv = ::arc::types::to_sample_value(p.value, p.type);
            if (!sv.has_value()) return "";
            const auto *s = std::get_if<std::string>(&*sv);
            return s != nullptr ? *s : "";
        };
        if (cfg.node.type == "create")
            return {
                std::make_unique<CreateRange>(
                    std::move(cfg.state),
                    this->client,
                    this->report,
                    get_str("name"),
                    get_str("color"),
                    get_str("parent")
                ),
                x::errors::NIL
            };
        return {
            std::make_unique<EndRange>(
                std::move(cfg.state),
                this->client,
                this->report,
                get_str("key")
            ),
            x::errors::NIL
        };
    }
};

}
