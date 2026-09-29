// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "client/cpp/mil1553/json.gen.h"

#include "driver/mil1553/link.h"

namespace driver::mil1553 {
std::pair<Channel *, x::errors::Error> Link::opened() {
    if (this->channel == nullptr) {
        auto [ch, err] = this->backend->open(this->props);
        if (err) return {nullptr, err};
        this->channel = std::move(ch);
    }
    return {this->channel.get(), x::errors::NIL};
}

x::errors::Error Link::open() {
    std::lock_guard lock(this->mu);
    return this->opened().second;
}

std::pair<Transfer, x::errors::Error> Link::transact(
    const codec::mil1553::Command &command,
    const std::span<const std::uint16_t> words
) {
    std::lock_guard lock(this->mu);
    auto [ch, err] = this->opened();
    if (err) return {Transfer{}, err};
    auto res = ch->transact(command, words);
    if (res.second) this->channel.reset();
    return res;
}

std::pair<bus::Batch, x::errors::Error>
Link::read(const std::span<Transfer> out, const x::telem::TimeSpan timeout) {
    std::lock_guard lock(this->mu);
    auto [ch, err] = this->opened();
    if (err) return {bus::Batch{}, err};
    auto res = ch->read(out, timeout);
    if (res.second) this->channel.reset();
    return res;
}

x::errors::Error Link::respond(
    const std::uint8_t rt,
    const std::uint8_t subaddress,
    const std::span<const std::uint16_t> words
) {
    std::lock_guard lock(this->mu);
    auto [ch, err] = this->opened();
    if (err) return err;
    err = ch->respond(rt, subaddress, words);
    if (err) this->channel.reset();
    return err;
}

Acquire acquirer(
    std::shared_ptr<Links> links,
    std::string key,
    std::shared_ptr<Backend> backend,
    synnax::mil1553::Properties props
) {
    return [links = std::move(links),
            key = std::move(key),
            backend = std::move(backend),
            props = std::move(props)] {
        return links->acquire(key, props.to_json(), [&] {
            return std::make_shared<Link>(backend, props);
        });
    };
}
}
