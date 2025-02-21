// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// std
#include <thread>
#include <exception>
#include <stdexcept>

/// external
#include "nlohmann/json.hpp"

/// module
#include "driver/errors/errors.h"

/// internal
#include "driver/pipeline/acquisition.h"

using json = nlohmann::json;

using namespace pipeline;

SynnaxWriter::SynnaxWriter(std::unique_ptr<synnax::Writer> internal)
    : internal(std::move(internal)) {
}

bool SynnaxWriter::write(synnax::Frame &fr) { return this->internal->write(fr); }

xerrors::Error SynnaxWriter::close() { return this->internal->close(); }

SynnaxWriterFactory::SynnaxWriterFactory(std::shared_ptr<synnax::Synnax> client)
    : client(std::move(client)) {
}

std::pair<std::unique_ptr<pipeline::Writer>, xerrors::Error>
SynnaxWriterFactory::open_writer(const WriterConfig &config) {
    auto [sw, err] = client->telem.open_writer(config);
    if (err) return {nullptr, err};
    return {
        std::make_unique<SynnaxWriter>(
            std::make_unique<synnax::Writer>(std::move(sw))),
        xerrors::NIL
    };
}

Acquisition::Acquisition(
    std::shared_ptr<synnax::Synnax> client,
    WriterConfig writer_config,
    std::shared_ptr<Source> source,
    const breaker::Config &breaker_config
) : thread(nullptr),
    factory(std::make_shared<SynnaxWriterFactory>(std::move(client))),
    writer_config(std::move(writer_config)),
    breaker(breaker_config),
    source(std::move(source)) {
}

Acquisition::Acquisition(
    std::shared_ptr<WriterFactory> factory,
    WriterConfig writer_config,
    std::shared_ptr<Source> source,
    const breaker::Config &breaker_config
) : thread(nullptr),
    factory(std::move(factory)),
    writer_config(std::move(writer_config)),
    breaker(breaker_config),
    source(std::move(source)) {
}

void Acquisition::ensure_thread_joined() const {
    if (
        this->thread == nullptr ||
        !this->thread->joinable() ||
        std::this_thread::get_id() == this->thread->get_id()
    )
        return;
    this->thread->join();
}

void Acquisition::start() {
    if (this->breaker.running()) return;
    VLOG(1) << "[acquisition] starting pipeline";
    this->ensure_thread_joined();
    this->breaker.start();
    this->thread = std::make_unique<std::thread>(&Acquisition::run, this);
}

void Acquisition::stop() {
    const auto was_running = this->breaker.running();
    if (was_running)
        VLOG(1) << "[acquisition] stopping pipeline";
    else
        VLOG(1) << "[acquisition] pipeline already stopped";
    this->breaker.stop();
    this->ensure_thread_joined();
    if (was_running)
        VLOG(1) << "[acquisition] pipeline stopped";
}

/// @brief the main run function for the acquisition thread. Servers as a wrapper
/// around runInternal to catch exceptions and log them.
void Acquisition::run() {
    try {
        // This will call itself recursively.
        this->run_internal();
    } catch (const std::exception &e) {
        LOG(ERROR) << "[acquisition] unhandled standard exception: " << e.what();
    } catch (...) {
        LOG(ERROR) << "[acquisition] unhandled unknown exception";
    }
    // Stop the acquisition thread. This is an idempotent operation, which means its
    // safe to call stop even in a scenario where a user called stop explicitly (as
    // opposed to an internal error).
    this->stop();
}


/// @brief attempts to resolve the start timestamp for the writer from a series in
/// the frame with a timestamp data type. If that can't be found, resolveStart falls
/// back to the
telem::TimeStamp resolve_start(const synnax::Frame &frame) {
    for (size_t i = 0; i < frame.size(); i++)
        if (frame.series->at(i).data_type == telem::TIMESTAMP_T) {
            const auto ts = frame.series->at(i).at<int64_t>(0);
            if (ts != 0) return telem::TimeStamp(ts);
        }
    return telem::TimeStamp::now();
}

void Acquisition::run_internal() {
    VLOG(1) << "[acquisition] acquisition thread started";
    std::unique_ptr<Writer> writer;
    bool writer_opened = false;
    xerrors::Error writer_err;
    // A running breaker means the pipeline user has not called stop.
    while (this->breaker.running()) {
        auto [frame, source_err] = this->source->read(this->breaker);
        this->middleware_chain.exec(frame);
        if (source_err) {
            LOG(ERROR) << "[acquisition] failed to read source: " << source_err.
                    message();
            // With a temporary error, we just continue the loop. With any other error
            // we break and shut things down.
            if (
                source_err.matches(driver::TEMPORARY_HARDWARE_ERROR) &&
                this->breaker.wait(source_err.message())
            )
                continue;
            break;
        }
        // Open the writer after receiving the first frame so we can resolve the start
        // timestamp from the data. This helps to account for clock drift between the
        // source we're recording data from and the system clock.
        if (!writer_opened) {
            this->writer_config.start = resolve_start(frame);
            // There are no scenarios where an acquisition task would want control
            // handoff between different levels of authorization, so we just reject
            // unauthorized writes.
            this->writer_config.err_on_unauthorized = true;
            auto res = factory->open_writer(writer_config);
            writer_err = res.second;
            if (writer_err) {
                LOG(ERROR) << "[acquisition] failed to open writer: " << writer_err.
                        message();
                break;
            }
            writer = std::move(res.first);
            writer_opened = true;
        }
        if (!writer->write(frame)) {
            LOG(ERROR) << "[acquisition] failed to write frame";
            break;
        }
        this->breaker.reset();
    }
    if (writer_opened) writer_err = writer->close();
    if (
        writer_err.matches(freighter::UNREACHABLE) &&
        this->breaker.wait(writer_err.message())
    )
        return this->run_internal();
    if (writer_err) this->source->stopped_with_err(writer_err);
    VLOG(1) << "[acquisition] acquisition thread stopped";
}

Acquisition::~Acquisition() {
    this->stop();
    this->ensure_thread_joined();
}
