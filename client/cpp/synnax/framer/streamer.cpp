// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <string>

#include "synnax/framer/framer.h"

std::string STREAM_ENDPOINT = "/frame/stream";

using namespace synnax::Framer;

Frame Streamer::read() {
    auto [fr, exc] = stream->receive();
    if (!exc.ok()) {
        throw;
    }
    return Frame(fr.frame());
}

void Streamer::close() {
    auto exc = stream->closeSend();
    if (!exc.ok()) {
        throw;
    }
    auto [res, recExc] = stream->receive();
    if (!recExc.ok()) {
        throw;
    }
}