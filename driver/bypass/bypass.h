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
#include <atomic>
#include <condition_variable>
#include <deque>
#include <functional>
#include <memory>
#include <mutex>
#include <shared_mutex>
#include <unordered_set>
#include <vector>

#include "glog/logging.h"

#include "client/cpp/channel/channel.h"
#include "x/cpp/telem/frame.h"

namespace driver::bypass {
/// @brief a subscription to frames on a set of channel keys.
class Subscription {
    mutable std::mutex mu;
    std::condition_variable cv;
    std::deque<x::telem::Frame> queue;
    std::vector<synnax::channel::Key> keys;
    std::unordered_set<synnax::channel::Key> key_set;
    std::atomic<bool> closed{false};
    std::function<void()> on_push;

public:
    explicit Subscription(std::vector<synnax::channel::Key> keys):
        keys(keys), key_set(keys.begin(), keys.end()) {}

    Subscription(const Subscription &) = delete;
    Subscription &operator=(const Subscription &) = delete;

    /// @brief non-blocking pop. Returns true if a frame was available.
    bool try_pop(x::telem::Frame &frame) {
        std::lock_guard lock(this->mu);
        if (this->queue.empty()) return false;
        frame = std::move(this->queue.front());
        this->queue.pop_front();
        return true;
    }

    /// @brief blocking pop. Returns false if closed with no remaining frames.
    bool pop(x::telem::Frame &frame) {
        std::unique_lock lock(this->mu);
        this->cv.wait(lock, [this] {
            return !this->queue.empty() || this->closed.load();
        });
        if (this->queue.empty()) return false;
        frame = std::move(this->queue.front());
        this->queue.pop_front();
        return true;
    }

    void close() {
        this->closed.store(true);
        this->cv.notify_all();
    }

    const std::vector<synnax::channel::Key> &subscribed_keys() const {
        return this->keys;
    }

    /// @brief returns true if the queue has no pending frames.
    bool empty() const {
        std::lock_guard lock(this->mu);
        return this->queue.empty();
    }

    void set_on_push(std::function<void()> fn) { this->on_push = std::move(fn); }

    /// @brief returns true if any key in the frame matches this subscription.
    bool matches(const x::telem::Frame &frame) const {
        for (const auto &[key, _]: frame)
            if (this->key_set.contains(key)) return true;
        return false;
    }

    /// @brief filters and pushes a frame, keeping only channels in this
    /// subscription. Uses a fast path when all channels already match.
    void filter_and_push(const x::telem::Frame &frame) {
        bool all_match = true;
        for (const auto &[key, _]: frame) {
            if (!this->key_set.contains(key)) {
                all_match = false;
                break;
            }
        }
        if (all_match) {
            this->push(frame.shallow_copy());
            return;
        }
        x::telem::Frame filtered;
        for (const auto &[key, series]: frame)
            if (this->key_set.contains(key))
                filtered.emplace(key, series.shallow_copy());
        if (!filtered.empty()) this->push(std::move(filtered));
    }

    void push(x::telem::Frame frame) {
        {
            std::lock_guard lock(this->mu);
            this->queue.push_back(std::move(frame));
        }
        this->cv.notify_one();
        if (this->on_push) this->on_push();
    }
};

/// @brief process-wide frame router that delivers frames by channel key.
/// Subscriptions are tracked via weak_ptr so that destroying a subscription
/// automatically expires its route entries — no explicit unsubscribe required.
///
/// The Bus assigns monotonically increasing alignment to each series before
/// delivery, fulfilling the role that Cesium plays on the server path.
/// Channels must be registered via register_channels before publishing.
/// Registration locks; the publish hot path uses only atomic fetch_add.
class Bus {
    mutable std::shared_mutex mu;
    std::vector<std::weak_ptr<Subscription>> subscribers;

    std::mutex alignment_mu;
    std::unordered_map<synnax::channel::Key, std::unique_ptr<std::atomic<uint64_t>>>
        alignment_counters;

public:
    /// @brief registers channels for alignment tracking. Must be called before
    /// publishing frames containing these channels (typically at writer open).
    void register_channels(const std::vector<synnax::channel::Key> &keys) {
        std::lock_guard lock(this->alignment_mu);
        for (auto key: keys) {
            if (this->alignment_counters.find(key) == this->alignment_counters.end())
                this->alignment_counters.emplace(
                    key,
                    std::make_unique<std::atomic<uint64_t>>(0)
                );
        }
    }

    /// @brief publishes a frame to all subscribers with matching channel keys.
    /// Assigns monotonically increasing alignment to each series before delivery.
    void publish(const x::telem::Frame &frame) {
        auto aligned = frame.shallow_copy();
        for (size_t i = 0; i < aligned.size(); i++) {
            auto key = aligned.channels->at(i);
            auto it = this->alignment_counters.find(key);
            if (it != this->alignment_counters.end()) {
                auto &counter = *it->second;
                const auto samples = aligned.series->at(i).size();
                const auto base = counter.fetch_add(
                    samples > 0 ? samples : 1,
                    std::memory_order_relaxed
                );
                aligned.series->at(i).alignment = x::telem::Alignment(base);
            }
        }
        bool has_expired = false;
        {
            std::shared_lock lock(this->mu);
            for (auto &weak_sub: this->subscribers) {
                auto sub = weak_sub.lock();
                if (!sub) {
                    has_expired = true;
                    continue;
                }
                if (sub->matches(aligned)) {
                    VLOG(1) << "[bus] routing frame with " << aligned.size()
                            << " channels to subscription";
                    sub->filter_and_push(aligned);
                }
            }
        }
        if (has_expired) this->sweep_expired();
    }

    /// @brief creates a subscription for the given channel keys.
    std::shared_ptr<Subscription>
    subscribe(const std::vector<synnax::channel::Key> &keys) {
        auto sub = std::make_shared<Subscription>(keys);
        std::unique_lock lock(this->mu);
        this->subscribers.push_back(sub);
        VLOG(1) << "[bus] new subscription for " << keys.size() << " channels";
        return sub;
    }

    /// @brief eagerly removes a subscription from the routing table.
    void unsubscribe(const Subscription &sub) {
        std::unique_lock lock(this->mu);
        this->subscribers.erase(
            std::remove_if(
                this->subscribers.begin(),
                this->subscribers.end(),
                [&sub](const std::weak_ptr<Subscription> &w) {
                    auto locked = w.lock();
                    return !locked || locked.get() == &sub;
                }
            ),
            this->subscribers.end()
        );
    }

private:
    void sweep_expired() {
        std::unique_lock lock(this->mu);
        this->subscribers.erase(
            std::remove_if(
                this->subscribers.begin(),
                this->subscribers.end(),
                [](const std::weak_ptr<Subscription> &w) { return w.expired(); }
            ),
            this->subscribers.end()
        );
    }
};
}
