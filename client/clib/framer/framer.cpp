// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <exception>
#include <utility>

#include "client/clib/framer/framer.h"
#include "client/clib/internal.h"

using namespace synnax::clib;

namespace {
synnax::framer::WriterMode mode_from_int(const int32_t m) {
    switch (m) {
        case SYNNAX_WRITER_MODE_PERSIST:
            return synnax::framer::PersistOnly;
        case SYNNAX_WRITER_MODE_STREAM:
            return synnax::framer::StreamOnly;
        default: // DEFAULT (0) and PERSIST_STREAM (1)
            return synnax::framer::PersistStream;
    }
}
}

struct SynnaxWriter {
    synnax::framer::Writer writer;

    explicit SynnaxWriter(synnax::framer::Writer &&w): writer(std::move(w)) {}
};

int32_t synnax_writer_open(
    SynnaxClient *client,
    const int64_t start,
    const uint32_t *channels,
    const size_t channel_count,
    const int32_t mode,
    const int32_t enable_auto_commit,
    SynnaxWriter **out_writer,
    SynnaxError *err
) {
    clear_err(err);
    if (client == nullptr || out_writer == nullptr) {
        set_err(err, CODE_INTERNAL, "sy.validation", "null client or out_writer");
        return CODE_INTERNAL;
    }
    try {
        synnax::framer::WriterConfig wc;
        if (channels != nullptr) wc.channels.assign(channels, channels + channel_count);
        wc.start = x::telem::TimeStamp(start);
        wc.mode = mode_from_int(mode);
        wc.enable_auto_commit = enable_auto_commit != 0;

        auto [writer, w_err] = client->client.telem.open_writer(wc);
        if (!w_err.ok()) {
            set_err(err, CODE_ERROR, w_err.type, w_err.data);
            return CODE_ERROR;
        }
        *out_writer = new SynnaxWriter(std::move(writer));
        return CODE_OK;
    } catch (const std::exception &e) {
        set_err(err, CODE_INTERNAL, "sy.internal", e.what());
        return CODE_INTERNAL;
    } catch (...) {
        set_err(err, CODE_INTERNAL, "sy.internal", "unknown exception");
        return CODE_INTERNAL;
    }
}

int32_t synnax_writer_write(
    SynnaxWriter *writer,
    const uint32_t channel,
    const double *data,
    const size_t sample_count,
    SynnaxError *err
) {
    clear_err(err);
    if (writer == nullptr || data == nullptr) {
        set_err(err, CODE_INTERNAL, "sy.validation", "null writer or data");
        return CODE_INTERNAL;
    }
    try {
        x::telem::Frame frame(1);
        frame.emplace(channel, x::telem::Series(data, sample_count));
        const auto w_err = writer->writer.write(frame);
        if (!w_err.ok()) {
            set_err(err, CODE_ERROR, w_err.type, w_err.data);
            return CODE_ERROR;
        }
        return CODE_OK;
    } catch (const std::exception &e) {
        set_err(err, CODE_INTERNAL, "sy.internal", e.what());
        return CODE_INTERNAL;
    } catch (...) {
        set_err(err, CODE_INTERNAL, "sy.internal", "unknown exception");
        return CODE_INTERNAL;
    }
}

int32_t
synnax_writer_commit(SynnaxWriter *writer, int64_t *out_end_ts, SynnaxError *err) {
    clear_err(err);
    if (writer == nullptr) {
        set_err(err, CODE_INTERNAL, "sy.validation", "null writer");
        return CODE_INTERNAL;
    }
    try {
        auto [ts, c_err] = writer->writer.commit();
        if (!c_err.ok()) {
            set_err(err, CODE_ERROR, c_err.type, c_err.data);
            return CODE_ERROR;
        }
        if (out_end_ts != nullptr) *out_end_ts = ts.nanoseconds();
        return CODE_OK;
    } catch (const std::exception &e) {
        set_err(err, CODE_INTERNAL, "sy.internal", e.what());
        return CODE_INTERNAL;
    } catch (...) {
        set_err(err, CODE_INTERNAL, "sy.internal", "unknown exception");
        return CODE_INTERNAL;
    }
}

int32_t synnax_writer_close(SynnaxWriter *writer, SynnaxError *err) {
    clear_err(err);
    if (writer == nullptr) return CODE_OK;
    try {
        const auto c_err = writer->writer.close();
        delete writer;
        if (!c_err.ok()) {
            set_err(err, CODE_ERROR, c_err.type, c_err.data);
            return CODE_ERROR;
        }
        return CODE_OK;
    } catch (const std::exception &e) {
        delete writer;
        set_err(err, CODE_INTERNAL, "sy.internal", e.what());
        return CODE_INTERNAL;
    } catch (...) {
        delete writer;
        set_err(err, CODE_INTERNAL, "sy.internal", "unknown exception");
        return CODE_INTERNAL;
    }
}
