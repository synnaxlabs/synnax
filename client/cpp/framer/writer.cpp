// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <string>
#include <utility>

#include "client/cpp/framer/framer.h"

namespace synnax::framer {
/// @brief enumeration of possible writer commands.
enum WriterCommand : uint32_t {
    OPEN = 0,
    WRITE = 1,
    COMMIT = 2,
    SET_AUTHORITY = 3,
};

std::pair<Writer, x::errors::Error> Client::open_writer(const WriterConfig &cfg) const {
    Codec codec;
    if (cfg.enable_experimental_codec) {
        codec = Codec(this->channel_client);
        if (const auto codec_err = codec.update(cfg.channels))
            return {Writer(), codec_err};
    }
    auto [net_writer, err] = this->writer_client->stream("/frame/write");
    if (err) return {Writer(), err};
    api::v1::FrameWriterRequest req;
    req.set_command(OPEN);
    cfg.to_proto(req.mutable_config());
    if (!net_writer->send(req).ok()) net_writer->close_send();
    auto [_, res_exc] = net_writer->receive();
    auto writer = Writer(std::move(net_writer), cfg, codec);
    return {std::move(writer), res_exc};
}

Writer::Writer(std::unique_ptr<WriterStream> s, WriterConfig cfg, const Codec &codec):
    cfg(std::move(cfg)), codec(codec), stream(std::move(s)) {}

void WriterConfig::to_proto(api::v1::FrameWriterConfig *f) const {
    this->subject.to_proto(f->mutable_control_subject());
    f->set_start(this->start.nanoseconds());
    f->mutable_authorities()->Add(this->authorities.begin(), this->authorities.end());
    f->mutable_keys()->Add(this->channels.begin(), this->channels.end());
    f->set_mode(this->mode);
    f->set_enable_auto_commit(this->enable_auto_commit);
    f->set_auto_index_persist_interval(this->auto_index_persist_interval.nanoseconds());
    f->set_err_on_unauthorized(this->err_on_unauthorized);
}

x::errors::Error Writer::write(const x::telem::Frame &fr) {
    if (this->close_err) return this->close_err;
    if (const auto err = this->init_request(fr)) return this->close(err);
    return this->exec(*this->cached_write_req, false).second;
}

std::pair<x::telem::TimeStamp, x::errors::Error> Writer::commit() {
    if (this->close_err) return {x::telem::TimeStamp(0), this->close_err};
    api::v1::FrameWriterRequest req;
    req.set_command(COMMIT);
    const auto [res, err] = this->exec(req, true);
    return {x::telem::TimeStamp(res.end()), err};
}

x::errors::Error Writer::set_authority(const x::control::Authority &auth) {
    return this->set_authority({}, std::vector{auth});
}

x::errors::Error
Writer::set_authority(const channel::Key &key, const x::control::Authority &authority) {
    return this->set_authority(std::vector{key}, std::vector{authority});
}

x::errors::Error Writer::set_authority(
    const std::vector<channel::Key> &keys,
    const std::vector<x::control::Authority> &authorities,
    const bool ack
) {
    if (this->close_err) return this->close_err;
    const WriterConfig config{.channels = keys, .authorities = authorities};
    api::v1::FrameWriterRequest req;
    req.set_command(SET_AUTHORITY);
    config.to_proto(req.mutable_config());
    return this->exec(req, ack).second;
}

x::errors::Error Writer::close() {
    return this->close(x::errors::NIL);
}

x::errors::Error Writer::init_request(const x::telem::Frame &fr) {
    if (this->cfg.enable_experimental_codec) {
        if (this->cached_write_req == nullptr)
            this->cached_write_req = std::make_unique<api::v1::FrameWriterRequest>();
        this->cached_write_req->set_command(WRITE);
        if (const auto err = this->codec.encode(fr, this->codec_data)) return err;
        this->cached_write_req->set_buffer(
            this->codec_data.data(),
            this->codec_data.size()
        );
        return x::errors::NIL;
    }

    if (this->cached_write_req != nullptr && this->cfg.enable_proto_frame_caching) {
        for (size_t i = 0; i < fr.series->size(); i++)
            fr.series->at(i).to_proto(
                cached_frame->mutable_series(static_cast<int>(i))
            );
        return x::errors::NIL;
    }
    this->cached_write_req = nullptr;
    this->cached_write_req = std::make_unique<api::v1::FrameWriterRequest>();
    this->cached_write_req->set_command(WRITE);
    this->cached_frame = cached_write_req->mutable_frame();
    fr.to_proto(cached_frame);
    return x::errors::NIL;
}

x::errors::Error Writer::close(const x::errors::Error &close_err) {
    if (this->close_err) return this->close_err.skip(WRITER_CLOSED);
    this->close_err = close_err;
    stream->close_send();
    while (true) {
        if (this->close_err) return this->close_err.skip(WRITER_CLOSED);
        auto [res, err] = stream->receive();
        if (err)
            this->close_err = err.matches(freighter::EOF_ERR) ? WRITER_CLOSED : err;
        else
            this->close_err = x::errors::Error(res.error());
    }
}

std::pair<api::v1::FrameWriterResponse, x::errors::Error>
Writer::exec(api::v1::FrameWriterRequest &req, const bool ack) {
    if (const auto err = this->stream->send(req); err)
        return {api::v1::FrameWriterResponse(), this->close(err)};
    while (ack) {
        auto [res, res_err] = stream->receive();
        if (res_err) return {res, this->close(res_err)};
        if (auto err = x::errors::Error(res.error())) return {res, this->close(err)};
        if (res.command() == req.command()) return {res, x::errors::NIL};
    }
    return {api::v1::FrameWriterResponse(), x::errors::NIL};
}
}
