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
#include <exception>
#include <stdexcept>

using json = nlohmann::json;

using namespace pipeline;

Acquisition::Acquisition(
    std::shared_ptr<task::Context> ctx,
    WriterConfig writer_config,
    std::shared_ptr<Source> source,
    const breaker::Config &breaker_config
): ctx(std::move(ctx)), thread(nullptr), writer_config(writer_config), breaker(breaker_config), source(std::move(source)) {
}

void Acquisition::start() {
    if(breaker.running()) return;
    if (this->thread != nullptr && thread->joinable() && std::this_thread::get_id() != thread->get_id())
        this->thread->join();
    breaker.start(); 
    auto s_err = source->start();
    if (s_err) {
        LOG(ERROR) << "[acquisition] Failed to start source: " << s_err.message();
        if (s_err.matches(driver::TYPE_TEMPORARY_HARDWARE_ERROR) && breaker.wait(
                s_err.message()))
            run();
        return;
    }  
    thread = std::make_unique<std::thread>(&Acquisition::run, this);
    LOG(INFO) << "[acquisition] Acquisition started";
}

void Acquisition::stop() {
    if (!breaker.running()) return;
    breaker.stop();
    if (this->thread != nullptr && thread->joinable() && std::this_thread::get_id() != thread->get_id()) {
       this->thread->join();
    };
    source->stop();
    LOG(INFO) << "[acquisition] Acquisition stopped";
}

synnax::TimeStamp resolve_start(const synnax::Frame &frame) {
    for (size_t i = 0; i < frame.size(); i++) {
        if (frame.series->at(i).data_type == synnax::TIMESTAMP) {
            std::int64_t ts = frame.series->at(i).at<int64_t>(0);
            if (ts != 0) return synnax::TimeStamp(ts);
        }
    }
    return synnax::TimeStamp::now();
}

void Acquisition::runInternal() {
    LOG(INFO) << "[acquisition] Acquisition thread started";
    synnax::Writer writer;
    bool writer_opened = false;
    freighter::Error wo_err;

    while (breaker.running()) {
        auto [frame, source_err] = source->read();
<<<<<<< HEAD
        if (source_err) {
            LOG(ERROR) << "[Acquisition] Failed to read source";
            if (
                source_err.matches(driver::TYPE_TEMPORARY_HARDWARE_ERROR) &&
                breaker.wait(source_err.message())
            ) continue;
=======

        if (source_err.matches(driver::TYPE_CRITICAL_HARDWARE_ERROR)) {
            LOG(ERROR) << "[acquisition] Failed to read source: CRITICAL_HARDWARE_ERROR. Closing pipe.";
>>>>>>> 3ce13a8260a56dfe9b121d431971116ebed49a86
            break;
        }

        if (source_err.matches(driver::TYPE_TEMPORARY_HARDWARE_ERROR) && breaker.wait(source_err.message())) {
            continue;
        }

        if (!writer_opened) {
            this->writer_config.start = resolve_start(frame);
            auto res = ctx->client->telem.openWriter(writer_config);
            wo_err = res.second;
            if (wo_err) {
                LOG(ERROR) << "[acquisition] Failed to open writer: " << wo_err.message();
                if (wo_err.matches(freighter::UNREACHABLE) && breaker.wait(wo_err.message()))
                    runInternal();
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
    if (err.matches(freighter::UNREACHABLE) && breaker.wait(err.message())) run();
    this->stop();
}

void Acquisition::run() {
    try{
        runInternal();
    } catch (const std::exception &e) {
        LOG(ERROR) << "[acquisition] Unhandled standard exception: " << e.what();
        this->stop();
    } catch (...) {
        LOG(ERROR) << "[acquisition] Unhandled unknown exception";
        this->stop();
    }
}
