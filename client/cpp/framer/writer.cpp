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
    auto [net_writer, err] = this->writer_client->stream(WRITE_ENDPOINT);
    if (err) return {Writer(), err};

    api::v1::FrameWriterRequest req;
    req.set_command(OPEN);
    cfg.to_proto(req.mutable_config());
    if (!net_writer->send(req).ok()) net_writer->close_send();
    auto [_, res_exc] = net_writer->receive();
    auto writer = Writer(std::move(net_writer), cfg);
    if (cfg.enable_experimental_codec) {
        writer.codec = Codec(this->channel_client);
        if (const auto codec_err = writer.codec.update(cfg.channels))
            return {Writer(), codec_err};
    }
    return {std::move(writer), res_exc};
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
    if (this->closed) return this->close_err;
    if (const auto err = this->init_request(fr)) {
        auto _ = this->close();
        this->close_err = err;
        return this->close_err;
    }
    return this->exec(*this->cached_write_req, false).second;
}

xerrors::Error Writer::init_request(const Frame &fr) {
    if (this->cfg.enable_experimental_codec) {
        if (this->cached_write_req == nullptr)
            this->cached_write_req = std::make_unique<api::v1::FrameWriterRequest>();
        this->cached_write_req->set_command(WRITE);
        if (const auto err = this->codec.encode(fr, this->codec_data)) return err;
        this->cached_write_req->set_buffer(
            this->codec_data.data(),
            this->codec_data.size()
        );
        return xerrors::NIL;
    }

    if (this->cached_write_req != nullptr && this->cfg.enable_proto_frame_caching) {
        for (size_t i = 0; i < fr.series->size(); i++)
            fr.series->at(i).to_proto(cached_frame->mutable_series(static_cast<int>(i))
            );
        return xerrors::NIL;
    }
    this->cached_write_req = nullptr;
    this->cached_write_req = std::make_unique<api::v1::FrameWriterRequest>();
    this->cached_write_req->set_command(WRITE);
    this->cached_frame = cached_write_req->mutable_frame();
    fr.to_proto(cached_frame);
    return xerrors::NIL;
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
            this->close_err = rec_exc.skip(freighter::EOF_ERR);
            return this->close_err;
        }
}
}
