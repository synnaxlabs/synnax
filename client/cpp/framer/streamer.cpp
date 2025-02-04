// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <string>

#include "client/cpp/framer/framer.h"

std::string STREAM_ENDPOINT = "/frame/stream";

using namespace synnax;

void StreamerConfig::toProto(api::v1::FrameStreamerRequest &f) const {
    f.mutable_keys()->Add(channels.begin(), channels.end());
    f.set_downsample_factor(downsample_factor);
}

std::pair<Streamer, freighter::Error> FrameClient::openStreamer(
    const StreamerConfig &config
) const {
    auto [s, exc] = streamer_client->stream(STREAM_ENDPOINT);
    if (exc)
        return {Streamer(), exc};
    auto req = api::v1::FrameStreamerRequest();
    config.toProto(req);
    auto exc2 = s->send(req);
    if (exc2) return {Streamer(std::move(s)), exc2};
    auto [_, resExc] = s->receive();
    return {Streamer(std::move(s)), resExc};
}

Streamer::Streamer(std::unique_ptr<StreamerStream> s) : stream(std::move(s)) {
}

std::pair<Frame, freighter::Error> Streamer::read() const {
    assertOpen();
    auto [fr, exc] = stream->receive();
    return {Frame(fr.frame()), exc};
}

void Streamer::closeSend() const { stream->closeSend(); }

freighter::Error Streamer::close() const {
    closeSend();
    auto [res, err] = stream->receive();
    if (err.matches(freighter::EOF_)) return freighter::NIL;
    return err;
}

freighter::Error Streamer::setChannels(std::vector<ChannelKey> channels) const {
    assertOpen();
    auto req = api::v1::FrameStreamerRequest();
    req.mutable_keys()->Add(channels.begin(), channels.end());
    return stream->send(req);
}

void Streamer::assertOpen() const {
    if (closed)
        throw std::runtime_error("streamer is closed");
}
