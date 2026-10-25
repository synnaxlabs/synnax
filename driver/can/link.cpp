// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <utility>

#include "client/cpp/can/json.gen.h"

#include "driver/can/link.h"

namespace driver::can {
Link::Link(Opener open, x::telem::NowFunc now):
    open(std::move(open)), now(std::move(now)) {}

std::pair<Opened, x::errors::Error> Link::bus() {
    std::lock_guard lock(this->mu);
    if (this->current != nullptr)
        return {{.bus = this->current, .opens = this->opens}, x::errors::NIL};
    const auto now = this->now();
    if (this->failure && now < this->next_attempt) return {{}, this->failure};
    auto [bus, err] = this->open();
    if (err) {
        this->failure = err;
        this->next_attempt = now + this->wait;
        this->wait = std::min(this->wait * 2, MAX_REOPEN_WAIT);
        return {{}, err};
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
    return err;
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
            return std::make_shared<Link>([backends, props] {
                return open(*backends, props);
            });
        });
    };
}
}
