// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <array>
#include <cstddef>
#include <cstdint>
#include <memory>
#include <optional>
#include <span>
#include <string>
#include <type_traits>
#include <utility>
#include <variant>
#include <vector>

#include "client/cpp/icd/types.gen.h"
#include "x/cpp/errors/errors.h"

#include "driver/codec/framing/framing.h"

namespace driver::codec::framing {
namespace {
using View = std::span<const std::uint8_t>;

x::errors::Error invalid(const std::string &text) {
    return x::errors::Error(x::errors::VALIDATION, text);
}

/// @brief returns the error for a frame that encode cannot put on the stream, or NIL.
x::errors::Error check_size(const View frame) {
    if (frame.empty()) return invalid("the frame is empty");
    if (frame.size() > MAX_SIZE)
        return invalid(
            "the frame has " + std::to_string(frame.size()) +
            " bytes, more than the maximum of " + std::to_string(MAX_SIZE)
        );
    return x::errors::NIL;
}

/// @brief Buffer holds the written bytes of a stream that no frame has read yet.
struct Buffer {
    std::vector<std::uint8_t> bytes;
    /// @brief head is the offset of the first byte no frame has read.
    std::size_t head = 0;

    Buffer() { this->bytes.reserve(MAX_SIZE); }

    /// @brief drops the bytes before head, then appends chunk. It returns how many
    /// bytes it dropped, so offsets into bytes can move with them.
    std::size_t append(const View chunk) {
        const auto dropped = this->head;
        this->bytes.erase(
            this->bytes.begin(),
            this->bytes.begin() + static_cast<std::ptrdiff_t>(dropped)
        );
        this->head = 0;
        this->bytes.insert(this->bytes.end(), chunk.begin(), chunk.end());
        return dropped;
    }

    [[nodiscard]] std::size_t pending() const {
        return this->bytes.size() - this->head;
    }

    void clear() {
        this->bytes.clear();
        this->head = 0;
    }
};

x::errors::Error too_long() {
    return invalid(
        "dropped a frame longer than " + std::to_string(MAX_SIZE) + " bytes"
    );
}

/// @brief Splitter finds the frames that a delimiter ends in a stream. Its frames are
/// raw: a framer that stuffs bytes decodes them in place.
class Splitter {
    std::vector<std::uint8_t> delim;
    /// @brief max is the most raw bytes that a frame of MAX_SIZE bytes takes.
    std::size_t max;
    Buffer buf;
    /// @brief scanned is the offset in buf where the next delimiter search starts.
    std::size_t scanned = 0;
    /// @brief discarding is true while the splitter drops the rest of a frame that is
    /// longer than max.
    bool discarding = false;

public:
    Splitter(std::vector<std::uint8_t> delimiter, const std::size_t max):
        delim(std::move(delimiter)), max(max) {}

    [[nodiscard]] const std::vector<std::uint8_t> &delimiter() const {
        return this->delim;
    }

    void write(const View chunk) { this->scanned -= this->buf.append(chunk); }

    /// @brief returns the next raw frame that is not empty, valid until the next call
    /// to write or clear.
    std::pair<std::optional<std::span<std::uint8_t>>, x::errors::Error> next() {
        auto &bytes = this->buf.bytes;
        const auto &d = this->delim;
        while (true) {
            const auto it = std::search(
                bytes.begin() + static_cast<std::ptrdiff_t>(this->scanned),
                bytes.end(),
                d.begin(),
                d.end()
            );
            if (it == bytes.end()) {
                // The last bytes can start a delimiter that the next chunk ends.
                const auto partial = std::min(this->buf.pending(), d.size() - 1);
                this->scanned = bytes.size() - partial;
                if (this->scanned - this->buf.head <= this->max)
                    return {std::nullopt, x::errors::NIL};
                this->buf.head = this->scanned;
                if (this->discarding) return {std::nullopt, x::errors::NIL};
                this->discarding = true;
                return {std::nullopt, too_long()};
            }
            const auto end = static_cast<std::size_t>(it - bytes.begin());
            const std::span frame(bytes.data() + this->buf.head, end - this->buf.head);
            this->buf.head = end + d.size();
            this->scanned = this->buf.head;
            if (this->discarding) {
                this->discarding = false;
                continue;
            }
            if (frame.size() > this->max) return {std::nullopt, too_long()};
            if (!frame.empty()) return {frame, x::errors::NIL};
        }
    }

    void clear() {
        this->buf.clear();
        this->scanned = 0;
        this->discarding = false;
    }
};

class Delimited final : public Framer {
    Splitter splitter;

public:
    explicit Delimited(std::vector<std::uint8_t> delimiter):
        splitter(std::move(delimiter), MAX_SIZE) {}

    void write(const View chunk) override { this->splitter.write(chunk); }

    std::pair<std::optional<View>, x::errors::Error> next() override {
        auto [frame, err] = this->splitter.next();
        if (!frame.has_value()) return {std::nullopt, err};
        return {View(*frame), x::errors::NIL};
    }

    x::errors::Error idle() override { return x::errors::NIL; }

    void clear() override { this->splitter.clear(); }

    x::errors::Error
    encode(const View frame, std::vector<std::uint8_t> &out) const override {
        if (auto err = check_size(frame)) return err;
        const auto &d = this->splitter.delimiter();
        const auto start = out.size();
        out.insert(out.end(), frame.begin(), frame.end());
        out.insert(out.end(), d.begin(), d.end());
        // A delimiter that starts inside the frame, even one that ends in the appended
        // delimiter, splits the frame on read.
        const auto begin = out.begin() + static_cast<std::ptrdiff_t>(start);
        const auto first = std::search(begin, out.end(), d.begin(), d.end());
        if (static_cast<std::size_t>(first - begin) != frame.size()) {
            out.resize(start);
            return invalid("the frame holds the delimiter");
        }
        return x::errors::NIL;
    }
};

constexpr std::uint8_t COBS_DELIMITER = 0x00;
/// @brief COBS_BLOCK is the most bytes that one COBS code byte covers.
constexpr std::size_t COBS_BLOCK = 254;

class Cobs final : public Framer {
    Splitter splitter{{COBS_DELIMITER}, MAX_SIZE + MAX_SIZE / COBS_BLOCK + 1};

    /// @brief decodes raw in place and returns its decoded bytes, or nothing when raw
    /// is not valid COBS.
    static std::optional<View> decode(const std::span<std::uint8_t> raw) {
        std::size_t r = 0;
        std::size_t w = 0;
        while (r < raw.size()) {
            const std::size_t code = raw[r++];
            const auto length = code - 1;
            if (length > raw.size() - r) return std::nullopt;
            std::copy_n(
                raw.begin() + static_cast<std::ptrdiff_t>(r),
                length,
                raw.begin() + static_cast<std::ptrdiff_t>(w)
            );
            r += length;
            w += length;
            if (code != COBS_BLOCK + 1 && r < raw.size()) raw[w++] = COBS_DELIMITER;
        }
        return View(raw.data(), w);
    }

public:
    void write(const View chunk) override { this->splitter.write(chunk); }

    std::pair<std::optional<View>, x::errors::Error> next() override {
        while (true) {
            const auto [raw, err] = this->splitter.next();
            if (!raw.has_value()) return {std::nullopt, err};
            const auto frame = decode(*raw);
            if (!frame.has_value())
                return {
                    std::nullopt,
                    invalid("dropped a frame with invalid COBS bytes")
                };
            if (frame->size() > MAX_SIZE) return {std::nullopt, too_long()};
            if (!frame->empty()) return {frame, x::errors::NIL};
        }
    }

    x::errors::Error idle() override { return x::errors::NIL; }

    void clear() override { this->splitter.clear(); }

    x::errors::Error
    encode(const View frame, std::vector<std::uint8_t> &out) const override {
        if (auto err = check_size(frame)) return err;
        auto code_at = out.size();
        out.push_back(0);
        std::uint8_t code = 1;
        const auto close = [&] {
            out[code_at] = code;
            code_at = out.size();
            out.push_back(0);
            code = 1;
        };
        for (std::size_t i = 0; i < frame.size(); i++) {
            if (frame[i] == COBS_DELIMITER) {
                close();
                continue;
            }
            out.push_back(frame[i]);
            if (++code == COBS_BLOCK + 1 && i + 1 < frame.size()) close();
        }
        out[code_at] = code;
        out.push_back(COBS_DELIMITER);
        return x::errors::NIL;
    }
};

constexpr std::uint8_t SLIP_END = 0xC0;
constexpr std::uint8_t SLIP_ESC = 0xDB;
constexpr std::uint8_t SLIP_ESC_END = 0xDC;
constexpr std::uint8_t SLIP_ESC_ESC = 0xDD;

class Slip final : public Framer {
    Splitter splitter{{SLIP_END}, 2 * MAX_SIZE};

    /// @brief decodes raw in place and returns its decoded bytes, or nothing when raw
    /// holds an invalid escape.
    static std::optional<View> decode(const std::span<std::uint8_t> raw) {
        std::size_t w = 0;
        for (std::size_t r = 0; r < raw.size(); r++) {
            if (raw[r] != SLIP_ESC) {
                raw[w++] = raw[r];
                continue;
            }
            if (++r == raw.size()) return std::nullopt;
            if (raw[r] == SLIP_ESC_END)
                raw[w++] = SLIP_END;
            else if (raw[r] == SLIP_ESC_ESC)
                raw[w++] = SLIP_ESC;
            else
                return std::nullopt;
        }
        return View(raw.data(), w);
    }

public:
    void write(const View chunk) override { this->splitter.write(chunk); }

    std::pair<std::optional<View>, x::errors::Error> next() override {
        const auto [raw, err] = this->splitter.next();
        if (!raw.has_value()) return {std::nullopt, err};
        const auto frame = decode(*raw);
        if (!frame.has_value())
            return {
                std::nullopt,
                invalid("dropped a frame with an invalid SLIP escape")
            };
        if (frame->size() > MAX_SIZE) return {std::nullopt, too_long()};
        return {frame, x::errors::NIL};
    }

    x::errors::Error idle() override { return x::errors::NIL; }

    void clear() override { this->splitter.clear(); }

    /// @brief starts the frame with an END byte as well, which ends any line noise
    /// before it.
    x::errors::Error
    encode(const View frame, std::vector<std::uint8_t> &out) const override {
        if (auto err = check_size(frame)) return err;
        out.push_back(SLIP_END);
        for (const auto b: frame) {
            if (b == SLIP_END) {
                out.push_back(SLIP_ESC);
                out.push_back(SLIP_ESC_END);
            } else if (b == SLIP_ESC) {
                out.push_back(SLIP_ESC);
                out.push_back(SLIP_ESC_ESC);
            } else
                out.push_back(b);
        }
        out.push_back(SLIP_END);
        return x::errors::NIL;
    }
};

class Fixed final : public Framer {
    std::size_t length;
    Buffer buf;

public:
    explicit Fixed(const std::size_t length): length(length) {}

    void write(const View chunk) override { this->buf.append(chunk); }

    std::pair<std::optional<View>, x::errors::Error> next() override {
        if (this->buf.pending() < this->length) return {std::nullopt, x::errors::NIL};
        const View frame(this->buf.bytes.data() + this->buf.head, this->length);
        this->buf.head += this->length;
        return {frame, x::errors::NIL};
    }

    x::errors::Error idle() override {
        const auto dropped = this->buf.pending();
        this->buf.clear();
        if (dropped == 0) return x::errors::NIL;
        return invalid(
            "dropped " + std::to_string(dropped) + " bytes of a partial frame"
        );
    }

    void clear() override { this->buf.clear(); }

    x::errors::Error
    encode(const View frame, std::vector<std::uint8_t> &out) const override {
        if (frame.size() != this->length)
            return invalid(
                "the frame has " + std::to_string(frame.size()) +
                " bytes, not the fixed length of " + std::to_string(this->length)
            );
        out.insert(out.end(), frame.begin(), frame.end());
        return x::errors::NIL;
    }
};

/// @brief returns the table of a CRC whose polynomial is not reflected.
template<typename T>
constexpr std::array<T, 256> table(const T poly) {
    constexpr int shift = (sizeof(T) * 8) - 8;
    constexpr T top = T{1} << ((sizeof(T) * 8) - 1);
    std::array<T, 256> t{};
    for (unsigned i = 0; i < t.size(); i++) {
        auto c = static_cast<T>(i << shift);
        for (int b = 0; b < 8; b++)
            c = (c & top) != 0 ? static_cast<T>((c << 1) ^ poly)
                               : static_cast<T>(c << 1);
        t[i] = c;
    }
    return t;
}

/// @brief returns the table of a CRC whose polynomial is reflected.
template<typename T>
constexpr std::array<T, 256> reflected_table(const T poly) {
    std::array<T, 256> t{};
    for (unsigned i = 0; i < t.size(); i++) {
        auto c = static_cast<T>(i);
        for (int b = 0; b < 8; b++)
            c = (c & 1) != 0 ? static_cast<T>((c >> 1) ^ poly) : static_cast<T>(c >> 1);
        t[i] = c;
    }
    return t;
}

constexpr auto CCITT_TABLE = table<std::uint16_t>(0x1021);
constexpr auto MODBUS_TABLE = reflected_table<std::uint16_t>(0xA001);
constexpr auto CRC32_TABLE = reflected_table<std::uint32_t>(0xEDB88320);

std::uint32_t crc16_ccitt_false(const View data) {
    std::uint16_t c = 0xFFFF;
    for (const auto b: data)
        c = static_cast<std::uint16_t>((c << 8) ^ CCITT_TABLE[((c >> 8) ^ b) & 0xFF]);
    return c;
}

std::uint32_t crc16_modbus(const View data) {
    std::uint16_t c = 0xFFFF;
    for (const auto b: data)
        c = static_cast<std::uint16_t>((c >> 8) ^ MODBUS_TABLE[(c ^ b) & 0xFF]);
    return c;
}

std::uint32_t crc32(const View data) {
    std::uint32_t c = 0xFFFFFFFF;
    for (const auto b: data)
        c = (c >> 8) ^ CRC32_TABLE[(c ^ b) & 0xFF];
    return ~c;
}

std::uint64_t read_uint(const std::uint8_t *p, const std::size_t size, const bool big) {
    std::uint64_t v = 0;
    for (std::size_t i = 0; i < size; i++)
        v = (v << 8) | p[big ? i : size - 1 - i];
    return v;
}

void write_uint(
    std::uint8_t *p,
    const std::uint64_t v,
    const std::size_t size,
    const bool big
) {
    for (std::size_t i = 0; i < size; i++)
        p[big ? size - 1 - i : i] = static_cast<std::uint8_t>(v >> (8 * i));
}

/// @brief Crc is the checksum that ends a sync frame.
struct Crc {
    /// @brief sum returns the checksum of the bytes before it.
    std::uint32_t (*sum)(View) = nullptr;
    /// @brief width is the number of bytes in the checksum.
    std::size_t width = 0;
    /// @brief big_endian is true when the checksum puts its most significant byte
    /// first.
    bool big_endian = false;
};

class Sync final : public Framer {
    synnax::icd::SyncFraming cfg;
    /// @brief length_size is the number of bytes in the length field.
    std::size_t length_size;
    /// @brief big_endian is true when the length field puts its most significant byte
    /// first.
    bool big_endian;
    /// @brief crc is the checksum that ends each frame, or nothing.
    std::optional<Crc> crc;
    /// @brief header is the number of bytes up to the end of the length field.
    std::size_t header;
    /// @brief width is the number of bytes in the checksum.
    std::size_t width;
    Buffer buf;

public:
    Sync(
        synnax::icd::SyncFraming config,
        const std::size_t size,
        const bool big,
        const std::optional<Crc> checksum
    ):
        cfg(std::move(config)),
        length_size(size),
        big_endian(big),
        crc(checksum),
        header(this->cfg.length_offset + size),
        width(checksum.has_value() ? checksum->width : 0) {}

    void write(const View chunk) override { this->buf.append(chunk); }

    std::pair<std::optional<View>, x::errors::Error> next() override {
        auto &bytes = this->buf.bytes;
        const auto &s = this->cfg.sync;
        const auto begin = bytes.begin() + static_cast<std::ptrdiff_t>(this->buf.head);
        const auto found = std::search(begin, bytes.end(), s.begin(), s.end());
        if (found != begin) {
            // The last bytes can start a sync sequence that the next chunk ends.
            const auto start = found != bytes.end()
                                 ? static_cast<std::size_t>(found - bytes.begin())
                                 : bytes.size() -
                                       std::min(this->buf.pending(), s.size() - 1);
            const auto dropped = start - this->buf.head;
            this->buf.head = start;
            if (dropped == 0) return {std::nullopt, x::errors::NIL};
            return {
                std::nullopt,
                invalid(
                    "dropped " + std::to_string(dropped) + " bytes that are not a frame"
                ),
            };
        }
        if (this->buf.pending() < this->header) return {std::nullopt, x::errors::NIL};
        const auto *p = bytes.data() + this->buf.head;
        const auto length = read_uint(
            p + this->cfg.length_offset,
            this->length_size,
            this->big_endian
        );
        const auto after = static_cast<std::int64_t>(length) +
                           this->cfg.length_adjustment;
        if (std::cmp_less(after, this->width) ||
            this->header + static_cast<std::size_t>(after) - this->width >
                this->cfg.max_length) {
            this->buf.head++;
            return {
                std::nullopt,
                invalid(
                    "dropped a frame with an invalid length of " +
                    std::to_string(length)
                ),
            };
        }
        const auto total = this->header + static_cast<std::size_t>(after);
        if (this->buf.pending() < total) return {std::nullopt, x::errors::NIL};
        const View frame(p, total - this->width);
        if (const auto &c = this->crc; c.has_value()) {
            const auto stored = read_uint(p + frame.size(), this->width, c->big_endian);
            if (stored != c->sum(frame)) {
                this->buf.head++;
                return {std::nullopt, invalid("dropped a frame with a bad checksum")};
            }
        }
        this->buf.head += total;
        return {frame, x::errors::NIL};
    }

    x::errors::Error idle() override {
        const auto pending = this->buf.pending();
        if (pending == 0) return x::errors::NIL;
        auto &bytes = this->buf.bytes;
        const auto &s = this->cfg.sync;
        const auto from = bytes.begin() + static_cast<std::ptrdiff_t>(this->buf.head) +
                          1;
        const auto found = std::search(from, bytes.end(), s.begin(), s.end());
        const auto start = static_cast<std::size_t>(found - bytes.begin());
        const auto dropped = start - this->buf.head;
        this->buf.head = start;
        return invalid(
            "dropped " + std::to_string(dropped) + " bytes of a partial frame"
        );
    }

    void clear() override { this->buf.clear(); }

    x::errors::Error
    encode(const View frame, std::vector<std::uint8_t> &out) const override {
        if (frame.size() < this->header)
            return invalid(
                "the frame has " + std::to_string(frame.size()) +
                " bytes, fewer than the " + std::to_string(this->header) +
                " bytes of its header"
            );
        if (frame.size() > this->cfg.max_length)
            return invalid(
                "the frame has " + std::to_string(frame.size()) +
                " bytes, more than the maximum length of " +
                std::to_string(this->cfg.max_length)
            );
        const auto total = frame.size() + this->width;
        const auto length = static_cast<std::int64_t>(total - this->header) -
                            this->cfg.length_adjustment;
        const auto bits = 8 * this->length_size;
        if (length < 0 || static_cast<std::uint64_t>(length) >> bits != 0)
            return invalid(
                "the length " + std::to_string(length) + " does not fit a " +
                std::to_string(this->length_size) + "-byte length field"
            );
        const auto start = out.size();
        out.insert(out.end(), frame.begin(), frame.end());
        auto *p = out.data() + start;
        std::ranges::copy(this->cfg.sync, p);
        write_uint(
            p + this->cfg.length_offset,
            static_cast<std::uint64_t>(length),
            this->length_size,
            this->big_endian
        );
        if (const auto &c = this->crc; c.has_value()) {
            const auto sum = c->sum(View(p, frame.size()));
            out.resize(out.size() + this->width);
            write_uint(
                out.data() + start + frame.size(),
                sum,
                this->width,
                c->big_endian
            );
        }
        return x::errors::NIL;
    }
};

using Result = std::pair<std::unique_ptr<Framer>, x::errors::Error>;

Result fail(const std::string &text) {
    return {nullptr, x::errors::Error(CONFIG_ERROR, text)};
}

/// @brief returns true when order is big-endian.
std::pair<bool, x::errors::Error> big_endian_of(const std::string &order) {
    if (order == synnax::icd::BYTE_ORDER_BIG_ENDIAN) return {true, x::errors::NIL};
    if (order == synnax::icd::BYTE_ORDER_LITTLE_ENDIAN) return {false, x::errors::NIL};
    return {false, x::errors::Error(CONFIG_ERROR, "unknown byte order " + order)};
}

std::pair<Crc, x::errors::Error> crc_of(const synnax::icd::Checksum &c) {
    auto [big, err] = big_endian_of(c.byte_order);
    if (err) return {Crc{}, err};
    if (c.algorithm == synnax::icd::CHECKSUM_ALGORITHM_CRC_16_CCITT_FALSE)
        return {
            Crc{.sum = crc16_ccitt_false, .width = 2, .big_endian = big},
            x::errors::NIL
        };
    if (c.algorithm == synnax::icd::CHECKSUM_ALGORITHM_CRC_16_MODBUS)
        return {
            Crc{.sum = crc16_modbus, .width = 2, .big_endian = big},
            x::errors::NIL
        };
    if (c.algorithm == synnax::icd::CHECKSUM_ALGORITHM_CRC_32)
        return {Crc{.sum = crc32, .width = 4, .big_endian = big}, x::errors::NIL};
    return {
        Crc{},
        x::errors::Error(CONFIG_ERROR, "unknown checksum algorithm " + c.algorithm),
    };
}

std::pair<std::size_t, x::errors::Error> length_bytes(const std::string &size) {
    if (size == synnax::icd::LENGTH_SIZE_ONE) return {1, x::errors::NIL};
    if (size == synnax::icd::LENGTH_SIZE_TWO) return {2, x::errors::NIL};
    if (size == synnax::icd::LENGTH_SIZE_FOUR) return {4, x::errors::NIL};
    return {0, x::errors::Error(CONFIG_ERROR, "unknown length size " + size)};
}

Result sync(const synnax::icd::SyncFraming &s) {
    auto [size, size_err] = length_bytes(s.length_size);
    if (size_err) return {nullptr, size_err};
    auto [big, order_err] = big_endian_of(s.byte_order);
    if (order_err) return {nullptr, order_err};
    std::optional<Crc> crc;
    if (s.checksum.has_value()) {
        auto [c, err] = crc_of(*s.checksum);
        if (err) return {nullptr, err};
        crc = c;
    }
    if (s.length_offset < s.sync.size())
        return fail("the length field overlaps the sync sequence");
    const auto header = s.length_offset + size;
    if (s.max_length < header)
        return fail(
            "the maximum length of " + std::to_string(s.max_length) +
            " is less than the " + std::to_string(header) + " bytes of the header"
        );
    return {std::make_unique<Sync>(s, size, big, crc), x::errors::NIL};
}
}

std::pair<std::unique_ptr<Framer>, x::errors::Error>
create(const synnax::icd::Framing &framing) {
    return std::visit(
        [](const auto &f) -> Result {
            using T = std::decay_t<decltype(f)>;
            if constexpr (std::is_same_v<T, synnax::icd::DelimitedFraming>)
                return {
                    std::make_unique<Delimited>(std::vector<std::uint8_t>(
                        f.delimiter.begin(),
                        f.delimiter.end()
                    )),
                    x::errors::NIL,
                };
            else if constexpr (std::is_same_v<T, synnax::icd::FixedFraming>)
                return {std::make_unique<Fixed>(f.length), x::errors::NIL};
            else if constexpr (std::is_same_v<T, synnax::icd::SyncFraming>)
                return sync(f);
            else if constexpr (std::is_same_v<T, synnax::icd::CobsFraming>)
                return {std::make_unique<Cobs>(), x::errors::NIL};
            else {
                static_assert(std::is_same_v<T, synnax::icd::SlipFraming>);
                return {std::make_unique<Slip>(), x::errors::NIL};
            }
        },
        framing
    );
}
}
