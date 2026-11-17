// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>

#include "absl/log/log.h"

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

void Authorities::apply(synnax::framer::WriterConfig &config) const {
    if (this->keys.empty()) {
        config.authorities = this->authorities;
        return;
    }
    if (config.authorities.size() != config.channels.size()) {
        const auto all = config.authorities.empty() ? x::control::AUTHORITY_ABSOLUTE
                                                    : config.authorities[0];
        config.authorities.assign(config.channels.size(), all);
    }
    for (size_t i = 0; i < this->keys.size(); i++) {
        const auto it = std::ranges::find(config.channels, this->keys[i]);
        if (it == config.channels.end()) continue;
        const auto at = it - config.channels.begin();
        const auto j = this->authorities.size() == 1 ? 0 : i;
        config.authorities[at] = this->authorities[j];
    }
}

void Acquisition::run() {
    Run run{.config = this->writer_config};
    run.config.err_on_unauthorized = this->err_on_unauthorized;
    this->factory->begin_run();
    while (this->run_once(run)) {}
}

void Acquisition::stopped_with_err(const x::errors::Error &err) {
    this->source->stopped_with_err(err);
}

bool Acquisition::run_once(Run &run) {
    auto &fr = run.fr;
    auto &authorities = run.authorities;
    auto [writer, writer_err] = this->factory->open_writer(run.config);
    if (writer_err) {
        if (this->wait_for_core(writer_err)) return true;
        LOG(ERROR) << "[acquisition] failed to open writer: " << writer_err.message();
        this->source->stopped_with_err(writer_err);
        this->breaker.stop();
        return false;
    }
    this->breaker.reset_unlimited();
    this->source->writer_opened();
    x::errors::Error source_err;
    // A running breaker means the pipeline user has not called stop.
    while (this->breaker.running()) {
        if (fr.empty() && authorities.empty()) {
            auto source_err_i = this->source->read(this->breaker, fr, authorities);
            if (source_err_i) {
                fr.clear();
                authorities = Authorities{};
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
            if (auto err = authorities.validate()) {
                LOG(ERROR) << "[acquisition] invalid authorities: " << err.message();
                authorities = Authorities{};
                break;
            }
        }
        // Apply authority changes before writing the frame so the frame
        // is sent at the correct authority level.
        if (!authorities.empty()) {
            if (auto err = writer->set_authority(authorities)) {
                LOG(ERROR) << "[acquisition] failed to set authority: "
                           << err.message();
                break;
            }
            authorities.apply(run.config);
            authorities = Authorities{};
        }
        if (!fr.empty()) {
            if (auto err = run.rewrite ? writer->rewrite(fr) : writer->write(fr)) {
                LOG(ERROR) << "[acquisition] failed to write frame" << err;
                run.rewrite = true;
                break;
            }
            fr.clear();
            run.rewrite = false;
        }
        this->breaker.reset();
    }
    writer_err = writer->close();
    if (this->wait_for_core(writer_err)) return true;
    if (source_err && !source_err.matches(errors::NOMINAL_SHUTDOWN_ERROR))
        this->source->stopped_with_err(source_err);
    else if (writer_err)
        this->source->stopped_with_err(writer_err);
    this->breaker.stop();
    VLOG(1) << "[acquisition] acquisition thread stopped";
    return false;
}
}
