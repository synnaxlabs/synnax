// Copyright 2025 Synnax Labs, Inc.
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
#include "synnax/pkg/api/grpc/v1/synnax/pkg/api/grpc/v1/framer.pb.h"

const std::string WRITE_ENDPOINT = "/frame/write";

/// @brief enumeration of possible writer commands.
enum WriterCommand : uint32_t {
    OPEN = 0,
    WRITE = 1,
    COMMIT = 2,
    SET_AUTHORITY = 3,
};

namespace synnax {
std::pair<Writer, xerrors::Error>
FrameClient::open_writer(const WriterConfig &cfg) const {
    auto [w, err] = this->writer_client->stream(WRITE_ENDPOINT);
    if (err) return {Writer(), err};

    api::v1::FrameWriterRequest req;
    req.set_command(OPEN);
    cfg.to_proto(req.mutable_config());
    if (!w->send(req).ok()) w->close_send();
    auto [_, res_exc] = w->receive();
    auto w2 = Writer(std::move(w), cfg);
    if (cfg.enable_experimental_codec) {
        auto [channels, err] = this->retrieve_channels(cfg.channels);
        if (err) return {Writer(), err};
        w2.codec = Codec(channels);
    }
    return {std::move(w2), res_exc};
}

Writer::Writer(std::unique_ptr<WriterStream> s, WriterConfig cfg):
    cfg(std::move(cfg)), stream(std::move(s)) {}

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

xerrors::Error Writer::write(const synnax::Frame &fr) {
    this->init_request(fr);
    return this->exec(*this->cached_write_req, false).second;
}

void Writer::init_request(const Frame &fr) {
    if (this->cfg.enable_experimental_codec) {
        if (this->cached_write_req == nullptr) this->cached_write_req = std::make_unique<api::v1::FrameWriterRequest>();
        this->cached_write_req->set_command(WRITE);
        // this->cached_frame = this->cached_write_req->mutable_frame();
        // fr.to_proto(this->cached_frame);
        std::vector<uint8_t> data;
        this->codec.encode(fr, 0, data);
        this->cached_write_req->set_buffer(data.data(), data.size());
        return;
    }

    if (this->cached_write_req != nullptr && this->cfg.enable_proto_frame_caching) {
        for (size_t i = 0; i < fr.series->size(); i++)
            fr.series->at(i).to_proto(cached_frame->mutable_series(i));
        return;
    }
    this->cached_write_req = nullptr;
    this->cached_write_req = std::make_unique<api::v1::FrameWriterRequest>();
    this->cached_write_req->set_command(WRITE);
    this->cached_frame = cached_write_req->mutable_frame();
    fr.to_proto(cached_frame);
}

std::pair<telem::TimeStamp, xerrors::Error> Writer::commit() {
    api::v1::FrameWriterRequest req;
    req.set_command(COMMIT);
    const auto [res, err] = this->exec(req, true);
    return {telem::TimeStamp(res.end()), err};
}

xerrors::Error Writer::set_authority(const telem::Authority &auth) {
    return this->set_authority({}, std::vector{auth});
}

xerrors::Error
Writer::set_authority(const ChannelKey &key, const telem::Authority &authority) {
    return this->set_authority(std::vector{key}, std::vector{authority});
}

xerrors::Error Writer::set_authority(
    const std::vector<ChannelKey> &keys,
    const std::vector<telem::Authority> &authorities
) {
    const WriterConfig config{.channels = keys, .authorities = authorities};
    api::v1::FrameWriterRequest req;
    req.set_command(SET_AUTHORITY);
    config.to_proto(req.mutable_config());
    return this->exec(req, true).second;
}

std::pair<api::v1::FrameWriterResponse, xerrors::Error>
Writer::exec(api::v1::FrameWriterRequest &req, const bool ack) {
    if (this->closed) return {api::v1::FrameWriterResponse(), this->close_err};
    if (const auto err = this->stream->send(req); err)
        return {api::v1::FrameWriterResponse(), this->close()};
    while (ack) {
        auto [res, res_err] = stream->receive();
        if (res_err) return {res, this->close()};
        if (res.command() == req.command()) return {res, xerrors::NIL};
    }
    return {api::v1::FrameWriterResponse(), xerrors::NIL};
}

xerrors::Error Writer::close() {
    if (this->closed) return this->close_err;
    this->closed = true;
    stream->close_send();
    while (true)
        if (const auto rec_exc = stream->receive().second; rec_exc) {
            this->close_err = rec_exc.skip(freighter::EOF_);
            return this->close_err;
        }
}
}
