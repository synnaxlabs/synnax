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

#include "client/cpp/channel/types.gen.h"
#include "x/cpp/telem/frame.h"

namespace driver::bypass {
/// @brief maximum number of frames buffered per subscription before the oldest frames
/// are dropped to make room for new ones.
constexpr size_t QUEUE_CAP = 2048;

/// @brief a subscription to frames on a set of channel keys.
class Subscription {
    mutable std::mutex mu;
    std::condition_variable cv;
    std::deque<x::telem::Frame> queue;
    std::vector<synnax::channel::Key> keys;
    std::unordered_set<synnax::channel::Key> key_set;
    std::atomic<bool> closed{false};
    std::function<void()> on_push;
    size_t cap;

public:
    explicit Subscription(
        std::vector<synnax::channel::Key> keys,
        size_t cap = QUEUE_CAP
    ):
        keys(keys), key_set(keys.begin(), keys.end()), cap(cap) {}

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

    void set_on_push(std::function<void()> fn) {
        std::lock_guard lock(this->mu);
        this->on_push = std::move(fn);
    }

    /// @brief filters, applies alignment, and pushes a frame. Returns true if any
    /// channel matched (frame was delivered).
    bool filter_and_push(
        const x::telem::Frame &frame,
        const std::vector<x::telem::Alignment> &alignments
    ) {
        bool all_match = true;
        for (const auto &[key, _]: frame) {
            if (!this->key_set.contains(key)) {
                all_match = false;
                break;
            }
        }
        if (all_match) {
            auto copy = frame.shallow_copy();
            for (size_t i = 0; i < copy.size(); i++)
                copy.series->at(i).alignment = alignments[i];
            this->push(std::move(copy));
            return true;
        }
        x::telem::Frame filtered;
        bool any_match = false;
        for (size_t i = 0; i < frame.size(); i++) {
            auto key = frame.channels->at(i);
            if (this->key_set.contains(key)) {
                auto s = frame.series->at(i).shallow_copy();
                s.alignment = alignments[i];
                filtered.emplace(key, std::move(s));
                any_match = true;
            }
        }
        if (!any_match) return false;
        this->push(std::move(filtered));
        return true;
    }

    void push(x::telem::Frame frame) {
        {
            std::lock_guard lock(this->mu);
            while (this->queue.size() >= this->cap)
                this->queue.pop_front();
            this->queue.push_back(std::move(frame));
            this->cv.notify_one();
            if (this->on_push) this->on_push();
        }
    }
};

/// @brief process-wide frame router that delivers frames by channel key. Subscriptions
/// are tracked via weak_ptr so that destroying a subscription automatically expires its
/// route entries — no explicit unsubscribe required.
///
/// The Bus assigns monotonically increasing alignment to each series before delivery,
/// fulfilling the role that Cesium plays on the server path. Channels must be
/// registered via register_channels before publishing. Registration locks; the publish
/// hot path uses only atomic fetch_add.
class Bus {
    mutable std::shared_mutex mu;
    std::vector<std::weak_ptr<Subscription>> subscribers;

    mutable std::shared_mutex alignment_mu;
    std::unordered_map<synnax::channel::Key, std::unique_ptr<std::atomic<uint64_t>>>
        alignment_counters;

public:
    /// @brief registers channels for alignment tracking. Must be called before
    /// publishing frames containing these channels (typically at writer open).
    void register_channels(const std::vector<synnax::channel::Key> &keys) {
        std::unique_lock lock(this->alignment_mu);
        for (auto key: keys) {
            if (!this->alignment_counters.contains(key))
                this->alignment_counters.emplace(
                    key,
                    std::make_unique<std::atomic<uint64_t>>(0)
                );
        }
    }

    /// @brief publishes a frame to all subscribers with matching channel keys.
    /// Alignment is computed lazily (only if a subscriber matches) and applied during
    /// the per-subscriber copy that filter_and_push already performs.
    void publish(const x::telem::Frame &frame) {
        thread_local std::vector<x::telem::Alignment> alignments;
        bool aligned = false;
        bool has_expired = false;
        {
            std::shared_lock lock(this->mu);
            for (auto &weak_sub: this->subscribers) {
                auto sub = weak_sub.lock();
                if (!sub) {
                    has_expired = true;
                    continue;
                }
                if (!aligned) {
                    std::shared_lock alm_lock(this->alignment_mu);
                    alignments.resize(frame.size());
                    for (size_t i = 0; i < frame.size(); i++) {
                        auto key = frame.channels->at(i);
                        auto it = this->alignment_counters.find(key);
                        if (it != this->alignment_counters.end()) {
                            auto &counter = *it->second;
                            const auto samples = frame.series->at(i).size();
                            alignments[i] = x::telem::Alignment(counter.fetch_add(
                                samples > 0 ? samples : 1,
                                std::memory_order_relaxed
                            ));
                        } else
                            alignments[i] = x::telem::Alignment(0);
                    }
                    aligned = true;
                }
                if (sub->filter_and_push(frame, alignments)) {
                    VLOG(1) << "[bus] routing frame with " << frame.size()
                            << " channels to subscription";
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
        std::erase_if(this->subscribers, [&sub](const std::weak_ptr<Subscription> &w) {
            auto locked = w.lock();
            return !locked || locked.get() == &sub;
        });
    }

private:
    void sweep_expired() {
        std::unique_lock lock(this->mu);
        this->subscribers.erase(
            std::ranges::remove_if(
                this->subscribers,
                [](const std::weak_ptr<Subscription> &w) { return w.expired(); }
            ).begin(),
            this->subscribers.end()
        );
    }
};
}
