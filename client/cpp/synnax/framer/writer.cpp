// Copyright 2023 Synnax Labs, Inc.
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
#include "v1/framer.pb.h"

/// internal
#include "synnax/framer/framer.h"

const std::string WRITE_ENDPOINT = "/frame/write";

using namespace synnax;


const int32_t OPEN = 0;
const int32_t WRITE = 1;
const int32_t COMMIT = 2;
const int32_t ERROR = 3;
const int32_t SET_AUTHORITY = 4;


bool Writer::write(Frame fr) {
    auto req = api::v1::FrameWriterRequest();
    req.set_command(WRITE);
    fr.to_proto(req.mutable_frame());
    auto exc = stream->send(req);
    if (!exc.ok()) throw exc;
    return true;
}

std::pair<synnax::TimeStamp, bool> Writer::commit() {
    auto req = api::v1::FrameWriterRequest();
    req.set_command(COMMIT);
    auto exc = stream->send(req);
    if (!exc.ok()) throw exc;
    while (true) {
        auto [res, recExc] = stream->receive();
        if (!recExc.ok()) throw recExc;
        if (res.command() == COMMIT) return {synnax::TimeStamp(res.end()), true};
    }
}

//Writer::error() {
//    auto req = api::v1::FrameWriterRequest();
//    req.set_command(ERROR);
//    auto exc = stream->send(req);
//    if (!exc.ok()) throw exc;
//    while (true) {
//        auto [res, recExc] = stream->receive();
//        if (!recExc.ok()) throw recExc;
//        if (res.command() == ERROR) return std::exception()
//    }
//}

void Writer::close() {
    auto exc = stream->closeSend();
    if (!exc.ok()) throw exc;
    auto [_, recExc] = stream->receive();
    if (!recExc.ok()) throw recExc;
}

void WriterConfig::to_proto(api::v1::FrameWriterConfig *f) const {
    subject.to_proto(f->mutable_control_subject());
    f->set_start(start.value);
    for (auto &auth: authorities) f->add_authorities(auth);
    for (auto &ch: channels) f->add_keys(ch);
}

Writer::Writer(WriterStream *s, const WriterConfig &config) {
    stream = s;
    auto req = api::v1::FrameWriterRequest();
    req.set_command(OPEN);
    config.to_proto(req.mutable_config());
    auto exc = stream->send(req);
    if (!exc.ok()) throw exc;
    auto [_, resExc] = stream->receive();
    if (!resExc.ok()) throw exc;
}

