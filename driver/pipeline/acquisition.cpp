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
    if (thread.joinable() && std::this_thread::get_id() != thread.get_id())
        thread.join();
    if (running) return;
    this->running = true;
    thread = std::thread(&Acquisition::run, this);
}

void Acquisition::stop() {
    if (!running) return;
    this->running = false;
    if (thread.joinable() && std::this_thread::get_id() != thread.get_id()) {
        thread.join();
    };

    LOG(INFO) << "[acquisition] Acquisition stopped";
}

synnax::TimeStamp resolve_start(const synnax::Frame &frame) {
    for (size_t i = 0; i < frame.size(); i++) {
        if (frame.series->at(i).data_type == synnax::TIMESTAMP) {
            std::int64_t ts = frame.series->at(i).at<int64_t>(-1);
            if (ts != 0) return synnax::TimeStamp(ts);
        }
    }
    return synnax::TimeStamp::now();
}

void Acquisition::run() {
    LOG(INFO) << "[acquisition] Acquisition thread started";
    auto s_err = source->start();
    if (s_err) {
        LOG(ERROR) << "[acquisition] Failed to start source: " << s_err.message();
        if (s_err.matches(driver::TYPE_TEMPORARY_HARDWARE_ERROR) && breaker.wait(
                s_err.message()))
            run();
        return;
    }

    synnax::Writer writer;
    bool writer_opened = false;
    freighter::Error wo_err;

    std::this_thread::sleep_for(std::chrono::milliseconds(100));
    while (this->running) {
        auto [frame, source_err] = source->read();

        if (source_err.matches(driver::TYPE_CRITICAL_HARDWARE_ERROR)) {
            LOG(ERROR) <<
                    "[acquisition] Failed to read source: CRITICAL_HARDWARE_ERROR. Closing pipe.";
            break;
        }

        if (
            source_err.matches(driver::TYPE_TEMPORARY_HARDWARE_ERROR) &&
            breaker.wait(source_err.message())
        ) {
            LOG(ERROR) <<
                    "[acquisition] Failed to read source: TEMPORARY_HARDWARE_ERROR";
            continue;
        }

        if (!writer_opened) {
            this->writer_config.start = resolve_start(frame);
            auto res = ctx->client->telem.openWriter(writer_config);
            wo_err = res.second;
            if (wo_err) {
                LOG(ERROR) << "[acquisition] Failed to open writer: " << wo_err.
                        message();
                if (wo_err.matches(freighter::UNREACHABLE) && breaker.wait(
                        wo_err.message()))
                    run();
                
                return;
            }
            writer = std::move(res.first);
            writer_opened = true;
        }

        if (!writer.write(std::move(frame))) {
            LOG(ERROR) << "[acquisition] Failed to write frame";
            break;
        }

        breaker.reset();
    }
    const auto err = writer.close();
    LOG(INFO) << "[acquisition] Writer closed";
    if (err.matches(freighter::UNREACHABLE) && breaker.wait(err.message())) run();
    LOG(INFO) << "[acquisition] Acquisition thread terminated";
    source->stop();
}
