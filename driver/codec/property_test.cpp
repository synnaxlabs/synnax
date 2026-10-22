// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <bit>
#include <cmath>
#include <optional>
#include <random>
#include <set>
#include <string>
#include <vector>

#include "gtest/gtest.h"

#include "client/cpp/library/types.gen.h"
#include "x/cpp/telem/series.h"
#include "x/cpp/test/test.h"
#include "x/cpp/uuid/uuid.h"

#include "driver/codec/errors.h"
#include "driver/codec/plan.h"

namespace driver::codec {
namespace {
namespace library = synnax::library;
using Bytes = std::vector<std::uint8_t>;

constexpr std::array SCALES = {0.5, 0.1, 0.001, 2.0, -1.0, -0.25, 10.0};
constexpr std::array OFFSETS = {0.0, -40.0, 1.5, 1000.0, -0.125};

/// @returns the payload bit positions a field covers, walking a big-endian field from
/// its most significant bit as DBC files number it.
std::vector<std::size_t>
positions(const std::size_t start, const std::size_t length, const bool big) {
    std::vector<std::size_t> out;
    std::size_t pos = start;
    for (std::size_t i = 0; i < length; i++) {
        out.push_back(big ? pos : start + i);
        if (big) pos = pos % 8 == 0 ? pos + 15 : pos - 1;
    }
    return out;
}

library::MessageEntry message(
    std::vector<library::BinaryField> fields,
    const std::optional<std::uint16_t> length
) {
    library::MessageEntry m;
    m.key = x::uuid::create();
    m.name = "message";
    m.payload = library::BinaryPayload{.length = length, .fields = std::move(fields)};
    return m;
}

/// @brief Layout is a random binary message whose fields cover disjoint bits, so that
/// every value survives an encode and decode.
class Layout {
public:
    library::MessageEntry entry;
    /// @brief parent is the index of each field's multiplexor in fields, or nullopt.
    std::vector<std::optional<std::size_t>> parent;

    explicit Layout(std::mt19937_64 &rng): rng(rng), length(rng() % 64 + 1) {
        if (const auto outer = this->add(this->integer(false, rng() % 7 + 2, true))) {
            this->muxes.push_back(*outer);
            auto inner = this->integer(false, rng() % 4 + 2, true);
            this->multiplex(inner, *outer);
            if (const auto i = this->add(inner, outer); i && rng() % 2 == 0)
                this->muxes.push_back(*i);
        }
        const auto count = rng() % 8 + 1;
        for (std::size_t i = 0; i < count; i++) {
            auto f = this->random();
            std::optional<std::size_t> by;
            if (!this->muxes.empty() && rng() % 3 == 0) {
                by = this->muxes[rng() % this->muxes.size()];
                this->multiplex(f, *by);
            }
            this->add(f, by);
        }
        this->entry = message(this->fields, static_cast<std::uint16_t>(this->length));
    }

    [[nodiscard]] const std::vector<library::BinaryField> &all() const {
        return this->fields;
    }

    /// @returns true when the field's multiplexors select it, given each field's raw
    /// value.
    [[nodiscard]] bool
    present(const std::size_t i, const std::vector<std::uint64_t> &raw) const {
        for (auto at = i; this->parent[at].has_value(); at = *this->parent[at]) {
            const auto &values = this->fields[at].multiplex_values;
            const auto mux = static_cast<std::int64_t>(raw[*this->parent[at]]);
            if (std::ranges::find(values, mux) == values.end()) return false;
        }
        return true;
    }

private:
    std::mt19937_64 &rng;
    /// @brief length is the message length in bytes.
    std::size_t length;
    std::vector<library::BinaryField> fields;
    /// @brief muxes holds the indices of the fields that may multiplex others.
    std::vector<std::size_t> muxes;
    /// @brief used holds the payload bits the fields cover.
    std::set<std::size_t> used;

    library::BinaryField
    integer(const bool signed_, const std::uint8_t bits, const bool exact) {
        library::BinaryField f;
        f.signed_ = signed_;
        f.bit_length = bits;
        if (!exact) {
            f.scale = SCALES[this->rng() % SCALES.size()];
            f.offset = OFFSETS[this->rng() % OFFSETS.size()];
        }
        return f;
    }

    library::BinaryField random() {
        switch (this->rng() % 6) {
            case 0: {
                library::BinaryField f;
                f.float_ = true;
                f.bit_length = this->rng() % 2 == 0 ? 32 : 64;
                return f;
            }
            case 1:
            case 2:
                return this->integer(this->rng() % 2 == 0, this->rng() % 64 + 1, true);
            default:
                return this->integer(this->rng() % 2 == 0, this->rng() % 32 + 1, false);
        }
    }

    void multiplex(library::BinaryField &f, const std::size_t mux) {
        const auto values = std::uint64_t{1} << this->fields[mux].bit_length;
        f.multiplex_values = {static_cast<std::int32_t>(this->rng() % values)};
        if (this->rng() % 2 == 0)
            f.multiplex_values.push_back(
                static_cast<std::int32_t>(this->rng() % values)
            );
        f.multiplexor = this->fields[mux].key;
    }

    /// @brief places the field at a random start where its bits are free.
    /// @returns the field's index, or nullopt when no place was found.
    std::optional<std::size_t>
    add(library::BinaryField f, const std::optional<std::size_t> by = std::nullopt) {
        const auto bits = this->length * 8;
        for (int attempt = 0; attempt < 32; attempt++) {
            const auto start = this->rng() % bits;
            const bool big = this->rng() % 2 == 0;
            const auto ps = positions(start, f.bit_length, big);
            const bool fits = std::ranges::all_of(ps, [&](const std::size_t p) {
                return p < bits && !this->used.contains(p);
            });
            if (!fits) continue;
            this->used.insert(ps.begin(), ps.end());
            f.key = x::uuid::create();
            f.name = "f" + std::to_string(this->fields.size());
            f.start_bit = static_cast<std::uint16_t>(start);
            f.byte_order = big ? library::BYTE_ORDER_BIG_ENDIAN
                               : library::BYTE_ORDER_LITTLE_ENDIAN;
            this->fields.push_back(std::move(f));
            this->parent.push_back(by);
            return this->fields.size() - 1;
        }
        return std::nullopt;
    }
};

/// @returns a random raw value of the field: its bit pattern for floats, otherwise
/// the integer, sign-extended for signed fields.
std::uint64_t random_raw(std::mt19937_64 &rng, const library::BinaryField &f) {
    if (f.float_) {
        while (true) {
            if (f.bit_length == 32) {
                const auto bits = static_cast<std::uint32_t>(rng());
                if (std::isfinite(std::bit_cast<float>(bits))) return bits;
            } else {
                const auto bits = rng();
                if (std::isfinite(std::bit_cast<double>(bits))) return bits;
            }
        }
    }
    const auto n = f.bit_length;
    const auto raw = n == 64 ? rng() : rng() & ((std::uint64_t{1} << n) - 1);
    if (!f.signed_ || n == 64) return raw;
    const auto shift = 64 - n;
    return static_cast<std::uint64_t>(static_cast<std::int64_t>(raw << shift) >> shift);
}

/// @returns the sample that holds the raw value in the field's physical units.
x::telem::SampleValue sample(const library::BinaryField &f, const std::uint64_t raw) {
    if (f.float_) {
        if (f.bit_length == 32)
            return static_cast<double>(
                std::bit_cast<float>(static_cast<std::uint32_t>(raw))
            );
        return std::bit_cast<double>(raw);
    }
    const bool exact = f.scale == 1 && f.offset == 0;
    if (exact && f.signed_) return static_cast<std::int64_t>(raw);
    if (exact) return raw;
    const auto integer = f.signed_ ? static_cast<double>(static_cast<std::int64_t>(raw))
                                   : static_cast<double>(raw);
    return integer * f.scale + f.offset;
}

library::TextField random_text_field(std::mt19937_64 &rng, const std::size_t i) {
    library::TextField field;
    if (rng() % 2 == 0)
        field = library::DelimitedTextField{.position = static_cast<std::uint32_t>(i)};
    else
        field = library::TaggedTextField{.tag = std::string(1, 'A' + i) + "="};
    std::visit(
        [&rng, i](auto &f) {
            f.key = x::uuid::create();
            f.name = "t" + std::to_string(i);
            if (rng() % 2 == 0) {
                f.scale = SCALES[rng() % SCALES.size()];
                f.offset = OFFSETS[rng() % OFFSETS.size()];
            }
        },
        field
    );
    return field;
}

const library::BaseField &base(const library::TextField &f) {
    return std::visit([](const auto &v) -> const library::BaseField & { return v; }, f);
}
}

/// @brief Decode must return an error or values for any layout the plan accepts and
/// any payload, never read out of bounds. Run it under ASan to catch overreads.
TEST(PlanProperty, DecodeHandlesAnyPayloadForAnyLayout) {
    std::mt19937_64 rng(4971);
    std::size_t compiled = 0;
    for (int layout = 0; layout < 2000; layout++) {
        std::vector<library::BinaryField> fields(rng() % 6 + 1);
        for (std::size_t i = 0; i < fields.size(); i++) {
            auto &f = fields[i];
            f.key = x::uuid::create();
            f.name = "f" + std::to_string(i);
            f.start_bit = static_cast<std::uint16_t>(rng() % 600);
            f.bit_length = static_cast<std::uint8_t>(rng() % 70);
            f.byte_order = rng() % 2 == 0 ? library::BYTE_ORDER_BIG_ENDIAN
                                          : library::BYTE_ORDER_LITTLE_ENDIAN;
            f.signed_ = rng() % 2 == 0;
            f.float_ = rng() % 8 == 0;
            f.scale = SCALES[rng() % SCALES.size()];
            if (i > 0 && rng() % 3 == 0) {
                f.multiplexor = fields[rng() % i].key;
                f.multiplex_values = {static_cast<std::int32_t>(rng() % 4) - 1};
            }
        }
        std::optional<std::uint16_t> length;
        if (rng() % 2 == 0) length = static_cast<std::uint16_t>(rng() % 80);
        const auto [plan, err] = Plan::compile(message(fields, length));
        if (err) {
            ASSERT_OCCURRED_AS(err, CONFIG_ERROR);
            continue;
        }
        compiled++;
        auto values = plan.values();
        for (int i = 0; i < 20; i++) {
            Bytes payload(rng() % 90);
            for (auto &b: payload)
                b = static_cast<std::uint8_t>(rng());
            const auto decode_err = plan.decode(payload, values);
            if (decode_err) ASSERT_OCCURRED_AS(decode_err, SHORT_PAYLOAD_ERROR);
        }
    }
    EXPECT_GT(compiled, 400);
}

TEST(PlanProperty, DecodeHandlesAnyTextLine) {
    std::mt19937_64 rng(4972);
    const std::string alphabet = "0123456789.-+eE ,;=ABCnai\t\r";
    for (int layout = 0; layout < 300; layout++) {
        library::TextPayload payload;
        payload.delimiter = rng() % 2 == 0 ? "," : (rng() % 2 == 0 ? ";;" : " ");
        const auto count = rng() % 5 + 1;
        for (std::size_t i = 0; i < count; i++)
            payload.fields.push_back(random_text_field(rng, i));
        library::MessageEntry m;
        m.key = x::uuid::create();
        m.payload = payload;
        const auto plan = ASSERT_NIL_P(Plan::compile(m));
        auto values = plan.values();
        for (int i = 0; i < 30; i++) {
            Bytes line(rng() % 60);
            for (auto &b: line)
                b = rng() % 8 == 0 ? static_cast<std::uint8_t>(rng())
                                   : alphabet[rng() % alphabet.size()];
            ASSERT_NIL(plan.decode(line, values));
            EXPECT_LE(values.invalid(), plan.size());
        }
    }
}

/// @brief A value the plan accepts comes back from decode: exactly for floats and exact
/// integers, and to within one raw step for scaled fields. Fields their multiplexors do
/// not select come back absent.
TEST(PlanProperty, DecodeReturnsTheEncodedValue) {
    std::mt19937_64 rng(4973);
    std::size_t selected = 0;
    std::size_t unselected = 0;
    for (int l = 0; l < 300; l++) {
        const Layout layout(rng);
        const auto &fields = layout.all();
        const auto plan = ASSERT_NIL_P(Plan::compile(layout.entry));
        auto values = plan.values();
        auto decoded = plan.values();
        for (int i = 0; i < 20; i++) {
            std::vector<std::uint64_t> raw(fields.size());
            for (std::size_t s = 0; s < fields.size(); s++)
                raw[s] = random_raw(rng, fields[s]);
            // Half the time, a multiplexor takes a value that selects a random field.
            for (std::size_t s = 0; s < fields.size(); s++)
                if (layout.parent[s].has_value() && rng() % 2 == 0) {
                    const auto &v = fields[s].multiplex_values;
                    raw[*layout.parent[s]] = static_cast<std::uint64_t>(
                        v[rng() % v.size()]
                    );
                }
            for (std::size_t s = 0; s < fields.size(); s++)
                ASSERT_NIL(plan.set(values, s, sample(fields[s], raw[s])));
            Bytes payload;
            plan.encode(values, payload);
            ASSERT_EQ(payload.size(), plan.length());
            ASSERT_NIL(plan.decode(payload, decoded));
            for (std::size_t s = 0; s < fields.size(); s++) {
                const auto &f = fields[s];
                const auto ctx = "layout " + std::to_string(l) + " field " + f.name;
                ASSERT_EQ(decoded.present(s), layout.present(s, raw)) << ctx;
                if (layout.parent[s].has_value())
                    (decoded.present(s) ? selected : unselected)++;
                if (!decoded.present(s)) continue;
                if (f.float_ || f.scale != 1 || f.offset != 0) {
                    const auto want = std::get<double>(sample(f, raw[s]));
                    EXPECT_NEAR(decoded.get(s), want, std::abs(f.scale)) << ctx;
                    if (f.float_) EXPECT_EQ(decoded.get(s), want) << ctx;
                    continue;
                }
                x::telem::Series out(x::telem::UINT64_T, 1);
                ASSERT_EQ(decoded.write(s, out), 1);
                EXPECT_EQ(out.at<std::uint64_t>(0), raw[s]) << ctx;
            }
            Bytes again;
            plan.encode(decoded, again);
            EXPECT_EQ(again, payload) << "layout " << l;
        }
    }
    EXPECT_GT(selected, 1000);
    EXPECT_GT(unselected, 1000);
}

TEST(PlanProperty, DecodeReturnsTheEncodedTextValue) {
    std::mt19937_64 rng(4974);
    for (int l = 0; l < 200; l++) {
        library::TextPayload payload{.delimiter = ","};
        const auto count = rng() % 5 + 1;
        for (std::size_t i = 0; i < count; i++)
            payload.fields.push_back(random_text_field(rng, i));
        library::MessageEntry m;
        m.key = x::uuid::create();
        m.payload = payload;
        const auto plan = ASSERT_NIL_P(Plan::compile(m));
        auto values = plan.values();
        auto decoded = plan.values();
        for (int i = 0; i < 20; i++) {
            std::vector<double> want(payload.fields.size());
            for (std::size_t s = 0; s < want.size(); s++) {
                const auto &f = base(payload.fields[s]);
                if (f.scale == 1 && f.offset == 0 && rng() % 2 == 0) {
                    const auto v = static_cast<std::int64_t>(rng() % (1ULL << 53)) -
                                   (std::int64_t{1} << 52);
                    want[s] = static_cast<double>(v);
                    ASSERT_NIL(plan.set(values, s, x::telem::SampleValue(v)));
                    continue;
                }
                const double v = std::ldexp(
                    static_cast<double>(rng() % 2000000) - 1000000,
                    static_cast<int>(rng() % 80) - 40
                );
                want[s] = v;
                ASSERT_NIL(plan.set(values, s, x::telem::SampleValue(v)));
            }
            Bytes line;
            plan.encode(values, line);
            ASSERT_NIL(plan.decode(line, decoded));
            EXPECT_EQ(decoded.invalid(), 0);
            for (std::size_t s = 0; s < want.size(); s++) {
                const auto &f = base(payload.fields[s]);
                const auto tolerance = 1e-12 *
                                       std::max(
                                           {1.0, std::abs(want[s]), std::abs(f.offset)}
                                       );
                EXPECT_NEAR(decoded.get(s), want[s], tolerance)
                    << std::string(line.begin(), line.end());
            }
        }
    }
}
}
