// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <thread>
#include "driver/pipeline/acquisition.h"
#include "nlohmann/json.hpp"
#include "driver/errors/errors.h"

using json = nlohmann::json;

using namespace pipeline;

Acquisition::Acquisition(
    std::shared_ptr<task::Context> ctx,
    WriterConfig writer_config,
    std::unique_ptr<Source> source,
    const breaker::Config &breaker_config
): ctx(std::move(ctx)),
   writer_config(std::move(writer_config)),
   source(std::move(source)),
   breaker(breaker::Breaker(breaker_config)) {
}


void Acquisition::start() {
    thread = std::thread(&Acquisition::run, this);
    running = true;
}

void Acquisition::stop() {
    if (!running) return;
    running = false;
    thread.join();
}

void Acquisition::run() {
    auto [writer, wo_err] = ctx->client->telem.openWriter(writer_config);
    if (wo_err) {
        if (wo_err.matches(freighter::UNREACHABLE) && breaker.wait(wo_err.message()))
            run();
        return;
    }
    while (running) {
        auto [frame, source_err] = source->read();
        if (source_err) {
            if (
                source_err.matches(driver::TYPE_TEMPORARY_HARDWARE_ERROR) &&
                breaker.wait(source_err.message())
            )
                continue;
            break;
        }
        if (!writer.write(frame)) break;
        breaker.reset();
    }
    const auto err = writer.close();
    if (err.matches(freighter::UNREACHABLE) && breaker.wait(err.message())) run();
    running = false;
}
