// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <utility>

#include "client/cpp/can/json.gen.h"

#include "driver/can/link.h"

namespace driver::can {
std::pair<std::shared_ptr<Bus>, x::errors::Error> Link::bus() {
    std::lock_guard lock(this->mu);
    if (this->current == nullptr) {
        auto [bus, err] = this->open();
        if (err) return {nullptr, err};
        this->current = std::move(bus);
    }
    return {this->current, x::errors::NIL};
}

void Link::close(const std::shared_ptr<Bus> &bus) {
    std::lock_guard lock(this->mu);
    if (this->current == bus) this->current.reset();
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
