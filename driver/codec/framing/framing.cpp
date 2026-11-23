// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <cstddef>
#include <cstdint>
#include <memory>
#include <optional>
#include <span>
#include <string>
#include <utility>
#include <vector>

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
}

std::pair<std::unique_ptr<Framer>, x::errors::Error>
delimited(std::vector<std::uint8_t> delimiter) {
    if (delimiter.empty())
        return {nullptr, x::errors::Error(CONFIG_ERROR, "the delimiter is empty")};
    return {std::make_unique<Delimited>(std::move(delimiter)), x::errors::NIL};
}

std::pair<std::unique_ptr<Framer>, x::errors::Error> fixed(const std::size_t length) {
    if (length == 0 || length > MAX_SIZE)
        return {
            nullptr,
            x::errors::Error(
                CONFIG_ERROR,
                "the length must be from 1 to " + std::to_string(MAX_SIZE) + ", not " +
                    std::to_string(length)
            ),
        };
    return {std::make_unique<Fixed>(length), x::errors::NIL};
}

std::unique_ptr<Framer> cobs() {
    return std::make_unique<Cobs>();
}

std::unique_ptr<Framer> slip() {
    return std::make_unique<Slip>();
}
}
