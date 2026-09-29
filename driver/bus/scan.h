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
#include <map>
#include <memory>
#include <string>
#include <unordered_map>
#include <utility>
#include <vector>

#include "x/cpp/lib/lib.h"

#include "driver/bus/vendor.h"
#include "driver/common/scan_task.h"

namespace driver::bus {
/// @brief reports one device for each channel a card backend lists. A tracked device
/// whose backend, card, and channel match a listed channel keeps its key and
/// properties, so a user's role or speed survives a scan. A backend whose vendor
/// library is absent, or that the Driver cannot drive yet, lists nothing.
/// @tparam Backend has list(), which returns the Infos of its channels, each with a
/// card, channel, and name.
template<typename Backend>
class Scanner final : public common::Scanner {
public:
    /// @param task the scan task.
    /// @param integration the name of the integration, which prefixes device keys.
    /// @param make the make of the integration's devices.
    /// @param backends the backends to list, by name.
    Scanner(
        synnax::task::Task task,
        std::string integration,
        std::string make,
        std::unordered_map<std::string, std::shared_ptr<Backend>> backends
    ):
        task(std::move(task)),
        integration(std::move(integration)),
        make(std::move(make)) {
        for (auto &[name, backend]: backends)
            this->backends.emplace(name, std::move(backend));
    }

    [[nodiscard]] common::ScannerConfig config() const override {
        return {
            .make = this->make,
            .log_prefix = "[" + this->integration + ".scan_task] "
        };
    }

    std::pair<std::vector<synnax::device::Device>, x::errors::Error>
    scan(const common::ScannerContext &ctx) override {
        std::vector<synnax::device::Device> out;
        for (const auto &[name, backend]: this->backends) {
            auto [infos, err] = backend->list();
            if (err.matches(x::lib::LOAD_ERROR) ||
                err.matches(vendor::UNSUPPORTED_ERROR))
                continue;
            if (err) return {{}, err};
            for (const auto &info: infos)
                out.push_back(this->device(ctx, name, info));
        }
        return {std::move(out), x::errors::NIL};
    }

private:
    synnax::task::Task task;
    std::string integration;
    std::string make;
    std::map<std::string, std::shared_ptr<Backend>> backends;

    template<typename Info>
    synnax::device::Device device(
        const common::ScannerContext &ctx,
        const std::string &backend,
        const Info &info
    ) const {
        const x::json::json::object_t props{
            {"backend", backend},
            {"card", info.card},
            {"channel", info.channel},
        };
        synnax::device::Device dev{
            .key = this->make_key(backend, info),
            .rack = this->task.rack,
            .location = info.name,
            .make = this->make,
            .model = backend,
            .name = info.name,
            .properties = props,
        };
        if (ctx.devices != nullptr)
            for (const auto &[_, tracked]: *ctx.devices)
                if (matches(tracked.properties, props)) {
                    dev = tracked;
                    break;
                }
        dev.status = synnax::device::Status{
            .key = synnax::device::status_key(dev),
            .name = dev.name,
            .variant = synnax::status::VARIANT_SUCCESS,
            .message = "Channel available",
            .time = x::telem::TimeStamp::now(),
            .details = {.rack = this->task.rack, .device = dev.key},
        };
        return dev;
    }

    template<typename Info>
    std::string make_key(const std::string &backend, const Info &info) const {
        return this->integration + "_" + std::to_string(this->task.rack) + "_" +
               backend + "_" + std::to_string(info.card) + "_" +
               std::to_string(info.channel);
    }

    static bool
    matches(const x::json::json::object_t &tracked, const x::json::json::object_t &p) {
        return std::ranges::all_of(p, [&](const auto &kv) {
            const auto it = tracked.find(kv.first);
            return it != tracked.end() && it->second == kv.second;
        });
    }
};
}
