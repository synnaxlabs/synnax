// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "absl/log/log.h"

#include "client/cpp/errors/errors.h"

#include "driver/errors/errors.h"
#include "driver/pipeline/acquisition.h"

namespace driver::pipeline {
SynnaxWriter::SynnaxWriter(synnax::framer::Writer internal):
    internal(std::move(internal)) {}

x::errors::Error SynnaxWriter::write(const x::telem::Frame &fr) {
    return this->internal.write(fr);
}

x::errors::Error SynnaxWriter::set_authority(const Authorities &authorities) {
    return this->internal
        .set_authority(authorities.keys, authorities.authorities, false);
}

x::errors::Error SynnaxWriter::close() {
    return this->internal.close();
}

SynnaxWriterFactory::SynnaxWriterFactory(std::shared_ptr<synnax::Synnax> client):
    client(std::move(client)) {}

std::pair<std::unique_ptr<pipeline::Writer>, x::errors::Error>
SynnaxWriterFactory::open_writer(const synnax::framer::WriterConfig &config) {
    auto [sw, err] = client->telem.open_writer(config);
    if (err) return {nullptr, err};
    return {std::make_unique<SynnaxWriter>(std::move(sw)), x::errors::NIL};
}

Acquisition::Acquisition(
    std::shared_ptr<synnax::Synnax> client,
    synnax::framer::WriterConfig writer_config,
    std::shared_ptr<Source> source,
    const x::breaker::Config &breaker_config,
    std::string thread_name,
    bool err_on_unauthorized
):
    Acquisition(
        std::make_shared<SynnaxWriterFactory>(std::move(client)),
        std::move(writer_config),
        std::move(source),
        breaker_config,
        std::move(thread_name),
        err_on_unauthorized
    ) {}

Acquisition::Acquisition(
    std::shared_ptr<WriterFactory> factory,
    synnax::framer::WriterConfig writer_config,
    std::shared_ptr<Source> source,
    const x::breaker::Config &breaker_config,
    std::string thread_name,
    bool err_on_unauthorized
):
    Base(breaker_config, std::move(thread_name)),
    factory(std::move(factory)),
    source(std::move(source)),
    writer_config(std::move(writer_config)),
    err_on_unauthorized(err_on_unauthorized) {}

void Acquisition::run() {
    this->writer_config.err_on_unauthorized = this->err_on_unauthorized;
    auto [writer, writer_err] = this->factory->open_writer(this->writer_config);
    if (writer_err) {
        if (synnax::errors::is_temporarily_unavailable(writer_err) &&
            this->breaker.wait(writer_err.message())) {
            this->run();
            return;
        }
        LOG(ERROR) << "[acquisition] failed to open writer: " << writer_err.message();
        this->source->stopped_with_err(writer_err);
        this->breaker.stop();
        return;
    }
    x::errors::Error source_err;
    x::telem::Frame fr(0);
    Authorities authorities;
    // A running breaker means the pipeline user has not called stop.
    while (this->breaker.running()) {
        fr.clear();
        authorities.keys.clear();
        authorities.authorities.clear();
        auto source_err_i = this->source->read(this->breaker, fr, authorities);
        if (source_err_i) {
            source_err = source_err_i;
            if (!source_err.matches(errors::NOMINAL_SHUTDOWN_ERROR))
                LOG(ERROR) << "[acquisition] failed to read source: "
                           << source_err.message();
            // With a temporary error, we just continue the loop. With any other
            // error we break and shut things down.
            if (source_err.matches(errors::TEMPORARY_HARDWARE_ERROR) &&
                this->breaker.wait(source_err.message()))
                continue;
            break;
        }
        if (source_err) source_err = x::errors::NIL;
        if (fr.empty() && authorities.empty()) {
            this->breaker.reset();
            continue;
        }
        // Apply authority changes before writing the frame so the frame
        // is sent at the correct authority level.
        if (!authorities.empty()) {
            if (auto err = authorities.validate()) {
                LOG(ERROR) << "[acquisition] invalid authorities: " << err.message();
                break;
            }
            if (auto err = writer->set_authority(authorities)) {
                LOG(ERROR) << "[acquisition] failed to set authority: "
                           << err.message();
                break;
            }
        }
        if (!fr.empty()) {
            if (auto err = writer->write(fr)) {
                LOG(ERROR) << "[acquisition] failed to write frame" << err;
                break;
            }
        }
        this->breaker.reset();
    }
    writer_err = writer->close();
    if (synnax::errors::is_temporarily_unavailable(writer_err) &&
        this->breaker.wait(writer_err.message())) {
        this->run();
        return;
    }
    if (source_err && !source_err.matches(errors::NOMINAL_SHUTDOWN_ERROR))
        this->source->stopped_with_err(source_err);
    else if (writer_err)
        this->source->stopped_with_err(writer_err);
    this->breaker.stop();
    VLOG(1) << "[acquisition] acquisition thread stopped";
}
}
