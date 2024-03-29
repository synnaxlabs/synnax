// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <string>

#include "client/cpp/synnax/framer/framer.h"

std::string STREAM_ENDPOINT = "/frame/stream";

using namespace synnax;

void StreamerConfig::toProto(api::v1::FrameStreamerRequest &f) const
{
    f.mutable_keys()->Add(channels.begin(), channels.end());
    f.set_start(start.value);
}

std::pair<Streamer, freighter::Error> FrameClient::openStreamer(const StreamerConfig &config) const
{
    auto [s, exc] = streamer_client->stream(STREAM_ENDPOINT);
    if (exc)
        return {Streamer(), exc};
    auto req = api::v1::FrameStreamerRequest();
    config.toProto(req);
    auto exc2 = s->send(req);
    return {Streamer(std::move(s)), exc2};
}

Streamer::Streamer(std::unique_ptr<StreamerStream> s) : stream(std::move(s)) {}

std::pair<Frame, freighter::Error> Streamer::read() const
{
    assertOpen();
    auto [fr, exc] = stream->receive();
    return {Frame(fr.frame()), exc};
}

freighter::Error Streamer::closeSend() const {
    return stream->closeSend();
}

freighter::Error Streamer::close() const
{
    auto exc = stream->closeSend();
    if (exc)
        return exc;
    auto [res, recExc] = stream->receive();
    if (recExc.type == freighter::EOF_.type)
        return freighter::NIL;
    return recExc;
}

freighter::Error Streamer::setChannels(std::vector<ChannelKey> channels) const
{
    assertOpen();
    auto req = api::v1::FrameStreamerRequest();
    req.mutable_keys()->Add(channels.begin(), channels.end());
    auto err = stream->send(req);
    return err;
}

void Streamer::assertOpen() const
{
    if (closed)
        throw std::runtime_error("streamer is closed");
}
