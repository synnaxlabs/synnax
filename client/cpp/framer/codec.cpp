// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <cstdint>
#include <iterator>
#include <memory>
#include <string>
#include <vector>

#include "client/cpp/channel/types.gen.h"
#include "client/cpp/framer/framer.h"
#include "x/cpp/telem/code.h"

namespace synnax::framer {
uint8_t CodecFlags::encode() const {
    uint8_t b = 0;
    b = x::binary::set_bit(b, FlagPosition::EqualLengths, equal_lens);
    b = x::binary::set_bit(b, FlagPosition::EqualTimeRanges, equal_time_ranges);
    b = x::binary::set_bit(b, FlagPosition::TimeRangesZero, time_ranges_zero);
    b = x::binary::set_bit(b, FlagPosition::AllChannelsPresent, all_channels_present);
    b = x::binary::set_bit(b, FlagPosition::EqualAlignments, equal_alignments);
    b = x::binary::set_bit(b, FlagPosition::ZeroAlignments, zero_alignments);
    b = x::binary::set_bit(b, FlagPosition::MultiplesPresent, multiples_present);
    b = x::binary::set_bit(b, FlagPosition::Extended, extended);
    return b;
}

CodecFlags CodecFlags::decode(const uint8_t b) {
    CodecFlags f;
    f.equal_lens = x::binary::get_bit(b, FlagPosition::EqualLengths);
    f.equal_time_ranges = x::binary::get_bit(b, FlagPosition::EqualTimeRanges);
    f.time_ranges_zero = x::binary::get_bit(b, FlagPosition::TimeRangesZero);
    f.all_channels_present = x::binary::get_bit(b, FlagPosition::AllChannelsPresent);
    f.equal_alignments = x::binary::get_bit(b, FlagPosition::EqualAlignments);
    f.zero_alignments = x::binary::get_bit(b, FlagPosition::ZeroAlignments);
    f.multiples_present = x::binary::get_bit(b, FlagPosition::MultiplesPresent);
    f.extended = x::binary::get_bit(b, FlagPosition::Extended);
    return f;
}

namespace {
constexpr std::uint8_t TYPES_DECLARED = 1 << 0;
constexpr std::uint8_t OVERRIDES_PRESENT = 1 << 1;
constexpr std::uint8_t CLOCKS_DECLARED = 1 << 3;
/// @brief the extension bits that no codec sets yet.
constexpr std::uint8_t RESERVED_EXTENSION_FLAGS = 0b1111'0100;
constexpr std::uint8_t TYPE_KIND = 1 << 0;
constexpr std::uint8_t CLOCK_KIND = 1 << 1;
/// @brief the override kinds that no codec sets yet.
constexpr std::uint8_t RESERVED_KINDS = 0b1111'1100;
/// @brief the next override ordinal that ends the chain.
constexpr std::uint32_t NO_OVERRIDE = 0xFFFFFFFF;

constexpr std::size_t ALIGNMENT_SIZE = 8;
constexpr std::size_t DATA_LENGTH_SIZE = 4;
constexpr std::size_t KEY_SIZE = 4;
constexpr std::size_t FLAGS_SIZE = 1;
constexpr std::size_t SEQ_NUM_SIZE = 4;
constexpr std::size_t TIME_RANGE_SIZE = 16;
constexpr std::size_t BITS_PER_BYTE = 8;

constexpr std::size_t bit_packed_byte_count(const std::size_t n_samples) {
    return (n_samples + BITS_PER_BYTE - 1) / BITS_PER_BYTE;
}

std::size_t series_wire_byte_length(const x::telem::Series &ser) {
    if (ser.data_type() == x::telem::BOOLEAN_T)
        return bit_packed_byte_count(ser.size());
    return ser.byte_size();
}

std::vector<std::uint8_t>
pack_bool_bits(const std::byte *src, const std::size_t n_samples) {
    std::vector<std::uint8_t> dst(bit_packed_byte_count(n_samples), 0);
    const auto *src_bytes = reinterpret_cast<const std::uint8_t *>(src);
    for (std::size_t i = 0; i < n_samples; i++)
        if (src_bytes[i] != 0)
            dst[i / BITS_PER_BYTE] |= static_cast<std::uint8_t>(
                1u << (i % BITS_PER_BYTE)
            );
    return dst;
}

std::vector<std::uint8_t>
unpack_bool_bits(const std::uint8_t *src, const std::size_t n_samples) {
    std::vector<std::uint8_t> dst(n_samples);
    for (std::size_t i = 0; i < n_samples; i++)
        dst[i] = (src[i / BITS_PER_BYTE] >> (i % BITS_PER_BYTE)) & 1;
    return dst;
}

bool is_int64_like(const x::telem::DataType &dt) {
    return dt == x::telem::INT64_T || dt == x::telem::TIMESTAMP_T;
}

bool needs_type_override(
    const x::telem::DataType &agreed,
    const x::telem::DataType &dt
) {
    return agreed != dt && (!is_int64_like(agreed) || !is_int64_like(dt));
}

std::string bits(const std::uint8_t b) {
    std::string out(8, '0');
    for (int i = 0; i < 8; i++)
        if ((b >> (7 - i)) & 1) out[i] = '1';
    return out;
}

/// @brief returns the ascending union of a and b, which are ascending and disjoint.
std::vector<channel::Key>
union_of(const std::vector<channel::Key> &a, const std::vector<channel::Key> &b) {
    if (b.empty()) return a;
    std::vector<channel::Key> out;
    out.reserve(a.size() + b.size());
    std::ranges::merge(a, b, std::back_inserter(out));
    return out;
}

/// @brief a decode failure, which decode returns as a validation error.
struct Invalid {
    std::string message;
};

/// @brief reads a frame, failing on any read past its end.
class FrameReader {
    const std::uint8_t *buf;
    std::size_t size;
    std::size_t off = 0;

    void need(const std::size_t n) const {
        if (n > this->remaining()) throw Invalid{"remote sent a frame that ends early"};
    }

public:
    FrameReader(const std::uint8_t *buf, const std::size_t size):
        buf(buf), size(size) {}

    [[nodiscard]] std::size_t remaining() const { return this->size - this->off; }

    std::uint8_t u8() {
        this->need(1);
        return this->buf[this->off++];
    }

    std::uint32_t u32() {
        this->need(4);
        std::uint32_t v = 0;
        for (int i = 0; i < 4; i++)
            v |= static_cast<std::uint32_t>(this->buf[this->off++]) << (8 * i);
        return v;
    }

    std::uint64_t u64() {
        this->need(8);
        std::uint64_t v = 0;
        for (int i = 0; i < 8; i++)
            v |= static_cast<std::uint64_t>(this->buf[this->off++]) << (8 * i);
        return v;
    }

    const std::uint8_t *bytes(const std::size_t n) {
        this->need(n);
        const auto *p = this->buf + this->off;
        this->off += n;
        return p;
    }

    x::telem::DataType data_type() {
        const auto code = this->u8();
        auto dt = x::telem::data_type_of(code);
        if (dt == x::telem::UNKNOWN_T)
            throw Invalid{
                "remote sent data type code " + std::to_string(code) +
                ", which this codec does not know"
            };
        return dt;
    }
};
}

Codec::State::State(
    const std::vector<channel::Key> &keys,
    const std::vector<x::telem::DataType> &data_types
):
    keys(keys) {
    std::ranges::sort(this->keys);
    const auto [first, last] = std::ranges::unique(this->keys);
    this->keys.erase(first, last);
    for (std::size_t i = 0; i < this->keys.size(); i++)
        this->index[this->keys[i]] = i;
    const auto n = this->keys.size();
    this->enc.data_types.assign(n, x::telem::UNKNOWN_T);
    this->dec.data_types.assign(n, x::telem::UNKNOWN_T);
    this->enc.clocks.assign(n, 0);
    this->dec.clocks.assign(n, 0);
    this->enc.all = this->keys;
    this->dec.all = this->keys;
    for (std::size_t i = 0; i < keys.size(); i++)
        this->enc.data_types[this->index.at(keys[i])] = data_types[i];
}

x::errors::Error Codec::update(const std::vector<channel::Key> &keys) {
    // The Core counts every update that carries keys, and both sides must agree on the
    // sequence number of each state.
    if (keys.empty()) return x::errors::NIL;
    auto [channels, err] = this->channel_client.retrieve(keys);
    if (err) return err;
    std::vector<channel::Key> state_keys;
    std::vector<x::telem::DataType> data_types;
    for (const auto &ch: channels) {
        state_keys.push_back(ch.key);
        data_types.push_back(ch.data_type);
    }
    std::lock_guard lock(*this->mu);
    this->states[++this->seq_num] = State(state_keys, data_types);
    return x::errors::NIL;
}

void Codec::throw_if_uninitialized() const {
    if (this->seq_num < 1) throw std::runtime_error("codec is uninitialized");
}

Codec::Codec(
    const std::vector<channel::Key> &channels,
    const std::vector<x::telem::DataType> &data_types
):
    seq_num(1), channel_client(nullptr, nullptr) {
    this->states[this->seq_num] = State(channels, data_types);
}

namespace {
/// @brief the extension flags, type codes, overrides, and declared clocks of one
/// frame.
struct Plan {
    std::uint8_t ext = 0;
    std::vector<std::uint8_t> codes;
    std::vector<std::uint8_t> kinds;
    std::vector<std::uint32_t> clocks;
    std::vector<channel::Key> declared;
    std::vector<channel::Key> all;
};

x::errors::Error no_code(const channel::Key key, const x::telem::DataType &dt) {
    return x::errors::Error(
        x::errors::VALIDATION,
        "data type \"" + dt.name() + "\" of channel " + std::to_string(key) +
            " has no type code"
    );
}
}

x::errors::Error
Codec::encode(const x::telem::Frame &frame, std::vector<uint8_t> &output) {
    std::lock_guard lock(*this->mu);
    this->throw_if_uninitialized();
    auto &state = this->states.at(this->seq_num);
    auto &ag = state.enc;

    this->entries.clear();
    for (std::size_t i = 0; i < frame.size(); i++) {
        const auto k = frame.channels->at(i);
        const auto &ser = frame.series->at(i);
        if (const auto it = state.index.find(k); it != state.index.end())
            this->entries.push_back({k, i, static_cast<std::int64_t>(it->second)});
        else if (ser.clock == k && is_int64_like(ser.data_type()))
            this->entries.push_back({k, i, -1});
        else
            return x::errors::Error(
                x::errors::VALIDATION,
                "frame contains extra key " + std::to_string(k) +
                    " not provided when opening the writer"
            );
    }
    std::ranges::stable_sort(this->entries, {}, &Entry::key);
    const auto series_of = [&](const Entry &e) -> const x::telem::Series & {
        return frame.series->at(e.raw);
    };

    Plan plan;
    plan.clocks = ag.clocks;
    plan.declared = ag.declared;
    plan.all = ag.all;
    if (!ag.types_declared) {
        plan.ext |= TYPES_DECLARED;
        for (std::size_t i = 0; i < state.keys.size(); i++) {
            const auto code = x::telem::code_of(ag.data_types[i]);
            if (code == x::telem::DataTypeCode::Unknown)
                return no_code(state.keys[i], ag.data_types[i]);
            plan.codes.push_back(static_cast<std::uint8_t>(code));
        }
    }
    bool overrides = false;
    bool declared_in_frame = false;
    for (const auto &e: this->entries) {
        if (e.index < 0) {
            declared_in_frame = true;
            continue;
        }
        const auto &ser = series_of(e);
        const auto i = static_cast<std::size_t>(e.index);
        if (ser.clock != ag.clocks[i] ||
            needs_type_override(ag.data_types[i], ser.data_type()))
            overrides = true;
    }
    if (overrides) {
        plan.ext |= OVERRIDES_PRESENT;
        for (const auto &e: this->entries) {
            std::uint8_t kinds = 0;
            if (e.index >= 0) {
                const auto &ser = series_of(e);
                const auto i = static_cast<std::size_t>(e.index);
                if (needs_type_override(ag.data_types[i], ser.data_type())) {
                    if (x::telem::code_of(ser.data_type()) ==
                        x::telem::DataTypeCode::Unknown)
                        return no_code(e.key, ser.data_type());
                    kinds |= TYPE_KIND;
                }
                if (ser.clock != plan.clocks[i]) {
                    kinds |= CLOCK_KIND;
                    plan.clocks[i] = ser.clock;
                }
            }
            plan.kinds.push_back(kinds);
        }
    }
    if (declared_in_frame || !ag.declared.empty()) {
        std::vector<channel::Key> declared;
        for (const auto &e: this->entries)
            if (e.index < 0 && (declared.empty() || declared.back() != e.key))
                declared.push_back(e.key);
        for (const auto k: ag.declared)
            if (!std::ranges::binary_search(declared, k) &&
                std::ranges::find(plan.clocks, k) != plan.clocks.end())
                declared.push_back(k);
        std::ranges::sort(declared);
        if (declared != ag.declared) {
            plan.ext |= CLOCKS_DECLARED;
            plan.all = union_of(state.keys, declared);
            plan.declared = std::move(declared);
        }
    }

    CodecFlags flags;
    flags.extended = plan.ext != 0;
    const auto n = this->entries.size();
    size_t byte_array_size = FLAGS_SIZE + SEQ_NUM_SIZE;
    flags.all_channels_present = n == plan.all.size();
    for (std::size_t i = 0; flags.all_channels_present && i < n; i++)
        if (this->entries[i].key != plan.all[i]) flags.all_channels_present = false;
    if (!flags.all_channels_present) byte_array_size += n * KEY_SIZE;

    size_t cur_data_size = 0;
    bool first_series = true;
    x::telem::TimeRange ref_tr = {};
    x::telem::Alignment ref_alignment;
    for (const auto &e: this->entries) {
        const auto &series = series_of(e);
        byte_array_size += series_wire_byte_length(series);
        if (series.data_type().is_variable()) flags.equal_lens = false;
        if (first_series) {
            cur_data_size = series.size();
            ref_tr = series.time_range;
            ref_alignment = series.alignment;
            first_series = false;
            continue;
        }
        if (series.size() != cur_data_size) flags.equal_lens = false;
        if (series.time_range != ref_tr) flags.equal_time_ranges = false;
        if (series.alignment != ref_alignment) flags.equal_alignments = false;
    }
    flags.time_ranges_zero = flags.equal_time_ranges &&
                             ref_tr.start.nanoseconds() == 0 &&
                             ref_tr.end.nanoseconds() == 0;
    flags.zero_alignments = flags.equal_alignments && ref_alignment == 0;
    byte_array_size += (flags.equal_lens ? 1 : n) * DATA_LENGTH_SIZE;
    if (!flags.time_ranges_zero)
        byte_array_size += (flags.equal_time_ranges ? 1 : n) * TIME_RANGE_SIZE;
    if (!flags.zero_alignments)
        byte_array_size += (flags.equal_alignments ? 1 : n) * ALIGNMENT_SIZE;
    if (plan.ext != 0) byte_array_size += 1 + plan.codes.size();
    if (plan.ext & CLOCKS_DECLARED)
        byte_array_size += KEY_SIZE * (1 + plan.declared.size());
    if (plan.ext & OVERRIDES_PRESENT) byte_array_size += 4;
    for (const auto kinds: plan.kinds) {
        if (kinds == 0) continue;
        byte_array_size += 5;
        if (kinds & TYPE_KIND) byte_array_size += 1;
        if (kinds & CLOCK_KIND) byte_array_size += 4;
    }

    x::binary::Writer buf(output, byte_array_size);
    buf.uint8(flags.encode());
    buf.uint32(this->seq_num);
    if (plan.ext != 0) {
        buf.uint8(plan.ext);
        for (const auto code: plan.codes)
            buf.uint8(code);
    }
    if (plan.ext & CLOCKS_DECLARED) {
        buf.uint32(static_cast<std::uint32_t>(plan.declared.size()));
        for (const auto k: plan.declared)
            buf.uint32(k);
    }
    if (flags.equal_lens)
        buf.uint32(first_series ? NO_OVERRIDE : static_cast<uint32_t>(cur_data_size));
    if (flags.equal_time_ranges && !flags.time_ranges_zero) {
        buf.int64(ref_tr.start.nanoseconds());
        buf.int64(ref_tr.end.nanoseconds());
    }
    if (flags.equal_alignments && !flags.zero_alignments)
        buf.uint64(ref_alignment.uint64());
    const auto next_override = [&](std::size_t from) -> std::uint32_t {
        for (; from < plan.kinds.size(); from++)
            if (plan.kinds[from] != 0) return static_cast<std::uint32_t>(from);
        return NO_OVERRIDE;
    };
    if (plan.ext & OVERRIDES_PRESENT) buf.uint32(next_override(0));
    for (std::size_t i = 0; i < n; i++) {
        const auto &e = this->entries[i];
        const auto &ser = series_of(e);
        if (!flags.all_channels_present) buf.uint32(e.key);
        if (const auto kinds = i < plan.kinds.size() ? plan.kinds[i] : 0; kinds != 0) {
            buf.uint8(kinds);
            if (kinds & TYPE_KIND)
                buf.uint8(
                    static_cast<std::uint8_t>(x::telem::code_of(ser.data_type()))
                );
            if (kinds & CLOCK_KIND) buf.uint32(ser.clock);
            buf.uint32(next_override(i + 1));
        }
        const auto byte_size = ser.byte_size();
        if (!flags.equal_lens)
            buf.uint32(
                static_cast<uint32_t>(
                    ser.data_type().is_variable() ? byte_size : ser.size()
                )
            );
        if (ser.data_type() == x::telem::BOOLEAN_T) {
            const auto packed = pack_bool_bits(ser.data(), ser.size());
            buf.write(
                reinterpret_cast<const std::byte *>(packed.data()),
                packed.size()
            );
        } else
            buf.write(ser.data(), byte_size);
        if (!flags.equal_time_ranges) {
            buf.int64(ser.time_range.start.nanoseconds());
            buf.int64(ser.time_range.end.nanoseconds());
        }
        if (!flags.equal_alignments) buf.uint64(ser.alignment.uint64());
    }
    ag.types_declared = true;
    ag.clocks = std::move(plan.clocks);
    ag.declared = std::move(plan.declared);
    ag.all = std::move(plan.all);
    return x::errors::NIL;
}

std::pair<x::telem::Frame, x::errors::Error>
Codec::decode(const std::vector<uint8_t> &data) {
    return this->decode(data.data(), data.size());
}

namespace {
x::errors::Error invalid(const std::string &message) {
    return x::errors::Error(x::errors::VALIDATION, message);
}
}

std::pair<x::telem::Frame, x::errors::Error>
Codec::decode(const uint8_t *data, const size_t size) {
    std::lock_guard lock(*this->mu);
    this->throw_if_uninitialized();
    FrameReader r(data, size);
    x::telem::Frame frame;
    State *state = nullptr;
    try {
        const auto flags = CodecFlags::decode(r.u8());
        if (flags.multiples_present)
            return {
                std::move(frame),
                invalid(
                    "remote sent alignment multiples, which this codec does not "
                    "support"
                )
            };
        const auto seq_n = r.u32();
        const auto it = this->states.find(seq_n);
        if (it == this->states.end())
            return {
                std::move(frame),
                invalid("remote sent invalid sequence number " + std::to_string(seq_n))
            };
        // The remote never encodes with an older state again.
        for (; this->oldest < seq_n; this->oldest++)
            this->states.erase(this->oldest);
        if (it->second.dec.dropped)
            return {
                std::move(frame),
                invalid(
                    "remote sent a frame for state " + std::to_string(seq_n) +
                    " after a frame of it failed to decode"
                )
            };
        state = &it->second;
        auto &ag = state->dec;
        std::uint8_t ext = 0;
        if (flags.extended) {
            ext = r.u8();
            if (ext == 0) throw Invalid{"remote sent extension flags of 0"};
            if (ext & RESERVED_EXTENSION_FLAGS)
                throw Invalid{
                    "remote sent extension flags " + bits(ext) +
                    ", which hold a reserved bit"
                };
        }
        if (!(ext & TYPES_DECLARED)) {
            if (!ag.types_declared)
                throw Invalid{
                    "remote sent no type declaration for state " +
                    std::to_string(seq_n) +
                    ", so it runs an older codec, which this codec does not support"
                };
        } else {
            if (ag.types_declared)
                throw Invalid{
                    "remote declared the types of state " + std::to_string(seq_n) +
                    " twice"
                };
            for (auto &dt: ag.data_types)
                dt = r.data_type();
            ag.types_declared = true;
        }
        if (ext & CLOCKS_DECLARED) {
            const auto count = r.u32();
            if (count > r.remaining() / KEY_SIZE)
                throw Invalid{"remote sent a frame that ends early"};
            std::vector<channel::Key> declared;
            declared.reserve(count);
            for (std::uint32_t i = 0; i < count; i++) {
                const auto key = r.u32();
                if (state->index.contains(key))
                    throw Invalid{
                        "remote declared clock " + std::to_string(key) +
                        ", which the request names"
                    };
                if (!declared.empty() && key <= declared.back())
                    throw Invalid{
                        "remote declared clock " + std::to_string(key) +
                        " after clock " + std::to_string(declared.back()) +
                        ", so the list is not ascending"
                    };
                declared.push_back(key);
            }
            ag.all = union_of(state->keys, declared);
            ag.declared = std::move(declared);
        }
        const std::uint32_t shared_len = flags.equal_lens ? r.u32() : 0;
        x::telem::TimeRange ref_tr{x::telem::TimeStamp(0), x::telem::TimeStamp(0)};
        if (flags.equal_time_ranges && !flags.time_ranges_zero) {
            ref_tr.start = x::telem::TimeStamp(static_cast<std::int64_t>(r.u64()));
            ref_tr.end = x::telem::TimeStamp(static_cast<std::int64_t>(r.u64()));
        }
        x::telem::Alignment ref_alignment;
        if (flags.equal_alignments && !flags.zero_alignments)
            ref_alignment = x::telem::Alignment(r.u64());
        std::uint32_t next = (ext & OVERRIDES_PRESENT) ? r.u32() : NO_OVERRIDE;
        std::uint32_t ordinal = 0;

        const auto read_kinds = [&]() {
            const auto kinds = r.u8();
            if (kinds == 0)
                throw Invalid{
                    "remote sent override kinds of 0 for series " +
                    std::to_string(ordinal)
                };
            if (kinds & RESERVED_KINDS)
                throw Invalid{
                    "remote sent override kinds " + bits(kinds) + " for series " +
                    std::to_string(ordinal) + ", which hold a reserved bit"
                };
            return kinds;
        };

        const auto decode_series = [&](const channel::Key key) {
            x::telem::DataType dt = x::telem::TIMESTAMP_T;
            std::uint32_t clock = key;
            if (const auto it = state->index.find(key); it != state->index.end()) {
                const auto i = it->second;
                dt = ag.data_types[i];
                if (ordinal == next) {
                    const auto kinds = read_kinds();
                    if (kinds & TYPE_KIND) dt = r.data_type();
                    if (kinds & CLOCK_KIND) ag.clocks[i] = r.u32();
                    next = r.u32();
                    if (next != NO_OVERRIDE && next <= ordinal)
                        throw Invalid{
                            "remote sent next override " + std::to_string(next) +
                            ", which is not after series " + std::to_string(ordinal)
                        };
                }
                clock = ag.clocks[i];
            } else {
                if (!std::ranges::binary_search(ag.declared, key))
                    throw Invalid{"unknown channel key: " + std::to_string(key)};
                if (ordinal == next) {
                    (void) read_kinds();
                    throw Invalid{
                        "remote sent an override for declared clock " +
                        std::to_string(key)
                    };
                }
            }
            ordinal++;
            if (flags.equal_lens && dt.is_variable())
                throw Invalid{
                    "remote sent variable data type " + dt.name() +
                    " under a shared length"
                };
            const std::uint32_t len = flags.equal_lens ? shared_len : r.u32();
            auto s = x::telem::Series(dt, len);
            if (dt == x::telem::BOOLEAN_T) {
                const auto *packed = r.bytes(bit_packed_byte_count(len));
                const auto unpacked = unpack_bool_bits(packed, len);
                s.write(unpacked.data(), len);
            } else {
                const std::size_t n = dt.is_variable() ? len : len * dt.density();
                x::binary::Reader sub(r.bytes(n), n);
                s.fill_from(sub);
            }
            s.time_range = ref_tr;
            s.alignment = ref_alignment;
            s.clock = clock;
            if (!flags.equal_time_ranges) {
                s.time_range.start = x::telem::TimeStamp(
                    static_cast<std::int64_t>(r.u64())
                );
                s.time_range.end = x::telem::TimeStamp(
                    static_cast<std::int64_t>(r.u64())
                );
            }
            if (!flags.equal_alignments) s.alignment = x::telem::Alignment(r.u64());
            if (frame.channels == nullptr) {
                frame.channels = std::make_unique<std::vector<channel::Key>>();
                frame.series = std::make_unique<std::vector<x::telem::Series>>();
            }
            frame.emplace(key, std::move(s));
        };

        if (flags.all_channels_present) {
            frame.reserve(ag.all.size());
            for (const auto key: ag.all)
                decode_series(key);
            if (r.remaining() > 0)
                throw Invalid{"remote sent bytes after the last series"};
        } else
            while (r.remaining() > 0)
                decode_series(r.u32());
        if (next != NO_OVERRIDE)
            throw Invalid{
                "remote ended the frame with the override for series " +
                std::to_string(next) + " pending"
            };
        return {std::move(frame), x::errors::NIL};
    } catch (const Invalid &e) {
        if (state != nullptr) state->dec.dropped = true;
        return {x::telem::Frame(), invalid(e.message)};
    }
}
}
