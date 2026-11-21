// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <string>

#include "client/cpp/framer/framer.h"

namespace synnax::framer {
void StreamerConfig::to_proto(grpc::framer::StreamerRequest &f) const {
    f.mutable_keys()->Add(channels.begin(), channels.end());
    f.set_downsample_factor(downsample_factor);
    f.mutable_exclude_groups()->Add(
        this->exclude_groups.begin(),
        this->exclude_groups.end()
    );
}

std::pair<Streamer, x::errors::Error>
Client::open_streamer(const StreamerConfig &config) const {
    auto [net_stream, err] = streamer_client->stream("/frame/stream");
    if (err) return {Streamer(), err};
    grpc::framer::StreamerRequest req;
    config.to_proto(req);
    if (!net_stream->send(req).ok()) net_stream->close_send();
    Codec codec(this->channel_client);
    if (const auto codec_err = codec.update(config.channels))
        return {Streamer(), codec_err};
    // The open ack is the first frame of the codec's state, and it declares the types.
    auto [ack, res_err] = net_stream->receive();
    if (!res_err && !ack.buffer().empty())
        if (auto [_, decode_err] = codec.decode(
                reinterpret_cast<const std::uint8_t *>(ack.buffer().data()),
                ack.buffer().size()
            );
            decode_err)
            return {Streamer(), decode_err};
    auto streamer = Streamer(std::move(net_stream), config);
    streamer.codec = std::move(codec);
    return {std::move(streamer), res_err};
}

Streamer::Streamer(std::unique_ptr<StreamerStream> stream, StreamerConfig config):
    cfg(std::move(config)), stream(std::move(stream)) {}

std::pair<x::telem::Frame, x::errors::Error> Streamer::read() {
    this->assert_open();
    auto [fr, exc] = this->stream->receive();
    if (!fr.buffer().empty())
        return this->codec.decode(
            reinterpret_cast<const std::uint8_t *>(fr.buffer().data()),
            fr.buffer().size()
        );
    if (exc) return {x::telem::Frame(), exc};
    auto [frame, proto_err] = x::telem::Frame::from_proto(fr.frame());
    if (proto_err) return {x::telem::Frame(), proto_err};
    return {std::move(frame), x::errors::NIL};
}

void Streamer::close_send() const {
    this->stream->close_send();
}

x::errors::Error Streamer::close() const {
    this->close_send();
    while (true)
        if (auto [_, err] = this->stream->receive(); err)
            return err.skip(freighter::ERR_EOF);
}

x::errors::Error Streamer::set_channels(const std::vector<channel::Key> &channels) {
    this->assert_open();
    if (const auto err = this->codec.update(channels)) return err;
    this->cfg.channels = channels;
    grpc::framer::StreamerRequest req;
    this->cfg.to_proto(req);
    return this->stream->send(req);
}

void Streamer::assert_open() const {
    if (closed) throw std::runtime_error("streamer is closed");
}
}
