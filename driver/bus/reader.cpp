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
#include <memory>
#include <utility>

#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/frame.h"
#include "x/cpp/telem/telem.h"

#include "driver/bus/reader.h"
#include "driver/codec/framing/framing.h"
#include "driver/common/common.h"
#include "driver/common/read_task.h"
#include "driver/icd/reader.h"
#include "driver/transport/transport.h"

namespace driver::bus {
namespace {
common::Warning dropped(const x::errors::Error &err) {
    return {.message = "Dropped invalid data", .description = err.data};
}
}

Reader::Reader(std::unique_ptr<codec::framing::Framer> framer, icd::Reader messages):
    framer(std::move(framer)), messages(std::move(messages)) {}

std::size_t Reader::decode(x::telem::Frame &fr, common::Warning &warning) {
    std::size_t n = 0;
    while (n < BATCH) {
        const auto [frame, err] = this->framer->next();
        if (err) {
            if (warning.empty()) warning = dropped(err);
            continue;
        }
        if (!frame.has_value()) break;
        n++;
        this->stamp = std::max(this->received, this->stamp + x::telem::NANOSECOND);
        const auto read_err = this->messages.read(*frame, this->stamp, fr, BATCH)
                                  .second;
        if (read_err && warning.empty())
            warning = {
                .message = "Failed to decode message",
                .description = read_err.data,
            };
    }
    return n;
}

common::ReadResult Reader::read(transport::Transport &t, x::telem::Frame &fr) {
    common::ReadResult res;
    if (this->decode(fr, res.warning) > 0) return res;
    const auto [chunk, err] = t.read(READ_TIMEOUT);
    if (err) {
        this->framer->clear();
        res.error = err;
        return res;
    }
    if (chunk.data.empty()) {
        if (const auto idle_err = this->framer->idle(); idle_err && res.warning.empty())
            res.warning = dropped(idle_err);
    } else {
        this->framer->write(chunk.data);
        this->received = chunk.time;
    }
    this->decode(fr, res.warning);
    return res;
}
}
