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
): ctx(std::move(ctx)),
   thread(nullptr),
   writer_config(std::move(writer_config)),
   breaker(breaker_config),
   source(std::move(source)) {
}

void Acquisition::start() {
    if (this->breaker.running()) return;
    this->maybeJoinThread();
    this->breaker.start();
    const auto s_err = this->source->start();
    if (s_err) {
        LOG(ERROR) << "[acquisition] failed to start source: " << s_err.message();
        if (
            s_err.matches(driver::TYPE_TEMPORARY_HARDWARE_ERROR)
            && breaker.wait(s_err.message())
        )
            run();
        return;
    }
    this->thread = std::make_unique<std::thread>(&Acquisition::run, this);
    LOG(INFO) << "[acquisition] acquisition started";
}

void Acquisition::maybeJoinThread() const {
    if (
        this->thread == nullptr ||
        !this->thread->joinable() ||
        std::this_thread::get_id() == this->thread->get_id()
    )
        return;
    this->thread->join();
}


void Acquisition::stop() {
    const auto was_running = this->breaker.running();
    // Stop the breaker and join the thread regardless of whether the breaker is running.
    // This ensures the thread gets joined in case of an internal error.
    this->breaker.stop();
    this->maybeJoinThread();
    if (was_running) this->source->stop();
    LOG(INFO) << "[acquisition] acquisition stopped";
}

synnax::TimeStamp resolve_start(const synnax::Frame &frame) {
    for (size_t i = 0; i < frame.size(); i++)
        if (frame.series->at(i).data_type == synnax::TIMESTAMP) {
            const auto ts = frame.series->at(i).at<int64_t>(0);
            if (ts != 0) return synnax::TimeStamp(ts);
        }
    return synnax::TimeStamp::now();
}

void Acquisition::runInternal() {
    LOG(INFO) << "[acquisition] acquisition thread started";
    synnax::Writer writer;
    bool writer_opened = false;
    freighter::Error writer_err;

    while (this->breaker.running()) {
        auto [frame, source_err] = this->source->read();
        if (source_err) {
            LOG(ERROR) << "[acquisition] failed to read source" << source_err.message();
            // With a temporary error, we just continue the loop. With any other error
            // we break and shut things down.
            if (
                source_err.matches(driver::TYPE_TEMPORARY_HARDWARE_ERROR) &&
                this->breaker.wait(source_err.message())
            )
                continue;
            break;
        }
        // Open the writer after receiving the first frame so we can resolve the start
        // timestamp from the data.
        if (!writer_opened) {
            this->writer_config.start = resolve_start(frame);
            this->writer_config.err_on_unauthorized = true;
            auto res = ctx->client->telem.openWriter(writer_config);
            writer_err = res.second;
            if (writer_err) {
                LOG(ERROR) << "[acquisition] failed to open writer: " << writer_err.
                        message();
                break;
            }
            writer = std::move(res.first);
            writer_opened = true;
        }
        if (!writer.write(frame)) {
            LOG(ERROR) << "[acquisition] failed to write frame";
            break;
        }
        this->breaker.reset();
    }
    if (writer_opened) writer_err = writer.close();
    if (!writer_err) return;

    this->sendError(writer_err);

    // The only case where we want to retry is if the error is a connection loss and
    // we haven't expired the maximum retries on the breaker
    if (
        !writer_err.matches(freighter::UNREACHABLE) ||
        !this->breaker.wait(writer_err.message())
    )
        return;

    this->runInternal();
}

void Acquisition::sendError(const freighter::Error &err) const {
    if (err.matches(synnax::UNAUTHORIZED_ERROR)) {
        return this->source->sendError(freighter::Error{
            synnax::UNAUTHORIZED_ERROR,
            err.data,
        });
    }
    this->source->sendError(err);
}

/// @brief the the main run function for the acquisition thread. Servers as a wrapper
/// around runInternal to catch exceptions and log them.
void Acquisition::run() {
    try {
        // This will call itself recursively.
        this->runInternal();
    } catch (const std::exception &e) {
        LOG(ERROR) << "[acquisition] unhandled standard exception: " << e.what();
    } catch (...) {
        LOG(ERROR) << "[acquisition] unhandled unknown exception";
    }
    // Stop the acquisition thread. This is an idempotent operation, which means its
    // safe to call stop even in a scenario where a user called stop explcitily (as
    // opposed to an internal error).
    this->stop();
}
