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
#include <unordered_map>
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
    std::atomic<bool> closed{false};
    std::function<void()> on_push;

public:
    explicit Subscription(std::vector<synnax::channel::Key> keys):
        keys(std::move(keys)) {}

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
class Bus {
    mutable std::shared_mutex mu;
    std::unordered_map<synnax::channel::Key, std::vector<std::weak_ptr<Subscription>>>
        routes;

public:
    /// @brief publishes a frame to all subscribers with matching channel keys.
    void publish(const x::telem::Frame &frame) {
        bool has_expired = false;
        {
            std::shared_lock lock(this->mu);
            if (this->routes.empty()) return;
            static constexpr size_t INLINE_CAP = 4;
            Subscription *inline_buf[INLINE_CAP];
            size_t inline_count = 0;
            std::vector<Subscription *> overflow;
            auto already_delivered = [&](Subscription *p) {
                for (size_t j = 0; j < inline_count; j++)
                    if (inline_buf[j] == p) return true;
                for (auto *o: overflow)
                    if (o == p) return true;
                return false;
            };
            auto mark_delivered = [&](Subscription *p) {
                if (inline_count < INLINE_CAP)
                    inline_buf[inline_count++] = p;
                else
                    overflow.push_back(p);
            };
            for (const auto &[key, _]: frame) {
                auto it = this->routes.find(key);
                if (it == this->routes.end()) continue;
                for (auto &weak_sub: it->second) {
                    auto sub = weak_sub.lock();
                    if (!sub) {
                        has_expired = true;
                        continue;
                    }
                    if (!already_delivered(sub.get())) {
                        mark_delivered(sub.get());
                        VLOG(1) << "[bus] routing frame with " << frame.size()
                                << " channels to subscription";
                        sub->push(frame.shallow_copy());
                    }
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
        for (const auto key: keys)
            this->routes[key].push_back(sub);
        VLOG(1) << "[bus] new subscription for " << keys.size() << " channels";
        return sub;
    }

    /// @brief eagerly removes a subscription from the routing table.
    void unsubscribe(const Subscription &sub) {
        std::unique_lock lock(this->mu);
        for (const auto key: sub.subscribed_keys()) {
            auto it = this->routes.find(key);
            if (it == this->routes.end()) continue;
            auto &vec = it->second;
            vec.erase(
                std::remove_if(
                    vec.begin(),
                    vec.end(),
                    [&sub](const std::weak_ptr<Subscription> &w) {
                        auto locked = w.lock();
                        return !locked || locked.get() == &sub;
                    }
                ),
                vec.end()
            );
            if (vec.empty()) this->routes.erase(it);
        }
    }

private:
    void sweep_expired() {
        std::unique_lock lock(this->mu);
        for (auto it = this->routes.begin(); it != this->routes.end();) {
            auto &vec = it->second;
            vec.erase(
                std::remove_if(
                    vec.begin(),
                    vec.end(),
                    [](const std::weak_ptr<Subscription> &w) { return w.expired(); }
                ),
                vec.end()
            );
            if (vec.empty())
                it = this->routes.erase(it);
            else
                ++it;
        }
    }

public:
    /// @brief checks if any live subscribers exist for any of the given keys.
    bool has_subscribers(const std::vector<synnax::channel::Key> &keys) const {
        std::shared_lock lock(this->mu);
        for (const auto key: keys) {
            auto it = this->routes.find(key);
            if (it == this->routes.end()) continue;
            for (const auto &w: it->second)
                if (!w.expired()) return true;
        }
        return false;
    }
};
}
