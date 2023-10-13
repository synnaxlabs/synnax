// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "synnax/framer/framer.h"

using namespace Synnax::Framer;

const std::string ITERATOR_ENDPOINT = "/frame/iterate";
const std::string STREAM_ENDPOINT = "/frame/stream";
const std::string WRITE_ENDPOINT = "/frame/write";

Iterator Client::openIterator(const IteratorConfig &config) {
    auto s = iterator_client->stream(ITERATOR_ENDPOINT);
    return {s, config};
}

Streamer Client::openStreamer(const StreamerConfig &config) {
    auto s = streamer_client->stream(STREAM_ENDPOINT);
    return {s, config};
}

Writer Client::openWriter(const WriterConfig &config) {
    auto s = writer_client->stream(WRITE_ENDPOINT);
    return {s, config};
}