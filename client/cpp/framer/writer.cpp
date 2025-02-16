// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <string>

#include "client/cpp/framer/framer.h"
#include "synnax/pkg/api/grpc/v1/synnax/pkg/api/grpc/v1/framer.pb.h"

const std::string WRITE_ENDPOINT = "/frame/write";

/// @brief enumeration of possible writer commands.
enum WriterCommand : uint32_t {
    OPEN = 0,
    WRITE = 1,
    COMMIT = 2,
    ERROR_MODE = 3,
    SET_AUTHORITY = 4,
    SET_MODE = 5,
};

namespace synnax {
std::pair<Writer, xerrors::Error>
FrameClient::open_writer(const WriterConfig &config) const {
    auto [w, err] = this->writer_client->stream(WRITE_ENDPOINT);
    if (err) return {Writer(), err};
    api::v1::FrameWriterRequest req;
    req.set_command(OPEN);
    config.to_proto(req.mutable_config());
    if (!w->send(req).ok()) w->close_send();
    auto [_, res_exc] = w->receive();
    return {Writer(std::move(w)), res_exc};
}

Writer::Writer(std::unique_ptr<WriterStream> s) : stream(std::move(s)) {
}

void WriterConfig::to_proto(api::v1::FrameWriterConfig *f) const {
    this->subject.to_proto(f->mutable_control_subject());
    f->set_start(this->start.value);
    f->mutable_authorities()->Add(this->authorities.begin(), this->authorities.end());
    f->mutable_keys()->Add(this->channels.begin(), this->channels.end());
    f->set_mode(this->mode);
    f->set_enable_auto_commit(this->enable_auto_commit);
    f->set_auto_index_persist_interval(this->auto_index_persist_interval.value);
    f->set_err_on_unauthorized(this->err_on_unauthorized);
}

bool Writer::write(const Frame &fr) {
    this->assert_open();
    if (this->err_accumulated) return false;
    api::v1::FrameWriterRequest req;
    req.set_command(WRITE);
    fr.to_proto(req.mutable_frame());
    if (const auto err = this->stream->send(req)) this->err_accumulated = true;
    return !this->err_accumulated;
}

std::pair<telem::TimeStamp, bool> Writer::commit() {
    this->assert_open();
    if (err_accumulated) return {telem::TimeStamp(), false};
    api::v1::FrameWriterRequest req;
    req.set_command(COMMIT);
    const auto res = this->ack(req);
    return {telem::TimeStamp(res.end()), res.ack() && !this->err_accumulated};
}

xerrors::Error Writer::error() {
    assert_open();
    api::v1::FrameWriterRequest req;
    req.set_command(ERROR_MODE);
    const auto res = this->ack(req);
    return xerrors::Error(res.error());
}

bool Writer::set_authority(const synnax::Authority &auth) {
    return this->set_authority({}, std::vector{auth});
}

bool Writer::set_authority(
    const ChannelKey &key,
    const synnax::Authority &authority
) {
    return this->set_authority(std::vector{key}, std::vector{authority});
}

bool Writer::set_authority(
    const std::vector<ChannelKey> &keys,
    const std::vector<synnax::Authority> &authorities
) {
    this->assert_open();
    const WriterConfig config{.channels = keys, .authorities = authorities,};
    api::v1::FrameWriterRequest req;
    req.set_command(SET_AUTHORITY);
    config.to_proto(req.mutable_config());
    this->ack(req);
    return !this->err_accumulated;
}

api::v1::FrameWriterResponse Writer::ack(api::v1::FrameWriterRequest &req) {
    if (const auto err = this->stream->send(req); err) {
        this->err_accumulated = true;
        return {};
    }
    while (true) {
        auto [res, res_err] = stream->receive();
        if (res_err) {
            this->err_accumulated = true;
            return res;
        }
        if (res.command() == req.command()) return res;
    }
}

xerrors::Error Writer::close() const {
    stream->close_send();
    while (true)
        if (const auto rec_exc = stream->receive().second; rec_exc)
            return rec_exc.skip(freighter::EOF_);
}


void Writer::assert_open() const {
    if (closed)
        throw std::runtime_error("cannot call method on closed writer");
}
}
