// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <map>
#include <string>
#include <thread>

#include "driver/arinc429/ddc/backend.h"

namespace driver::arinc429::ddc {
namespace {
constexpr std::uint32_t PARITY_BIT = 1u << 31;
/// @brief how often a read checks an empty receive queue.
const auto POLL_INTERVAL = x::telem::MILLISECOND;
/// @brief how long a write waits for room in a full transmit queue.
const auto WRITE_TIMEOUT = x::telem::SECOND;

x::errors::Error misuse(const std::string &message) {
    return x::errors::Error(errors::CRITICAL_HARDWARE_ERROR, message);
}

std::string name(const std::uint16_t card, const std::uint16_t channel) {
    return "DDC card " + std::to_string(card) + " channel " + std::to_string(channel);
}
}

codec::arinc429::Word from_card(const unsigned long raw) {
    const auto word = codec::arinc429::Word(static_cast<std::uint32_t>(raw))
                          .with_parity();
    if ((raw & PARITY_BIT) == 0) return word;
    return codec::arinc429::Word(word.raw() ^ PARITY_BIT);
}

unsigned long to_card(const codec::arinc429::Word word) {
    return word.raw() & ~PARITY_BIT;
}

struct Backend::Cards {
    /// @brief mu guards every field.
    std::mutex mu;
    /// @brief loaded is true once the SDK load was tried.
    bool loaded = false;
    std::shared_ptr<API> api;
    x::errors::Error load_err;
    /// @brief refs counts the open channels of each initialized card.
    std::map<std::uint16_t, std::size_t> refs;

    std::pair<std::shared_ptr<API>, x::errors::Error> load() {
        if (!this->loaded) {
            std::tie(this->api, this->load_err) = API::load();
            this->loaded = true;
        }
        return {this->api, this->load_err};
    }

    /// @brief initializes card when no channel has it open. InitCard resets every
    /// channel of a card, so a card is initialized once for all its channels. The
    /// caller holds mu for this and release.
    x::errors::Error acquire(const std::uint16_t card) {
        auto &refs = this->refs[card];
        if (refs == 0) {
            const auto c = static_cast<short>(card);
            if (auto err = this->api->error(this->api->init_card(c))) {
                this->refs.erase(card);
                return err;
            }
            if (auto err = this->api->error(
                    this->api->set_bit_format(c, sdk::BITFORMAT_ORIG)
                )) {
                this->api->free_card(c);
                this->refs.erase(card);
                return err;
            }
        }
        refs++;
        return x::errors::NIL;
    }

    /// @brief frees card when its last channel closes.
    void release(const std::uint16_t card) {
        if (--this->refs[card] > 0) return;
        this->refs.erase(card);
        this->api->free_card(static_cast<short>(card));
    }
};

namespace {
/// @brief Channel is one open receiver or transmitter on a DDC card.
class Channel final : public arinc429::Channel {
    std::shared_ptr<Backend::Cards> cards;
    std::shared_ptr<API> api;
    std::uint16_t card;
    short number;
    Direction direction;
    std::vector<unsigned long> buf;

public:
    Channel(
        std::shared_ptr<Backend::Cards> cards,
        std::shared_ptr<API> api,
        const std::uint16_t card,
        const short number,
        const Direction direction
    ):
        cards(std::move(cards)),
        api(std::move(api)),
        card(card),
        number(number),
        direction(direction),
        buf(sdk::MAX_LOAD) {}

    ~Channel() override {
        const auto c = static_cast<short>(this->card);
        if (this->direction == Direction::RECEIVE)
            this->api->enable_rx(c, this->number, sdk::DISABLE);
        else
            this->api->enable_tx(c, this->number, sdk::DISABLE);
        std::lock_guard lock(this->cards->mu);
        this->cards->release(this->card);
    }

    // TODO: map ERR_OVERFLOW to Batch::dropped once its value is known. The manual
    // names the code but not its number, so an overflow is a read error for now.
    std::pair<Batch, x::errors::Error>
    read(const std::span<Received> out, const x::telem::TimeSpan timeout) override {
        if (this->direction != Direction::RECEIVE)
            return {{}, misuse("cannot read from a transmit channel")};
        const auto n = static_cast<short>(
            std::min<std::size_t>(out.size(), sdk::MAX_LOAD)
        );
        const auto deadline = x::telem::TimeStamp::now() + timeout;
        while (true) {
            const auto r = this->api->read_rx_queue_irig_more(
                static_cast<short>(this->card),
                this->number,
                n,
                this->buf.data(),
                nullptr,
                nullptr
            );
            if (auto err = this->api->error(r)) return {{}, err};
            if (r > 0) {
                const auto now = x::telem::TimeStamp::now();
                for (short i = 0; i < r; i++)
                    out[i] = {.word = from_card(this->buf[i]), .time = now};
                return {{.count = static_cast<std::size_t>(r)}, x::errors::NIL};
            }
            if (x::telem::TimeStamp::now() >= deadline) return {{}, x::errors::NIL};
            std::this_thread::sleep_for(POLL_INTERVAL.chrono());
        }
    }

    x::errors::Error
    write(const std::span<const codec::arinc429::Word> words) override {
        if (this->direction != Direction::TRANSMIT)
            return misuse("cannot write to a receive channel");
        const auto deadline = x::telem::TimeStamp::now() + WRITE_TIMEOUT;
        std::size_t sent = 0;
        while (sent < words.size()) {
            const auto n = static_cast<short>(
                std::min<std::size_t>(words.size() - sent, sdk::MAX_LOAD)
            );
            for (short i = 0; i < n; i++)
                this->buf[i] = to_card(words[sent + i]);
            const auto r = this->api->load_tx_queue_more(
                static_cast<short>(this->card),
                this->number,
                n,
                this->buf.data()
            );
            if (auto err = this->api->error(r)) return err;
            sent += static_cast<std::size_t>(r);
            if (r == n) continue;
            if (x::telem::TimeStamp::now() >= deadline)
                return x::errors::Error(
                    errors::TEMPORARY_HARDWARE_ERROR,
                    "the transmit queue of " + name(this->card, this->number - 1) +
                        " stayed full"
                );
            std::this_thread::sleep_for(POLL_INTERVAL.chrono());
        }
        return x::errors::NIL;
    }
};

std::pair<short, x::errors::Error> speed(const std::string &speed) {
    if (speed == synnax::arinc429::SPEED_HIGH) return {sdk::HIGH_SPEED, x::errors::NIL};
    if (speed == synnax::arinc429::SPEED_LOW) return {sdk::LOW_SPEED, x::errors::NIL};
    return {
        0,
        x::errors::Error(
            errors::CONFIGURATION_ERROR,
            "unknown ARINC 429 speed " + speed
        ),
    };
}
}

Backend::Backend(): cards(std::make_shared<Cards>()) {}

std::pair<std::unique_ptr<arinc429::Channel>, x::errors::Error>
Backend::open(const synnax::arinc429::Properties &props, const Direction direction) {
    auto [spd, spd_err] = speed(props.speed);
    if (spd_err) return {nullptr, spd_err};
    std::lock_guard lock(this->cards->mu);
    auto [api, load_err] = this->cards->load();
    if (load_err) return {nullptr, load_err};
    if (auto err = this->cards->acquire(props.card)) return {nullptr, err};
    const auto c = static_cast<short>(props.card);
    const auto fail = [&](const x::errors::Error &err) {
        this->cards->release(props.card);
        return std::pair<std::unique_ptr<arinc429::Channel>, x::errors::Error>{
            nullptr,
            err,
        };
    };
    sdk::ChannelCount count{};
    if (auto err = api->error(api->get_channel_count(c, &count))) return fail(err);
    const bool rx = direction == Direction::RECEIVE;
    if (props.channel >= (rx ? count.rx : count.tx))
        return fail(
            x::errors::Error(
                errors::CONFIGURATION_ERROR,
                name(props.card, props.channel) + " has no " +
                    (rx ? "receiver" : "transmitter")
            )
        );
    const auto n = static_cast<short>(props.channel + 1);
    const auto configured = rx ? api->error(api->set_rx_channel_speed(c, n, spd))
                               : api->error(api->set_tx_speed(c, n, spd));
    if (configured) return fail(configured);
    const auto parity = rx ? api->set_rx_channel_parity(c, n, sdk::ODD_PARITY)
                           : api->set_tx_parity(c, n, sdk::ODD_PARITY);
    if (auto err = api->error(parity)) return fail(err);
    const auto enabled = rx ? api->enable_rx(c, n, sdk::ENABLE)
                            : api->enable_tx(c, n, sdk::ENABLE);
    if (auto err = api->error(enabled)) return fail(err);
    return {
        std::make_unique<Channel>(this->cards, api, props.card, n, direction),
        x::errors::NIL,
    };
}

std::pair<std::vector<Info>, x::errors::Error> Backend::list() {
    std::lock_guard lock(this->cards->mu);
    auto [api, load_err] = this->cards->load();
    if (load_err) return {{}, load_err};
    std::vector<Info> infos;
    for (std::uint16_t card = 0; card < MAX_CARDS; card++) {
        const bool open = this->cards->refs.contains(card);
        if (!open && this->cards->acquire(card)) continue;
        sdk::ChannelCount count{};
        const auto r = api->get_channel_count(static_cast<short>(card), &count);
        if (!open) this->cards->release(card);
        if (r != sdk::SUCCESS) continue;
        const auto channels = std::max(count.rx, count.tx);
        for (std::uint16_t ch = 0; ch < channels; ch++)
            infos.push_back({.card = card, .channel = ch, .name = name(card, ch)});
    }
    return {infos, x::errors::NIL};
}
}
