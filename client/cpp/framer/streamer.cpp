// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// std
#include <string>

/// internal
#include "client/cpp/framer/framer.h"

const std::string STREAM_ENDPOINT = "/frame/stream";

using namespace synnax;

void StreamerConfig::to_proto(api::v1::FrameStreamerRequest &f) const {
    f.mutable_keys()->Add(channels.begin(), channels.end());
    f.set_downsample_factor(downsample_factor);
}

std::pair<Streamer, freighter::Error>
FrameClient::open_streamer(const StreamerConfig &config) const {
    auto [s, exc] = streamer_client->stream(STREAM_ENDPOINT);
    if (exc) return {Streamer(), exc};
    api::v1::FrameStreamerRequest req;
    config.to_proto(req);
    if (auto exc2 = s->send(req)) {
        s->close_send();
        auto [_, err] = s->receive();
        return {Streamer(std::move(s)), err};
    }
    auto [_, resExc] = s->receive();
    return {Streamer(std::move(s)), resExc};
}

Streamer::Streamer(std::unique_ptr<StreamerStream> stream) :
    stream(std::move(stream)) {
}

std::pair<Frame, freighter::Error> Streamer::read() const {
    this->assert_open();
    auto [fr, exc] = this->stream->receive();
    return {Frame(fr.frame()), exc};
}

void Streamer::close_send() const { this->stream->close_send(); }

freighter::Error Streamer::close() const {
    this->close_send();
    auto [_, err] = this->stream->receive();
    return err.skip(freighter::EOF_);
}

freighter::Error Streamer::set_channels(std::vector<ChannelKey> channels) const {
    this->assert_open();
    api::v1::FrameStreamerRequest req;
    req.mutable_keys()->Add(channels.begin(), channels.end());
    return this->stream->send(req);
}

void Streamer::assert_open() const {
    if (closed) throw std::runtime_error("streamer is closed");
}
