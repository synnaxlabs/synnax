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
#include <optional>
#include <string>
#include <unordered_map>
#include <unordered_set>
#include <utility>
#include <vector>

#include "absl/log/log.h"

#include "client/cpp/rack/key.h"
#include "client/cpp/status/types.gen.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"

namespace driver::common {
/// @brief how long a resource's status must be silent before a scan on another rack
/// takes the resource over.
const auto TAKEOVER_THRESHOLD = x::telem::SECOND * 30;

/// @brief stores the resources of one type and tells them apart. One implementation
/// exists per resource type, such as the NI device or the NI chassis.
template<typename D>
struct Store {
    virtual ~Store() = default;

    /// @brief returns the key that matches found hardware to a stored resource, such as
    /// the serial number when the hardware has one.
    [[nodiscard]] virtual std::string match_key(const D &resource) const = 0;

    /// @brief returns stored with the identity group of found, and the configuration
    /// group of stored.
    [[nodiscard]] virtual D merge_identity(D stored, const D &found) const = 0;

    /// @brief retrieves the resources on the rack, with their statuses.
    virtual std::pair<std::vector<D>, x::errors::Error>
    retrieve(synnax::rack::Key rack) = 0;

    /// @brief retrieves the stored resource that matches found, on any rack, with its
    /// status.
    /// @returns no resource when none matches.
    virtual std::pair<std::optional<D>, x::errors::Error> find(const D &found) = 0;

    /// @brief creates the resources, and sets the key of each one.
    virtual x::errors::Error create(std::vector<D> &resources) = 0;

    /// @brief writes the identity group of the resource, and keeps the configuration
    /// group as stored.
    virtual x::errors::Error update_identity(const D &resource) = 0;

    /// @brief writes the status of each resource.
    virtual x::errors::Error set_statuses(const std::vector<D> &resources) = 0;
};

/// @brief keeps the stored resources of one type in step with the hardware that a scan
/// finds on a rack. It creates new hardware with both field groups, writes only the
/// identity group after that, and marks hardware that a scan no longer finds as
/// disconnected. It never deletes. Sync is not safe for concurrent use.
template<typename D>
class Sync {
    std::unique_ptr<Store<D>> store;
    synnax::rack::Key rack;
    std::string log_prefix;
    /// @brief the tracked resources by match key.
    std::unordered_map<std::string, D> tracked;

    /// @brief sets the status of the resource to the given state.
    static void
    set_status(D &resource, const std::string &variant, const std::string &message) {
        if (!resource.status.has_value()) resource.status.emplace();
        resource.status->name = resource.name;
        resource.status->variant = variant;
        resource.status->message = message;
        resource.status->description = "";
        resource.status->time = x::telem::TimeStamp::now();
    }

    /// @brief returns whether a scan on this rack can move stored to this rack. A
    /// resource moves when another rack has not reported its status for a while.
    [[nodiscard]] bool can_take_over(const D &stored) const {
        if (stored.rack == this->rack) return true;
        if (!stored.status.has_value()) return true;
        return x::telem::TimeStamp::now() - stored.status->time > TAKEOVER_THRESHOLD;
    }

public:
    Sync(
        std::unique_ptr<Store<D>> store,
        const synnax::rack::Key rack,
        std::string log_prefix
    ):
        store(std::move(store)), rack(rack), log_prefix(std::move(log_prefix)) {}

    /// @brief loads the stored resources on the rack, so that the first scan does not
    /// look them up again.
    x::errors::Error init() {
        auto [resources, err] = this->store->retrieve(this->rack);
        if (err) return err;
        for (auto &resource: resources)
            this->tracked[this->store->match_key(resource)] = std::move(resource);
        return x::errors::NIL;
    }

    /// @brief saves the hardware that a scan found, and the status of every tracked
    /// resource.
    /// @param found the hardware that the scan found, with both field groups set to
    /// what a new resource holds.
    /// @returns the first error, after it tries every resource.
    x::errors::Error sync(std::vector<D> found) {
        x::errors::Error first_err;
        const auto keep = [&](const x::errors::Error &err) {
            if (err && !first_err) first_err = err;
        };
        std::vector<D> to_create;
        std::unordered_set<std::string> present;
        for (auto &res: found) {
            const auto match = this->store->match_key(res);
            if (!present.insert(match).second) continue;
            std::optional<D> stored;
            if (const auto it = this->tracked.find(match); it != this->tracked.end())
                stored = it->second;
            else {
                auto [got, err] = this->store->find(res);
                if (err) {
                    keep(err);
                    present.erase(match);
                    continue;
                }
                stored = std::move(got);
            }
            if (!stored.has_value()) {
                set_status(res, synnax::status::VARIANT_SUCCESS, "Device present");
                to_create.push_back(std::move(res));
                continue;
            }
            if (!this->can_take_over(*stored)) {
                present.erase(match);
                continue;
            }
            auto merged = this->store->merge_identity(*stored, res);
            if (merged.to_json() != stored->to_json()) {
                if (const auto err = this->store->update_identity(merged)) {
                    LOG(WARNING) << this->log_prefix << "failed to update "
                                 << merged.name << ": " << err;
                    keep(err);
                    merged = *stored;
                }
            }
            set_status(merged, synnax::status::VARIANT_SUCCESS, "Device present");
            this->tracked[match] = std::move(merged);
        }
        if (!to_create.empty()) {
            if (const auto err = this->store->create(to_create)) {
                LOG(WARNING) << this->log_prefix
                             << "failed to create resources: " << err;
                keep(err);
                for (const auto &res: to_create)
                    present.erase(this->store->match_key(res));
            } else
                for (auto &res: to_create)
                    this->tracked[this->store->match_key(res)] = std::move(res);
        }
        std::vector<D> statuses;
        statuses.reserve(this->tracked.size());
        for (auto &[match, res]: this->tracked) {
            if (!present.contains(match))
                set_status(res, synnax::status::VARIANT_WARNING, "Device disconnected");
            statuses.push_back(res);
        }
        if (!statuses.empty()) keep(this->store->set_statuses(statuses));
        return first_err;
    }

    /// @brief returns the tracked resource with the match key.
    [[nodiscard]] std::optional<D> get(const std::string &match_key) const {
        const auto it = this->tracked.find(match_key);
        if (it == this->tracked.end()) return std::nullopt;
        return it->second;
    }

    /// @brief tracks a resource that the Core reports as set, so that a scan keeps the
    /// configuration group that a user wrote. A resource on another rack drops from
    /// the tracked set.
    void on_set(const D &resource) {
        const auto match = this->store->match_key(resource);
        if (resource.rack != this->rack) {
            this->tracked.erase(match);
            return;
        }
        auto &tracked = this->tracked[match];
        const auto status = tracked.status;
        tracked = resource;
        if (!tracked.status.has_value()) tracked.status = status;
    }

    /// @brief stops tracking the resource with the given key.
    template<typename Key>
    void on_delete(const Key &key) {
        std::erase_if(this->tracked, [&](const auto &entry) {
            return entry.second.key == key;
        });
    }
};
}
