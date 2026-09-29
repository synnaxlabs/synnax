// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <string>

#include "driver/codec/framing/checksum.h"
#include "driver/codec/framing/framing.h"

namespace driver::codec::framing {
namespace {
namespace bus = synnax::bus;
namespace library = synnax::library;

using Bytes = std::vector<std::uint8_t>;
using View = std::span<const std::uint8_t>;

std::uint64_t
read_uint(const std::uint8_t *data, const std::size_t size, const bool big) {
    std::uint64_t v = 0;
    for (std::size_t i = 0; i < size; i++)
        v = (v << 8) | (big ? data[i] : data[size - 1 - i]);
    return v;
}

void write_uint(
    std::uint8_t *data,
    const std::uint64_t v,
    const std::size_t size,
    const bool big
) {
    for (std::size_t i = 0; i < size; i++) {
        const auto shift = (big ? size - 1 - i : i) * 8;
        data[i] = static_cast<std::uint8_t>(v >> shift);
    }
}

x::errors::Error overflow() {
    return x::errors::Error(
        OVERFLOW_ERROR,
        "frame exceeded " + std::to_string(MAX_SIZE) + " bytes"
    );
}

x::errors::Error encode_error(const std::string &msg) {
    return x::errors::Error(ENCODE_ERROR, msg);
}

x::errors::Error config_error(const std::string &msg) {
    return x::errors::Error(CONFIG_ERROR, msg);
}

x::errors::Error validate(const View frame, const std::size_t wire_size) {
    if (frame.empty()) return encode_error("frame is empty");
    if (wire_size > MAX_SIZE)
        return encode_error(
            "frame of " + std::to_string(frame.size()) + " bytes exceeds " +
            std::to_string(MAX_SIZE) + " bytes on the wire"
        );
    return x::errors::NIL;
}

class Delimiter final : public Framer {
    const Bytes delimiter;
    /// @brief buf holds the bytes of the frame in progress.
    Bytes buf;
    /// @brief scanned is where the next delimiter search starts in buf.
    std::size_t scanned = 0;
    /// @brief discarding is true while dropping the rest of an oversized frame.
    bool discarding = false;
    /// @brief errors_ counts framing errors.
    std::size_t errors_ = 0;

    void extract(const Handler &handler) {
        const auto &d = this->delimiter;
        std::size_t head = 0;
        while (true) {
            const auto from = static_cast<std::ptrdiff_t>(
                std::max(head, this->scanned)
            );
            const auto it = std::search(
                this->buf.begin() + from,
                this->buf.end(),
                d.begin(),
                d.end()
            );
            if (it == this->buf.end()) break;
            const auto end = static_cast<std::size_t>(it - this->buf.begin());
            if (this->discarding)
                this->discarding = false;
            else if (end > head)
                handler(View(this->buf.data() + head, end - head));
            head = end + d.size();
        }
        const auto tail = this->buf.size() - std::min(this->buf.size(), d.size() - 1);
        this->scanned = std::max(head, tail) - head;
        this->buf.erase(
            this->buf.begin(),
            this->buf.begin() + static_cast<std::ptrdiff_t>(head)
        );
    }

public:
    explicit Delimiter(Bytes delimiter): delimiter(std::move(delimiter)) {
        this->buf.reserve(MAX_SIZE);
    }

    x::errors::Error write(View chunk, const Handler &handler) override {
        auto err = x::errors::NIL;
        while (!chunk.empty()) {
            const auto room = MAX_SIZE - this->buf.size();
            if (room == 0) {
                if (!this->discarding) {
                    this->errors_++;
                    err = overflow();
                    this->discarding = true;
                }
                // Keeps the bytes that could begin a delimiter split across chunks.
                const auto keep = this->delimiter.size() - 1;
                this->buf.erase(
                    this->buf.begin(),
                    this->buf.end() - static_cast<std::ptrdiff_t>(keep)
                );
                this->scanned = 0;
                continue;
            }
            const auto n = std::min(room, chunk.size());
            this->buf.insert(this->buf.end(), chunk.begin(), chunk.begin() + n);
            chunk = chunk.subspan(n);
            this->extract(handler);
        }
        return err;
    }

    [[nodiscard]] std::size_t errors() const override { return this->errors_; }

    void reset() override {
        this->buf.clear();
        this->scanned = 0;
        this->discarding = false;
    }

    x::errors::Error encode(const View frame, Bytes &out) const override {
        const auto &d = this->delimiter;
        if (auto err = validate(frame, frame.size() + d.size())) return err;
        const auto start = out.size();
        out.insert(out.end(), frame.begin(), frame.end());
        out.insert(out.end(), d.begin(), d.end());
        // The first delimiter on the wire must be the one appended, which also rules
        // out a frame whose tail and the delimiter form an earlier match.
        const auto begin = out.begin() + static_cast<std::ptrdiff_t>(start);
        if (std::search(begin, out.end(), d.begin(), d.end()) - begin !=
            static_cast<std::ptrdiff_t>(frame.size())) {
            out.resize(start);
            return encode_error("frame contains the delimiter");
        }
        return x::errors::NIL;
    }
};

class Fixed final : public Framer {
    const std::size_t length;
    /// @brief buf holds the bytes of the frame in progress.
    Bytes buf;

public:
    explicit Fixed(const std::size_t length): length(length) {
        this->buf.reserve(length);
    }

    x::errors::Error write(View chunk, const Handler &handler) override {
        while (!chunk.empty()) {
            if (this->buf.empty() && chunk.size() >= this->length) {
                handler(chunk.first(this->length));
                chunk = chunk.subspan(this->length);
                continue;
            }
            const auto n = std::min(this->length - this->buf.size(), chunk.size());
            this->buf.insert(this->buf.end(), chunk.begin(), chunk.begin() + n);
            chunk = chunk.subspan(n);
            if (this->buf.size() == this->length) {
                handler(this->buf);
                this->buf.clear();
            }
        }
        return x::errors::NIL;
    }

    [[nodiscard]] std::size_t errors() const override { return 0; }

    void reset() override { this->buf.clear(); }

    x::errors::Error encode(const View frame, Bytes &out) const override {
        if (frame.size() != this->length)
            return encode_error(
                "frame of " + std::to_string(frame.size()) + " bytes is not " +
                std::to_string(this->length) + " bytes long"
            );
        out.insert(out.end(), frame.begin(), frame.end());
        return x::errors::NIL;
    }
};

/// @brief SyncLayout is a sync framing resolved from its schema.
struct SyncLayout {
    /// @brief sync is the sequence that starts each frame.
    Bytes sync;
    /// @brief length_offset is the byte offset of the length field.
    std::size_t length_offset = 0;
    /// @brief length_size is the size of the length field in bytes.
    std::size_t length_size = 1;
    /// @brief length_big is true when the length field is big-endian.
    bool length_big = false;
    /// @brief length_adjustment is added to the length field's value.
    std::int64_t length_adjustment = 0;
    /// @brief checksum ends each frame and covers every byte before it.
    Checksum checksum = Checksum::NONE;
    /// @brief checksum_big is true when the checksum is big-endian.
    bool checksum_big = false;
};

class Sync final : public Framer {
    const SyncLayout layout;
    /// @brief header is the number of bytes through the end of the length field.
    const std::size_t header;
    /// @brief min_size is the size of the smallest valid frame on the wire.
    const std::size_t min_size;
    /// @brief buf holds unconsumed bytes, starting at a sync sequence when found.
    Bytes buf;
    /// @brief errors_ counts framing errors.
    std::size_t errors_ = 0;

    /// @returns the wire size of the frame at head, or 0 when the frame is invalid.
    [[nodiscard]] std::size_t frame_size(const std::uint8_t *frame) const {
        const auto &l = this->layout;
        const auto value = read_uint(
            frame + l.length_offset,
            l.length_size,
            l.length_big
        );
        const auto size = static_cast<std::int64_t>(this->header) +
                          static_cast<std::int64_t>(value) + l.length_adjustment;
        if (size < static_cast<std::int64_t>(this->min_size) ||
            size > static_cast<std::int64_t>(MAX_SIZE))
            return 0;
        return static_cast<std::size_t>(size);
    }

    [[nodiscard]] bool
    checksum_valid(const std::uint8_t *frame, const std::size_t size) const {
        const auto &l = this->layout;
        if (l.checksum == Checksum::NONE) return true;
        const auto w = width(l.checksum);
        const auto stored = read_uint(frame + size - w, w, l.checksum_big);
        return checksum(l.checksum, View(frame, size - w)) == stored;
    }

    void extract(const Handler &handler) {
        const auto &sync = this->layout.sync;
        std::size_t head = 0;
        while (true) {
            if (!sync.empty()) {
                const auto it = std::search(
                    this->buf.begin() + static_cast<std::ptrdiff_t>(head),
                    this->buf.end(),
                    sync.begin(),
                    sync.end()
                );
                auto pos = static_cast<std::size_t>(it - this->buf.begin());
                // Keeps the bytes that could begin a sync sequence split across chunks.
                if (it == this->buf.end())
                    pos = std::max(
                        head,
                        this->buf.size() - std::min(this->buf.size(), sync.size() - 1)
                    );
                if (pos > head) this->errors_++;
                head = pos;
                if (it == this->buf.end()) break;
            }
            const auto avail = this->buf.size() - head;
            if (avail < this->header) break;
            const auto *frame = this->buf.data() + head;
            const auto size = this->frame_size(frame);
            if (size == 0) {
                this->errors_++;
                head++;
                continue;
            }
            if (avail < size) break;
            if (!this->checksum_valid(frame, size)) {
                this->errors_++;
                head++;
                continue;
            }
            handler(View(frame, size - width(this->layout.checksum)));
            head += size;
        }
        this->buf.erase(
            this->buf.begin(),
            this->buf.begin() + static_cast<std::ptrdiff_t>(head)
        );
    }

public:
    explicit Sync(SyncLayout layout):
        layout(std::move(layout)),
        header(this->layout.length_offset + this->layout.length_size),
        min_size(this->header + width(this->layout.checksum)) {
        this->buf.reserve(MAX_SIZE);
    }

    x::errors::Error write(View chunk, const Handler &handler) override {
        // extract leaves fewer than MAX_SIZE bytes: an incomplete frame no longer than
        // MAX_SIZE, or a partial header or sync sequence.
        while (!chunk.empty()) {
            const auto n = std::min(MAX_SIZE - this->buf.size(), chunk.size());
            this->buf.insert(this->buf.end(), chunk.begin(), chunk.begin() + n);
            chunk = chunk.subspan(n);
            this->extract(handler);
        }
        return x::errors::NIL;
    }

    [[nodiscard]] std::size_t errors() const override { return this->errors_; }

    void reset() override { this->buf.clear(); }

    x::errors::Error encode(const View frame, Bytes &out) const override {
        const auto &l = this->layout;
        const auto w = width(l.checksum);
        const auto size = frame.size() + w;
        if (auto err = validate(frame, size)) return err;
        if (frame.size() < this->header)
            return encode_error(
                "frame of " + std::to_string(frame.size()) +
                " bytes is shorter than its header"
            );
        const auto value = static_cast<std::int64_t>(size) -
                           static_cast<std::int64_t>(this->header) -
                           l.length_adjustment;
        const auto max = (std::int64_t{1} << (8 * l.length_size)) - 1;
        if (value < 0 || value > max)
            return encode_error(
                "length field cannot hold " + std::to_string(value) +
                " for a frame of " + std::to_string(size) + " bytes"
            );
        const auto start = out.size();
        out.insert(out.end(), frame.begin(), frame.end());
        auto *f = out.data() + start;
        std::copy(l.sync.begin(), l.sync.end(), f);
        write_uint(
            f + l.length_offset,
            static_cast<std::uint64_t>(value),
            l.length_size,
            l.length_big
        );
        if (w == 0) return x::errors::NIL;
        const auto sum = checksum(l.checksum, View(f, frame.size()));
        out.resize(start + size);
        write_uint(out.data() + start + frame.size(), sum, w, l.checksum_big);
        return x::errors::NIL;
    }
};

/// @brief Unstuffed holds the decoded frame and error state that COBS and SLIP share.
struct Unstuffed {
    /// @brief out holds the decoded bytes of the frame in progress.
    Bytes out;
    /// @brief discarding is true while dropping the rest of a corrupt frame.
    bool discarding = false;
    /// @brief errors counts framing errors.
    std::size_t errors = 0;
    /// @brief err is the overflow error of the current write, if any.
    x::errors::Error err = x::errors::NIL;

    Unstuffed() { this->out.reserve(MAX_SIZE); }

    void push(const std::uint8_t b) {
        if (this->out.size() < MAX_SIZE) {
            this->out.push_back(b);
            return;
        }
        this->err = overflow();
        this->corrupt();
    }

    void corrupt() {
        this->errors++;
        this->out.clear();
        this->discarding = true;
    }

    /// @brief handles the frame at an end byte. complete is false when the frame ended
    /// in the middle of a stuffed sequence.
    void end(const bool complete, const Handler &handler) {
        if (!this->discarding) {
            if (!complete)
                this->errors++;
            else if (!this->out.empty())
                handler(this->out);
        }
        this->reset();
    }

    void reset() {
        this->out.clear();
        this->discarding = false;
    }
};

class COBS final : public Framer {
    Unstuffed u;
    /// @brief remaining is the number of data bytes left in the current block.
    std::size_t remaining = 0;
    /// @brief zero_pending is true when the next block starts after a zero.
    bool zero_pending = false;

public:
    x::errors::Error write(const View chunk, const Handler &handler) override {
        this->u.err = x::errors::NIL;
        for (const auto b: chunk) {
            if (b == 0) {
                this->u.end(this->remaining == 0, handler);
                this->remaining = 0;
                this->zero_pending = false;
                continue;
            }
            if (this->u.discarding) continue;
            if (this->remaining > 0) {
                this->u.push(b);
                this->remaining--;
                continue;
            }
            if (this->zero_pending) this->u.push(0);
            this->remaining = b - 1u;
            this->zero_pending = b != 0xFF;
        }
        return this->u.err;
    }

    [[nodiscard]] std::size_t errors() const override { return this->u.errors; }

    void reset() override {
        this->u.reset();
        this->remaining = 0;
        this->zero_pending = false;
    }

    x::errors::Error encode(const View frame, Bytes &out) const override {
        if (auto err = validate(frame, frame.size())) return err;
        auto code_at = out.size();
        out.push_back(0);
        std::uint8_t code = 1;
        for (const auto b: frame) {
            if (b != 0) {
                out.push_back(b);
                code++;
            }
            if (b == 0 || code == 0xFF) {
                out[code_at] = code;
                code_at = out.size();
                out.push_back(0);
                code = 1;
            }
        }
        out[code_at] = code;
        out.push_back(0);
        return x::errors::NIL;
    }
};

class SLIP final : public Framer {
    static constexpr std::uint8_t END = 0xC0;
    static constexpr std::uint8_t ESC = 0xDB;
    static constexpr std::uint8_t ESC_END = 0xDC;
    static constexpr std::uint8_t ESC_ESC = 0xDD;

    Unstuffed u;
    /// @brief escaped is true after an ESC byte.
    bool escaped = false;

public:
    x::errors::Error write(const View chunk, const Handler &handler) override {
        this->u.err = x::errors::NIL;
        for (const auto b: chunk) {
            if (b == END) {
                this->u.end(!this->escaped, handler);
                this->escaped = false;
                continue;
            }
            if (this->u.discarding) continue;
            if (!this->escaped) {
                if (b == ESC)
                    this->escaped = true;
                else
                    this->u.push(b);
                continue;
            }
            this->escaped = false;
            if (b == ESC_END)
                this->u.push(END);
            else if (b == ESC_ESC)
                this->u.push(ESC);
            else
                this->u.corrupt();
        }
        return this->u.err;
    }

    [[nodiscard]] std::size_t errors() const override { return this->u.errors; }

    void reset() override {
        this->u.reset();
        this->escaped = false;
    }

    x::errors::Error encode(const View frame, Bytes &out) const override {
        if (auto err = validate(frame, frame.size())) return err;
        // A leading END flushes any line noise the receiver has buffered.
        out.push_back(END);
        for (const auto b: frame) {
            if (b == END) {
                out.push_back(ESC);
                out.push_back(ESC_END);
            } else if (b == ESC) {
                out.push_back(ESC);
                out.push_back(ESC_ESC);
            } else
                out.push_back(b);
        }
        out.push_back(END);
        return x::errors::NIL;
    }
};

int hex_digit(const char c) {
    if (c >= '0' && c <= '9') return c - '0';
    if (c >= 'a' && c <= 'f') return c - 'a' + 10;
    if (c >= 'A' && c <= 'F') return c - 'A' + 10;
    return -1;
}

std::pair<Bytes, x::errors::Error> parse_hex(const std::string &hex) {
    if (hex.size() % 2 != 0)
        return {{}, config_error("sync " + hex + " has an odd number of hex digits")};
    Bytes out;
    for (std::size_t i = 0; i < hex.size(); i += 2) {
        const auto hi = hex_digit(hex[i]);
        const auto lo = hex_digit(hex[i + 1]);
        if (hi < 0 || lo < 0) return {{}, config_error("sync " + hex + " is not hex")};
        out.push_back(static_cast<std::uint8_t>(hi << 4 | lo));
    }
    return {std::move(out), x::errors::NIL};
}

std::pair<Checksum, x::errors::Error> parse_checksum(const std::string &name) {
    if (name == bus::CHECKSUM_NONE) return {Checksum::NONE, x::errors::NIL};
    if (name == bus::CHECKSUM_CRC_16_CCITT)
        return {Checksum::CRC16_CCITT_FALSE, x::errors::NIL};
    if (name == bus::CHECKSUM_CRC_16_MODBUS)
        return {Checksum::CRC16_MODBUS, x::errors::NIL};
    if (name == bus::CHECKSUM_CRC_32) return {Checksum::CRC32, x::errors::NIL};
    return {Checksum::NONE, config_error("unknown checksum " + name)};
}

/// @returns true for big-endian, false for little-endian.
std::pair<bool, x::errors::Error> parse_byte_order(const std::string &order) {
    if (order == library::BYTE_ORDER_BIG_ENDIAN) return {true, x::errors::NIL};
    if (order == library::BYTE_ORDER_LITTLE_ENDIAN) return {false, x::errors::NIL};
    return {false, config_error("unknown byte order " + order)};
}

using Result = std::pair<std::unique_ptr<Framer>, x::errors::Error>;

Result create(const bus::DelimiterFraming &f) {
    if (f.delimiter.empty()) return {nullptr, config_error("delimiter is empty")};
    return {
        std::make_unique<Delimiter>(Bytes(f.delimiter.begin(), f.delimiter.end())),
        x::errors::NIL,
    };
}

Result create(const bus::FixedFraming &f) {
    if (f.length == 0 || f.length > MAX_SIZE)
        return {
            nullptr,
            config_error("length must be from 1 to " + std::to_string(MAX_SIZE)),
        };
    return {std::make_unique<Fixed>(f.length), x::errors::NIL};
}

Result create(const bus::SyncFraming &f) {
    auto [sync, sync_err] = parse_hex(f.sync);
    if (sync_err) return {nullptr, sync_err};
    auto [sum, sum_err] = parse_checksum(f.checksum);
    if (sum_err) return {nullptr, sum_err};
    if (f.length_size != 1 && f.length_size != 2 && f.length_size != 4)
        return {nullptr, config_error("length size must be 1, 2, or 4 bytes")};
    auto [length_big, order_err] = parse_byte_order(f.byte_order);
    if (order_err) return {nullptr, order_err};
    auto [checksum_big, checksum_order_err] = parse_byte_order(f.checksum_byte_order);
    if (checksum_order_err) return {nullptr, checksum_order_err};
    if (f.length_offset < sync.size())
        return {nullptr, config_error("length field overlaps the sync sequence")};
    if (f.length_offset + f.length_size + width(sum) > MAX_SIZE)
        return {nullptr, config_error("length offset is beyond the maximum frame")};
    return {
        std::make_unique<Sync>(SyncLayout{
            .sync = std::move(sync),
            .length_offset = f.length_offset,
            .length_size = f.length_size,
            .length_big = length_big,
            .length_adjustment = f.length_adjustment,
            .checksum = sum,
            .checksum_big = checksum_big,
        }),
        x::errors::NIL,
    };
}

Result create(const bus::CobsFraming &) {
    return {std::make_unique<COBS>(), x::errors::NIL};
}

Result create(const bus::SlipFraming &) {
    return {std::make_unique<SLIP>(), x::errors::NIL};
}
}

std::pair<std::unique_ptr<Framer>, x::errors::Error>
create(const bus::Framing &framing) {
    return std::visit([](const auto &f) { return create(f); }, framing);
}
}
