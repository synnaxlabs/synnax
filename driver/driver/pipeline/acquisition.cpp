//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <thread>
#include "driver/driver/pipeline/acquisition.h"
#include "nlohmann/json.hpp"

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
    running = true;
    thread = std::thread(&Acquisition::run, this);
}

void Acquisition::stop() {
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
        if (source_err) break;
        if (!writer.write(frame)) break;
    }

    const auto err = writer.close();
    if (err.matches(freighter::UNREACHABLE) && breaker.wait(err.message())) run();
    running = false;
}
