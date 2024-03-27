// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// std
#include <string>


/// api protos
#include "synnax/pkg/api/grpc/v1/synnax/pkg/api/grpc/v1/framer.pb.h"

/// internal
#include "client/cpp/synnax/framer/framer.h"

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


std::pair<Writer, freighter::Error> FrameClient::openWriter(const WriterConfig &config) {
    auto [s, exc] = writer_client->stream(WRITE_ENDPOINT);
    if (exc) return {Writer(), exc};
    auto req = api::v1::FrameWriterRequest();
    req.set_command(OPEN);
    config.toProto(req.mutable_config());
    exc = s->send(req);
    if (exc) return {Writer(), exc};
    auto [_, recExc] = s->receive();
    return {Writer(std::move(s)), recExc};
}

Writer::Writer(std::unique_ptr<WriterStream> s): stream(std::move(s)) {}


void WriterConfig::toProto(api::v1::FrameWriterConfig *f) const {
    subject.to_proto(f->mutable_control_subject());
    f->set_start(start.value);
    for (auto &auth: authorities) f->add_authorities(auth);
    for (auto &ch: channels) f->add_keys(ch);
    f->set_mode(mode);
}

bool Writer::write(Frame fr) {
    assertOpen();
    if (err_accumulated) return false;
    api::v1::FrameWriterRequest req;
    req.set_command(WRITE);
    fr.toProto(req.mutable_frame());
    auto exc = stream->send(req);
    if (exc) err_accumulated = true;
    return !err_accumulated;
}

bool Writer::setMode(synnax::WriterMode mode) {
    assertOpen();
    if (err_accumulated) return false;
    auto req = api::v1::FrameWriterRequest();
    req.set_command(SET_MODE);
    auto config = WriterConfig();
    config.mode = mode;
    config.toProto(req.mutable_config());
    auto exc = stream->send(req);
    if (exc) {
        err_accumulated = true;
        return false;
    }
    while (true) {
        auto [res, recExc] = stream->receive();
        if (recExc) {
            err_accumulated = true;
            return false;
        }
        if (res.command() == SET_MODE) return res.ack();
    }
}


std::pair<synnax::TimeStamp, bool> Writer::commit() {
    assertOpen();
    if (err_accumulated) return {synnax::TimeStamp(), false};

    auto req = api::v1::FrameWriterRequest();
    req.set_command(COMMIT);
    auto exc = stream->send(req);
    if (exc) {
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

freighter::Error Writer::error() {
    assertOpen();

    auto req = api::v1::FrameWriterRequest();
    req.set_command(ERROR_MODE);
    auto exc = stream->send(req);
    if (exc) return exc;

    while (true) {
        auto [res, recExc] = stream->receive();
        if (recExc) return recExc;
        if (res.command() == ERROR_MODE) return {res.error()};
    }
}

freighter::Error Writer::close() {
    auto exc = stream->closeSend();
    if (exc) return exc;
    while (true) {
        auto [_, recExc] = stream->receive();
        if (recExc) {
            if (recExc.type == freighter::EOF_.type) return freighter::NIL;
            return recExc;
        }
    }
}


void Writer::assertOpen() const {
    if (closed)
        throw std::runtime_error("cannot call method on closed writer");
}
