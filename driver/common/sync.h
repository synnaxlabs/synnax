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
#include <memory>
#include <mutex>
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

    /// @brief writes the status of each resource. It sets the owner and the details of
    /// each status from the resource and the status variant.
    virtual x::errors::Error set_statuses(const std::vector<D> &resources) = 0;
};

/// @brief keeps the stored resources of one type in step with the hardware that a scan
/// finds on a rack. It creates new hardware with both field groups, writes only the
/// identity group after that, takes hardware over from a rack that has gone silent,
/// and marks hardware that a scan no longer finds as disconnected. It never deletes.
/// Sync is safe for concurrent use, and it holds no lock while it calls the store.
template<typename D>
class Sync {
    std::unique_ptr<Store<D>> store;
    synnax::rack::Key rack;
    std::string log_prefix;
    std::mutex mu;
    /// @brief the tracked resources by match key.
    std::unordered_map<std::string, D> tracked;
    /// @brief counts the set and delete signals for each match key, so that a scan
    /// keeps a change that a signal made while the scan called the store.
    std::unordered_map<std::string, std::uint64_t> versions;

    /// @brief a found resource and what the scan decided for it.
    struct Result {
        std::string match;
        std::uint64_t version = 0;
        D resource;
    };

    /// @brief returns whether a scan on this rack can move stored to this rack. A
    /// resource moves when another rack has not reported its status for a while.
    [[nodiscard]] bool can_take_over(const D &stored) const {
        if (stored.rack == this->rack) return true;
        if (!stored.status.has_value()) return true;
        return x::telem::TimeStamp::now() - stored.status->time > TAKEOVER_THRESHOLD;
    }

    /// @brief gives present the status that the scanner reported for found, or a
    /// success when the scanner reported none.
    static void set_present(D &present, const D &found) {
        if (found.status.has_value())
            present.status = found.status;
        else {
            if (!present.status.has_value()) present.status.emplace();
            present.status->variant = synnax::status::VARIANT_SUCCESS;
            present.status->message = "Device present";
            present.status->description = "";
        }
        present.status->name = present.name;
        present.status->time = x::telem::TimeStamp::now();
    }

    /// @brief marks resource as disconnected, and keeps the time it was last seen.
    static void set_missing(D &resource) {
        if (!resource.status.has_value()) {
            resource.status.emplace();
            resource.status->time = x::telem::TimeStamp(0);
        }
        resource.status->name = resource.name;
        resource.status->variant = synnax::status::VARIANT_WARNING;
        resource.status->message = "Device disconnected";
        resource.status->description = "";
    }

    /// @brief stores the result of a scan. When a signal changed the resource while
    /// the scan called the store, it keeps the configuration group of the signal, and
    /// it skips a resource that a signal dropped. Requires mu.
    void apply(Result &res) {
        if (this->versions[res.match] == res.version) {
            this->tracked[res.match] = std::move(res.resource);
            return;
        }
        const auto it = this->tracked.find(res.match);
        if (it == this->tracked.end()) return;
        auto merged = this->store->merge_identity(it->second, res.resource);
        merged.status = std::move(res.resource.status);
        it->second = std::move(merged);
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
        std::lock_guard lock(this->mu);
        for (auto &resource: resources)
            this->tracked[this->store->match_key(resource)] = std::move(resource);
        return x::errors::NIL;
    }

    /// @brief saves the hardware that a scan found, and the status of every tracked
    /// resource.
    /// @param found the hardware that the scan found, with both field groups set to
    /// what a new resource holds, and the status the scanner reports, if any.
    /// @returns the first error, after it tries every resource.
    x::errors::Error sync(std::vector<D> found) {
        x::errors::Error first_err;
        const auto keep = [&](const x::errors::Error &err) {
            if (err && !first_err) first_err = err;
        };
        std::vector<Result> results;
        std::vector<std::optional<D>> snapshot;
        {
            std::lock_guard lock(this->mu);
            std::unordered_set<std::string> seen;
            for (auto &res: found) {
                auto match = this->store->match_key(res);
                if (!seen.insert(match).second) continue;
                const auto it = this->tracked.find(match);
                snapshot.push_back(
                    it == this->tracked.end() ? std::nullopt : std::optional(it->second)
                );
                const auto version = this->versions[match];
                results.push_back({
                    .match = std::move(match),
                    .version = version,
                    .resource = std::move(res),
                });
            }
        }
        std::vector<Result> updated;
        std::vector<Result> to_create;
        for (std::size_t i = 0; i < results.size(); i++) {
            auto &res = results[i];
            auto stored = std::move(snapshot[i]);
            if (!stored.has_value()) {
                auto [got, err] = this->store->find(res.resource);
                if (err) {
                    keep(err);
                    continue;
                }
                stored = std::move(got);
            }
            if (!stored.has_value()) {
                set_present(res.resource, res.resource);
                to_create.push_back(std::move(res));
                continue;
            }
            if (!this->can_take_over(*stored)) continue;
            auto merged = this->store->merge_identity(*stored, res.resource);
            if (merged.to_json() != stored->to_json()) {
                if (const auto err = this->store->update_identity(merged)) {
                    LOG(WARNING) << this->log_prefix << "failed to update "
                                 << merged.name << ": " << err;
                    keep(err);
                    // A resource that did not move stays with the other rack.
                    if (stored->rack != this->rack) continue;
                    merged = *stored;
                }
            }
            set_present(merged, res.resource);
            res.resource = std::move(merged);
            updated.push_back(std::move(res));
        }
        if (!to_create.empty()) {
            std::vector<D> created;
            created.reserve(to_create.size());
            for (const auto &res: to_create)
                created.push_back(res.resource);
            if (const auto err = this->store->create(created)) {
                LOG(WARNING) << this->log_prefix
                             << "failed to create resources: " << err;
                keep(err);
                to_create.clear();
            } else
                for (std::size_t i = 0; i < created.size(); i++)
                    to_create[i].resource = std::move(created[i]);
        }
        std::vector<D> statuses;
        {
            std::lock_guard lock(this->mu);
            std::unordered_set<std::string> present;
            for (auto &res: updated) {
                present.insert(res.match);
                this->apply(res);
            }
            for (auto &res: to_create) {
                present.insert(res.match);
                this->apply(res);
            }
            statuses.reserve(this->tracked.size());
            for (auto &[match, res]: this->tracked) {
                if (!present.contains(match)) set_missing(res);
                statuses.push_back(res);
            }
        }
        if (!statuses.empty()) keep(this->store->set_statuses(statuses));
        return first_err;
    }

    /// @brief returns the tracked resource with the match key.
    [[nodiscard]] std::optional<D> get(const std::string &match_key) {
        std::lock_guard lock(this->mu);
        const auto it = this->tracked.find(match_key);
        if (it == this->tracked.end()) return std::nullopt;
        return it->second;
    }

    /// @brief tracks a resource that the Core reports as set, so that a scan keeps the
    /// configuration group that a user wrote. It keeps the tracked status. A resource
    /// on another rack drops from the tracked set.
    void on_set(const D &resource) {
        const auto match = this->store->match_key(resource);
        std::lock_guard lock(this->mu);
        this->versions[match]++;
        if (resource.rack != this->rack) {
            this->tracked.erase(match);
            return;
        }
        auto &tracked = this->tracked[match];
        auto status = std::move(tracked.status);
        tracked = resource;
        if (status.has_value()) tracked.status = std::move(status);
    }

    /// @brief stops tracking the resource with the given key.
    template<typename Key>
    void on_delete(const Key &key) {
        std::lock_guard lock(this->mu);
        for (auto it = this->tracked.begin(); it != this->tracked.end(); ++it) {
            if (it->second.key != key) continue;
            this->versions[it->first]++;
            this->tracked.erase(it);
            return;
        }
    }
};
}
