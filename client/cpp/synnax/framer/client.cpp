// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "synnax/framer/framer.h"

using namespace synnax;

const std::string ITERATOR_ENDPOINT = "/frame/iterate";
const std::string STREAM_ENDPOINT = "/frame/stream";
const std::string WRITE_ENDPOINT = "/frame/write";

std::pair<Iterator, freighter::Error> FrameClient::openIterator(const IteratorConfig &config) {
    auto [s, exc] =  iterator_client->stream(ITERATOR_ENDPOINT);
    return {Iterator(s, config), exc};
}

std::pair<Streamer, freighter::Error> FrameClient::openStreamer(const StreamerConfig &config) {
    auto [s, exc]  = streamer_client->stream(STREAM_ENDPOINT);
    return {Streamer(s, config), exc};
}

std::pair<Writer, freighter::Error> FrameClient::openWriter(const WriterConfig &config) {
    auto [s, exc] = writer_client->stream(WRITE_ENDPOINT);
    return {Writer(s, config), exc};
}