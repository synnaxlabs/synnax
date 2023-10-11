#include <string>

#include "synnax/framer/framer.h"
#include "v1/framer.pb.h"

std::string WRITE_ENDPOINT = "/frame/write";

using namespace Synnax::Framer;


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
}

std::pair<Telem::TimeStamp, bool> Writer::commit() {
    auto req = api::v1::FrameWriterRequest();
    req.set_command(COMMIT);
    auto exc = stream->send(req);
    if (!exc.ok()) throw exc;
    while (true) {
        auto [res, recExc] = stream->receive();
        if (!recExc.ok()) throw recExc;
        if (res.command() == COMMIT) return {Telem::TimeStamp(res.end()), true};
    }
}

std::exception Writer::error() {
   auto req = api::v1::FrameWriterRequest();
    req.set_command(ERROR);
    auto exc = stream->send(req);
    if (!exc.ok()) throw exc;
    while (true) {
        auto [res, recExc] = stream->receive();
        if (!recExc.ok()) throw recExc;
        if (res.command() == ERROR) return std::exception(res.error().c_str());
    }
}

void Writer::close() {
    auto exc = stream->closeSend();
    if (!exc.ok()) throw exc;
    auto [_, recExc] = stream->receive();
    if (!recExc.ok()) throw recExc;
}

api::v1::FrameWriterConfig WriterConfig::to_proto(api::v1::FrameWriterConfig *f) const {
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

