// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "client/cpp/framer/framer.h"

namespace synnax::framer {
namespace {
enum Command : std::int32_t {
    OPEN = 0,
    NEXT = 1,
    PREV = 2,
    SEEK_FIRST = 3,
    SEEK_LAST = 4,
    SEEK_LE = 5,
    SEEK_GE = 6,
    VALID = 7,
};

constexpr std::int32_t ACK_RESPONSE = 1;

grpc::framer::IteratorRequest request(const Command command) {
    grpc::framer::IteratorRequest req;
    req.set_command(command);
    return req;
}

grpc::framer::IteratorRequest
read_request(const Command command, const x::telem::TimeSpan span) {
    auto req = request(command);
    req.set_span(span.nanoseconds());
    return req;
}

grpc::framer::IteratorRequest
seek_request(const Command command, const x::telem::TimeStamp stamp) {
    auto req = request(command);
    req.set_stamp(stamp.nanoseconds());
    return req;
}
}

std::pair<Iterator, x::errors::Error>
Client::open_iterator(const IteratorConfig &config) const {
    auto codec = Codec(this->channel_client);
    if (const auto err = codec.update(config.channels)) return {Iterator(), err};
    auto [range, range_err] = config.bounds.to_proto();
    if (range_err) return {Iterator(), range_err};
    auto [net_stream, err] = this->iterator_client->stream("/frame/iterate");
    if (err) return {Iterator(), err};
    auto iter = Iterator(std::move(net_stream), std::move(codec));
    auto req = request(OPEN);
    req.mutable_keys()->Add(config.channels.begin(), config.channels.end());
    *req.mutable_range() = range;
    req.set_chunk_size(config.chunk_size);
    if (auto [_, open_err] = iter.exec(req); open_err) return {Iterator(), open_err};
    return {std::move(iter), x::errors::NIL};
}

Iterator::Iterator(std::unique_ptr<IteratorStream> stream, Codec codec):
    codec(std::move(codec)), stream(std::move(stream)) {}

std::pair<bool, x::errors::Error> Iterator::next(const x::telem::TimeSpan span) {
    return this->exec(read_request(NEXT, span));
}

std::pair<bool, x::errors::Error> Iterator::prev(const x::telem::TimeSpan span) {
    return this->exec(read_request(PREV, span));
}

std::pair<bool, x::errors::Error> Iterator::seek_first() {
    return this->exec(request(SEEK_FIRST));
}

std::pair<bool, x::errors::Error> Iterator::seek_last() {
    return this->exec(request(SEEK_LAST));
}

std::pair<bool, x::errors::Error> Iterator::seek_le(const x::telem::TimeStamp stamp) {
    return this->exec(seek_request(SEEK_LE, stamp));
}

std::pair<bool, x::errors::Error> Iterator::seek_ge(const x::telem::TimeStamp stamp) {
    return this->exec(seek_request(SEEK_GE, stamp));
}

std::pair<bool, x::errors::Error> Iterator::valid() {
    return this->exec(request(VALID));
}

x::errors::Error Iterator::close() const {
    this->stream->close_send();
    while (true) {
        auto [res, err] = this->stream->receive();
        if (err) return err.skip(freighter::ERR_EOF);
        if (auto res_err = x::errors::Error(res.error())) return res_err;
    }
}

std::pair<bool, x::errors::Error> Iterator::exec(grpc::framer::IteratorRequest req) {
    // A validity check moves nothing, so value keeps the telemetry of the last read.
    if (req.command() != VALID) this->value = x::telem::Frame();
    if (auto err = this->stream->send(req)) return {false, err};
    while (true) {
        auto [res, err] = this->stream->receive();
        if (err) return {false, err};
        if (res.variant() == ACK_RESPONSE)
            return {res.ack(), x::errors::Error(res.error())};
        x::telem::Frame fr;
        if (!res.buffer().empty()) {
            auto [decoded, decode_err] = this->codec.decode(
                reinterpret_cast<const std::uint8_t *>(res.buffer().data()),
                res.buffer().size()
            );
            if (decode_err) return {false, decode_err};
            fr = std::move(decoded);
        } else {
            auto [parsed, parse_err] = x::telem::Frame::from_proto(res.frame());
            if (parse_err) return {false, parse_err};
            fr = std::move(parsed);
        }
        for (auto [key, series]: fr)
            this->value.emplace(key, std::move(series));
    }
}
}
