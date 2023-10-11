#include "synnax/framer/framer.h"

using namespace Synnax::Framer;

const std::string ITERATOR_ENDPOINT = "/frame/iterate";
const std::string STREAM_ENDPOINT = "/frame/stream";
const std::string WRITE_ENDPOINT = "/frame/write";

Iterator Client::openIterator(const IteratorConfig &config) {
    auto s = iterator_client->stream(ITERATOR_ENDPOINT);
    return {s, config};
}

Streamer Client::openStreamer(const StreamerConfig &config) {
    auto s = streamer_client->stream(STREAM_ENDPOINT);
    return {s, config};
}

Writer Client::openWriter(const WriterConfig &config) {
    auto s = writer_client->stream(WRITE_ENDPOINT);
    return {s, config};
}