// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <exception>
#include <string>
#include <utility>
#include <vector>

#include "client/clib/framer/framer.h"
#include "client/clib/internal.h"
#include "x/cpp/uuid/uuid.h"

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
    const uint8_t *authorities,
    const size_t authority_count,
    const char *subject_name,
    const uint32_t subject_group,
    const int32_t mode,
    const int32_t err_on_unauthorized,
    const int32_t enable_auto_commit,
    const int64_t auto_index_persist_interval,
    const int32_t auto_index,
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
        if (authorities != nullptr && authority_count > 0)
            wc.authorities.assign(authorities, authorities + authority_count);
        wc.start = x::telem::TimeStamp(start);
        wc.subject = x::control::Subject{
            .key = x::uuid::create().to_string(),
            .name = str_or(subject_name, ""),
            .group = subject_group,
        };
        wc.mode = mode_from_int(mode);
        wc.err_on_unauthorized = err_on_unauthorized != 0;
        wc.enable_auto_commit = enable_auto_commit != 0;
        wc.auto_index_persist_interval = auto_index_persist_interval > 0
                                           ? x::telem::TimeSpan(
                                                 auto_index_persist_interval
                                             )
                                           : x::telem::TimeSpan(x::telem::SECOND);
        wc.auto_index = auto_index != 0;

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
    const uint32_t index_channel,
    const int64_t *timestamps,
    const uint32_t *channels,
    const size_t channel_count,
    const void *data,
    const size_t sample_count,
    const char *data_type,
    SynnaxError *err
) {
    clear_err(err);
    if (writer == nullptr || channels == nullptr || data == nullptr) {
        set_err(err, CODE_INTERNAL, "sy.validation", "null writer, channels, or data");
        return CODE_INTERNAL;
    }
    try {
        const x::telem::DataType dt{str_or(data_type, "")};
        if (dt.density() == 0) {
            set_err(err, CODE_INTERNAL, "sy.validation", "unknown data type");
            return CODE_INTERNAL;
        }
        const bool has_index = index_channel != 0 && timestamps != nullptr;
        x::telem::Frame frame(channel_count + (has_index ? 1 : 0));
        if (has_index)
            frame.emplace(
                index_channel,
                x::telem::Series(timestamps, sample_count, x::telem::TIMESTAMP_T)
            );
        const auto *bytes = static_cast<const uint8_t *>(data);
        for (size_t i = 0; i < channel_count; i++)
            frame.emplace(
                channels[i],
                x::telem::Series(
                    bytes + i * sample_count * dt.density(),
                    sample_count,
                    dt
                )
            );
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

int32_t synnax_writer_write_strings(
    SynnaxWriter *writer,
    const uint32_t index_channel,
    const int64_t *timestamps,
    const uint32_t *channels,
    const size_t channel_count,
    const void *data,
    const size_t data_size,
    const size_t sample_count,
    SynnaxError *err
) {
    clear_err(err);
    if (writer == nullptr || channels == nullptr || data == nullptr) {
        set_err(err, CODE_INTERNAL, "sy.validation", "null writer, channels, or data");
        return CODE_INTERNAL;
    }
    try {
        const bool has_index = index_channel != 0 && timestamps != nullptr;
        x::telem::Frame frame(channel_count + (has_index ? 1 : 0));
        if (has_index)
            frame.emplace(
                index_channel,
                x::telem::Series(timestamps, sample_count, x::telem::TIMESTAMP_T)
            );
        // data is channel_count*sample_count records, channel-major (channel 0's
        // samples, then channel 1's). Each record is a uint32-LE byte length then that
        // many raw bytes; every read is bounds-checked against data_size to prevent
        // over-reads.
        const auto *bytes = static_cast<const uint8_t *>(data);
        size_t offset = 0;
        std::vector<std::string> all;
        all.reserve(channel_count * sample_count);
        for (size_t i = 0; i < channel_count * sample_count; i++) {
            if (offset + 4 > data_size) {
                set_err(
                    err,
                    CODE_INTERNAL,
                    "sy.validation",
                    "string buffer too small for length prefix"
                );
                return CODE_INTERNAL;
            }
            const uint32_t len = static_cast<uint32_t>(bytes[offset]) |
                                 static_cast<uint32_t>(bytes[offset + 1]) << 8 |
                                 static_cast<uint32_t>(bytes[offset + 2]) << 16 |
                                 static_cast<uint32_t>(bytes[offset + 3]) << 24;
            offset += 4;
            if (offset + len > data_size) {
                set_err(
                    err,
                    CODE_INTERNAL,
                    "sy.validation",
                    "string buffer too small for payload"
                );
                return CODE_INTERNAL;
            }
            all.emplace_back(reinterpret_cast<const char *>(bytes + offset), len);
            offset += len;
        }
        if (offset != data_size) {
            set_err(
                err,
                CODE_INTERNAL,
                "sy.validation",
                "string buffer has trailing bytes after the declared samples"
            );
            return CODE_INTERNAL;
        }
        for (size_t i = 0; i < channel_count; i++)
            frame.emplace(
                channels[i],
                x::telem::Series(
                    std::vector<std::string>(
                        all.begin() + i * sample_count,
                        all.begin() + (i + 1) * sample_count
                    ),
                    x::telem::STRING_T
                )
            );
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
