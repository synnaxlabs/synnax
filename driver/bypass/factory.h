// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "driver/bypass/streamer.h"
#include "driver/bypass/writer.h"
#include "driver/pipeline/acquisition.h"
#include "driver/pipeline/control.h"
#include "driver/task/task.h"

namespace driver::bypass {
inline std::shared_ptr<pipeline::WriterFactory>
make_writer_factory(const std::shared_ptr<task::Context> &ctx) {
    auto factory = std::make_shared<pipeline::SynnaxWriterFactory>(ctx->client);
    if (ctx->bus() == nullptr || ctx->authority_mirror() == nullptr) {
        VLOG(1) << "[bus] no bus available, using direct server writer";
        return factory;
    }
    VLOG(1) << "[bus] wrapping writer factory with bus publish, group="
            << ctx->rack_key();
    return std::make_shared<WriterFactory>(
        factory,
        *ctx->bus(),
        ctx->rack_key(),
        *ctx->authority_mirror()
    );
}

inline std::shared_ptr<pipeline::StreamerFactory> make_streamer_factory(
    const std::shared_ptr<task::Context> &ctx,
    const x::control::Subject &subject
) {
    auto factory = std::make_shared<pipeline::SynnaxStreamerFactory>(ctx->client);
    if (ctx->bus() == nullptr) {
        VLOG(1) << "[bus] no bus available, using direct server streamer";
        return factory;
    }
    auto sub = subject;
    if (ctx->rack_key() != 0 && sub.group == 0) sub.group = ctx->rack_key();
    VLOG(1) << "[bus] wrapping streamer factory with bus subscription, subject="
            << sub.name << ", group=" << sub.group;
    return std::make_shared<StreamerFactory>(factory, *ctx->bus(), sub);
}
}
