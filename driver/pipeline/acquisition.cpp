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
    std::shared_ptr<Source> source,
    const breaker::Config &breaker_config
) {    
    assert(ctx != nullptr);
    assert(source != nullptr);
    
    this->ctx = std::move(ctx);
    this->writer_config = writer_config;
    this->breaker = breaker::Breaker(breaker_config);
    this->source = std::move(source);
}


void Acquisition::start() {
    LOG(INFO) << "[Acquisition] Starting acquisition";
    thread = std::thread(&Acquisition::run, this);
    this->running = true;
}

void Acquisition::stop() {
    if (!running) return;
    this->running = false;
    thread.join();
    LOG(INFO) << "[Acquisition] Acquisition stopped";
}

void Acquisition::run() {
    this->writer_config.start = synnax::TimeStamp::now();
    auto [writer, wo_err] = ctx->client->telem.openWriter(writer_config);

    if (wo_err) {
        LOG(ERROR) << "[Acquisition] Failed to open writer: " << wo_err.message();
        if (wo_err.matches(freighter::UNREACHABLE) && breaker.wait(wo_err.message()))
            run();
        return;
    }
    while (this->running) {
        auto [frame, source_err] = source->read();
        // LOG(INFO) << "[Acquisition] Source read";
        if (source_err) {
            LOG(ERROR) << "[Acquisition] Failed to read source";
            if (
                source_err.matches(driver::TYPE_TEMPORARY_HARDWARE_ERROR) &&
                breaker.wait(source_err.message())
            )
                continue;
            break;
        }

        if (!writer.write(frame)) {
            LOG(ERROR) << "[Acquisition] Failed to write frame";
            break;
        }
        breaker.reset();
    }
    const auto err = writer.close();
    LOG(INFO) << "[Acquisition] Writer closed";
    if (err.matches(freighter::UNREACHABLE) && breaker.wait(err.message())) run();
    LOG(INFO) << "[Acquisition] Acquisition thread terminated";
}