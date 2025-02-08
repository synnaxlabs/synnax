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

using namespace synnax;

/// @brief enumeration of possible writer commands.
enum WriterCommand : uint32_t {
    OPEN = 0,
    WRITE = 1,
    COMMIT = 2,
    ERROR_MODE = 3,
    SET_AUTHORITY = 4,
    SET_MODE = 5,
};


std::pair<Writer, freighter::Error> FrameClient::open_writer(
    const WriterConfig &config) const {
    auto [s, err] = writer_client->stream(WRITE_ENDPOINT);
    if (err) return {Writer(), err};
    api::v1::FrameWriterRequest req;
    req.set_command(OPEN);
    config.to_proto(req.mutable_config());
    err = s->send(req);
    if (err) return {Writer(), err};
    auto [_, recExc] = s->receive();
    return {Writer(std::move(s)), recExc};
}

Writer::Writer(std::unique_ptr<WriterStream> s) : stream(std::move(s)) {
}


void WriterConfig::to_proto(api::v1::FrameWriterConfig *f) const {
    subject.to_proto(f->mutable_control_subject());
    f->set_start(start.value);
    for (auto &auth: authorities) f->add_authorities(auth);
    for (auto &ch: channels) f->add_keys(ch);
    f->set_mode(mode);
    f->set_enable_auto_commit(enable_auto_commit);
    f->set_auto_index_persist_interval(auto_index_persist_interval.value);
    f->set_err_on_unauthorized(err_on_unauthorized);
}

bool Writer::write(const Frame &fr) {
    assert_open();
    if (err_accumulated) return false;
    api::v1::FrameWriterRequest req;
    req.set_command(WRITE);
    fr.to_proto(req.mutable_frame());
    if (const auto err = stream->send(req); err) err_accumulated = true;
    return !err_accumulated;
}

std::pair<synnax::TimeStamp, bool> Writer::commit() {
    assert_open();
    if (err_accumulated) return {synnax::TimeStamp(), false};

    api::v1::FrameWriterRequest req;
    req.set_command(COMMIT);

    if (const auto err = stream->send(req); err) {
        err_accumulated = true;
        return {synnax::TimeStamp(0), false};
    }

    while (true) {
        auto [res, recExc] = stream->receive();
        if (recExc) {
            err_accumulated = true;
            return {synnax::TimeStamp(0), false};
        }
        if (res.command() == COMMIT) return {synnax::TimeStamp(res.end()), res.ack()};
    }
}

bool Writer::set_authority(const synnax::Authority &auth) const {
    return this->set_authority({}, std::vector{auth});
}

bool Writer::set_authority(
    const ChannelKey &key,
    const synnax::Authority &authority
) const {
    return this->set_authority(std::vector{key}, std::vector{authority});
}

bool Writer::set_authority(
    const std::vector<ChannelKey> &keys,
    const std::vector<synnax::Authority> &authorities
) const {
    const WriterConfig config{.channels = keys, .authorities = authorities,};
    api::v1::FrameWriterRequest req;
    req.set_command(SET_AUTHORITY);
    config.to_proto(req.mutable_config());
    if (const auto err = stream->send(req); err) return false;
    while (true) {
        auto [res, recExc] = stream->receive();
        if (recExc) return false;
        if (res.command() == SET_AUTHORITY) return res.ack();
    }
}


freighter::Error Writer::error() const {
    assert_open();
    api::v1::FrameWriterRequest req;
    req.set_command(ERROR_MODE);
    if (const auto err = stream->send(req); err) return err;
    while (true) {
        auto [res, recExc] = stream->receive();
        if (recExc) return recExc;
        if (res.command() == ERROR_MODE) return freighter::Error(res.error());
    }
}

freighter::Error Writer::close() const {
    stream->close_send();
    while (true)
        if (const auto rec_exc = stream->receive().second; rec_exc) {
            if (rec_exc.matches(freighter::EOF_)) return freighter::NIL;
            return rec_exc;
        }
}


void Writer::assert_open() const {
    if (closed)
        throw std::runtime_error("cannot call method on closed writer");
}
