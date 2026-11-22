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

class Delimited final : public Framer {
    std::vector<std::uint8_t> delimiter;
    Buffer buf;
    /// @brief scanned is the offset in buf where the next delimiter search starts.
    std::size_t scanned = 0;
    /// @brief discarding is true while the framer drops the rest of a frame that is
    /// longer than MAX_SIZE.
    bool discarding = false;

    static x::errors::Error too_long() {
        return invalid(
            "dropped a frame longer than " + std::to_string(MAX_SIZE) + " bytes"
        );
    }

public:
    explicit Delimited(std::vector<std::uint8_t> delimiter):
        delimiter(std::move(delimiter)) {}

    void write(const View chunk) override { this->scanned -= this->buf.append(chunk); }

    std::pair<std::optional<View>, x::errors::Error> next() override {
        auto &bytes = this->buf.bytes;
        const auto &d = this->delimiter;
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
                if (this->scanned - this->buf.head <= MAX_SIZE)
                    return {std::nullopt, x::errors::NIL};
                this->buf.head = this->scanned;
                if (this->discarding) return {std::nullopt, x::errors::NIL};
                this->discarding = true;
                return {std::nullopt, too_long()};
            }
            const auto end = static_cast<std::size_t>(it - bytes.begin());
            const View frame(bytes.data() + this->buf.head, end - this->buf.head);
            this->buf.head = end + d.size();
            this->scanned = this->buf.head;
            if (this->discarding) {
                this->discarding = false;
                continue;
            }
            if (frame.size() > MAX_SIZE) return {std::nullopt, too_long()};
            if (!frame.empty()) return {frame, x::errors::NIL};
        }
    }

    x::errors::Error idle() override { return x::errors::NIL; }

    void clear() override {
        this->buf.clear();
        this->scanned = 0;
        this->discarding = false;
    }

    x::errors::Error
    encode(const View frame, std::vector<std::uint8_t> &out) const override {
        if (auto err = check_size(frame)) return err;
        const auto start = out.size();
        out.insert(out.end(), frame.begin(), frame.end());
        out.insert(out.end(), this->delimiter.begin(), this->delimiter.end());
        // A delimiter that starts inside the frame, even one that ends in the appended
        // delimiter, splits the frame on read.
        const auto begin = out.begin() + static_cast<std::ptrdiff_t>(start);
        const auto first = std::search(
            begin,
            out.end(),
            this->delimiter.begin(),
            this->delimiter.end()
        );
        if (static_cast<std::size_t>(first - begin) != frame.size()) {
            out.resize(start);
            return invalid("the frame holds the delimiter");
        }
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
}
