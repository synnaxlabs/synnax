// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "driver/pipeline/acquisition.h"
#include "driver/pipeline/control.h"
#include "driver/task/task.h"
#include "streamer.h"
#include "writer.h"

namespace driver::bypass::pipeline {
inline std::shared_ptr<::driver::pipeline::WriterFactory>
create_writer_factory(const std::shared_ptr<task::Context> &ctx) {
    auto factory = std::make_shared<::driver::pipeline::SynnaxWriterFactory>(
        ctx->client
    );
    if (ctx->bus() == nullptr) {
        VLOG(1) << "[bypass] no bus available, using direct Core writer";
        return factory;
    }
    VLOG(1) << "[bypass] wrapping writer factory with bus publish, group="
            << ctx->rack_key();
    return std::make_shared<WriterFactory>(
        factory,
        ctx->bus(),
        ctx->control_states(),
        ctx->rack_key()
    );
}

inline std::shared_ptr<::driver::pipeline::StreamerFactory> create_streamer_factory(
    const std::shared_ptr<task::Context> &ctx,
    const x::control::Subject &subject
) {
    auto factory = std::make_shared<::driver::pipeline::SynnaxStreamerFactory>(
        ctx->client
    );
    if (ctx->bus() == nullptr) {
        VLOG(1) << "[bypass] no bus available, using direct Core streamer";
        return factory;
    }
    auto sub = subject;
    if (ctx->rack_key() != 0 && sub.group == 0) sub.group = ctx->rack_key();
    VLOG(1) << "[bypass] wrapping streamer factory with bus subscription, subject="
            << sub.name << ", group=" << sub.group;
    return std::make_shared<StreamerFactory>(factory, ctx->bus(), sub);
}
}
