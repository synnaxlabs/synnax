#include <string>

#include "synnax/framer/framer.h"

std::string STREAM_ENDPOINT = "/frame/stream";

using namespace Synnax::Framer;

Frame Streamer::read() {
    auto [fr, exc] = stream->receive();
    if (!exc.ok()) {
        throw;
    }
    return {fr.frame()};
}

void Streamer::close() {
    auto exc = stream->closeSend();
    if (!exc.ok()) {
        throw;
    }
    auto [res, recExc] = stream->receive();
    if (!recExc.ok()) {
        throw;
    }
}