// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <chrono>
#include <utility>

#include "client/cpp/can/json.gen.h"

#include "driver/can/link.h"

namespace driver::can {
x::telem::TimeStamp steady_now() {
    return x::telem::TimeStamp(
        std::chrono::duration_cast<std::chrono::nanoseconds>(
            std::chrono::steady_clock::now().time_since_epoch()
        )
            .count()
    );
}

Link::Link(std::string channel, Opener open, x::telem::NowFunc now):
    channel(std::move(channel)), open(std::move(open)), now(std::move(now)) {}

x::errors::Error Link::shown(const x::errors::Error &err) const {
    if (!err.matches(UNREACHABLE_ERROR)) return err;
    const auto verb = this->opens == 0 ? "waiting for " : "reconnecting to ";
    return {UNREACHABLE_ERROR, verb + this->channel + ": " + err.data};
}

std::pair<Opened, x::errors::Error> Link::bus() {
    std::lock_guard lock(this->mu);
    if (this->current != nullptr)
        return {{.bus = this->current, .opens = this->opens}, x::errors::NIL};
    const auto now = this->now();
    if (this->failure && now < this->next_attempt)
        return {{}, this->shown(this->failure)};
    auto [bus, err] = this->open();
    if (err) {
        this->failure = err;
        this->next_attempt = now + this->wait;
        this->wait = std::min(this->wait * 2, MAX_REOPEN_WAIT);
        return {{}, this->shown(err)};
    }
    this->current = std::move(bus);
    this->opens++;
    this->failure = x::errors::NIL;
    this->wait = MIN_REOPEN_WAIT;
    return {{.bus = this->current, .opens = this->opens}, x::errors::NIL};
}

x::errors::Error Link::report(const Opened &opened, const x::errors::Error &err) {
    if (!err.matches(UNREACHABLE_ERROR) && !err.matches(CRITICAL_HARDWARE_ERROR))
        return err;
    std::lock_guard lock(this->mu);
    if (this->current == opened.bus) this->current.reset();
    return this->shown(err);
}

Acquire acquirer(
    std::shared_ptr<Links> links,
    std::string key,
    std::shared_ptr<const Backends> backends,
    synnax::can::Properties props
) {
    auto settings = props.to_json();
    return [links = std::move(links),
            key = std::move(key),
            settings = std::move(settings),
            backends = std::move(backends),
            props = std::move(props)] {
        return links->acquire(key, settings, [&] {
            return std::make_shared<Link>(props.channel, [backends, props] {
                return open(*backends, props);
            });
        });
    };
}
}
