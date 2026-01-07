// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <exception>
#include <stdexcept>
#include <thread>

#include "glog/logging.h"
#include "nlohmann/json.hpp"

#include "driver/errors/errors.h"
#include "driver/pipeline/acquisition.h"

using json = nlohmann::json;

namespace pipeline {
SynnaxWriter::SynnaxWriter(synnax::Writer internal): internal(std::move(internal)) {}

xerrors::Error SynnaxWriter::write(const telem::Frame &fr) {
    return this->internal.write(fr);
}

xerrors::Error SynnaxWriter::close() {
    return this->internal.close();
}

SynnaxWriterFactory::SynnaxWriterFactory(std::shared_ptr<synnax::Synnax> client):
    client(std::move(client)) {}

std::pair<std::unique_ptr<pipeline::Writer>, xerrors::Error>
SynnaxWriterFactory::open_writer(const synnax::WriterConfig &config) {
    auto [sw, err] = client->telem.open_writer(config);
    if (err) return {nullptr, err};
    return {std::make_unique<SynnaxWriter>(std::move(sw)), xerrors::NIL};
}

Acquisition::Acquisition(
    std::shared_ptr<synnax::Synnax> client,
    synnax::WriterConfig writer_config,
    std::shared_ptr<Source> source,
    const breaker::Config &breaker_config,
    std::string thread_name
):
    Acquisition(
        std::make_shared<SynnaxWriterFactory>(std::move(client)),
        std::move(writer_config),
        std::move(source),
        breaker_config,
        std::move(thread_name)
    ) {}

Acquisition::Acquisition(
    std::shared_ptr<WriterFactory> factory,
    synnax::WriterConfig writer_config,
    std::shared_ptr<Source> source,
    const breaker::Config &breaker_config,
    std::string thread_name
):
    Base(breaker_config, std::move(thread_name)),
    factory(std::move(factory)),
    source(std::move(source)),
    writer_config(std::move(writer_config)) {}

/// @brief attempts to resolve the start timestamp for the writer from a series in
/// the frame with a timestamp data type. If that can't be found, resolveStart falls
/// back to now().
telem::TimeStamp resolve_start(const telem::Frame &frame) {
    auto min_timestamp = telem::TimeStamp::MAX();
    for (size_t i = 0; i < frame.size(); i++) {
        const auto &series = frame.series->at(i);
        if (series.data_type() == telem::TIMESTAMP_T && series.size() > 0) {
            const auto ts = series.at<telem::TimeStamp>(0);
            if (ts < min_timestamp) min_timestamp = ts;
        }
    }
    if (min_timestamp < telem::TimeStamp::MAX()) return min_timestamp;
    return telem::TimeStamp::now();
}

void Acquisition::run() {
    std::unique_ptr<Writer> writer;
    bool writer_opened = false;
    xerrors::Error writer_err;
    xerrors::Error source_err;
    // A running breaker means the pipeline user has not called stop.
    telem::Frame frame(0);
    while (this->breaker.running()) {

        if (auto source_err_i = this->source->read(this->breaker, frame)) {
            source_err = source_err_i;
            LOG(ERROR) << "[acquisition] failed to read source: "
                       << source_err.message();
            // With a temporary error, we just continue the loop. With any other
            // error we break and shut things down.
            if (source_err.matches(driver::TEMPORARY_HARDWARE_ERROR) &&
                this->breaker.wait(source_err.message()))
                continue;
            break;
        }
        if (source_err) source_err = xerrors::NIL;
        if (frame.empty()) continue;
        // Open the writer after receiving the first frame so we can resolve the
        // start timestamp from the data. This helps to account for clock drift
        // between the source we're recording data from and the system clock.
        if (!writer_opened) {
            this->writer_config.start = resolve_start(frame);
            // There are no scenarios where an acquisition task would want control
            // handoff between different levels of authorization, so we just reject
            // unauthorized writes.
            this->writer_config.err_on_unauthorized = true;
            auto [writer_i, writer_err_i] = factory->open_writer(writer_config);
            writer_err = writer_err_i;
            if (writer_err) {
                LOG(ERROR) << "[acquisition] failed to open writer: "
                           << writer_err.message();
                break;
            }
            writer = std::move(writer_i);
            writer_opened = true;
        }
        if (auto err = writer->write(frame)) {
            LOG(ERROR) << "[acquisition] failed to write frame" << err;
            break;
        }
        this->breaker.reset();
    }
    if (writer_opened) writer_err = writer->close();
    if (writer_err.matches(freighter::UNREACHABLE) &&
        this->breaker.wait(writer_err.message()))
        return this->run();
    if (source_err)
        this->source->stopped_with_err(source_err);
    else if (writer_err)
        this->source->stopped_with_err(writer_err);
    VLOG(1) << "[acquisition] acquisition thread stopped";
}
}
