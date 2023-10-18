// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <vector>
#include <utility>

#include "freighter/freighter.h"
#include "synnax/telem/telem.h"
#include "synnax/telem/series.h"
#include "synnax/telem/control.h"
#include "synnax/channel/channel.h"
#include "v1/framer.pb.h"
#include <grpcpp/grpcpp.h>


using namespace synnax;

namespace synnax::Framer {
typedef freighter::Stream<
        api::v1::FrameIteratorResponse,
        api::v1::FrameIteratorRequest
> IteratorStream;

typedef freighter::StreamClient<
        api::v1::FrameIteratorResponse,
        api::v1::FrameIteratorRequest
> IteratorClient;

typedef freighter::Stream<
        api::v1::FrameStreamerResponse,
        api::v1::FrameStreamerRequest
> StreamerStream;

typedef freighter::StreamClient<
        api::v1::FrameStreamerResponse,
        api::v1::FrameStreamerRequest
> StreamerClient;

typedef freighter::Stream<
        api::v1::FrameWriterResponse,
        api::v1::FrameWriterRequest
> WriterStream;

typedef freighter::StreamClient<
        api::v1::FrameWriterResponse,
        api::v1::FrameWriterRequest
> WriterClient;


/// @brief Frame type.
class Frame {
    std::vector<channel::Key> *columns;
    std::vector<synnax::Series> *series;

public:
    Frame(std::vector<channel::Key> *channels, std::vector<synnax::Series> *series);

    Frame(size_t size);

    explicit Frame(const api::v1::Frame &f);

    void to_proto(api::v1::Frame *f) const;

    void push_back(channel::Key col, synnax::Series ser);

    size_t size() const { return series->size(); }

    std::pair<channel::Key, synnax::Series> operator[](size_t i) const {
        return std::make_pair((*columns)[i], (*series)[i]);
    }
};

struct IteratorConfig {
    std::vector<channel::Key> channels;
    synnax::TimeRange bounds;
};

class Iterator {
private:
public:
    Iterator(IteratorStream *stream, const IteratorConfig &config);

    bool next(synnax::TimeSpan span);

    bool prev(synnax::TimeSpan span);

    bool seekFirst();

    bool seekLast();

    bool seekLT(synnax::TimeStamp ts);

    bool seekGE(synnax::TimeStamp ts);

    bool valid();

    void close();
};

struct StreamerConfig {
    synnax::TimeStamp start;
    std::vector<channel::Key> channels;
};

class Streamer {
private:
    StreamerStream *stream;
public:
    Streamer(StreamerStream *stream, const StreamerConfig &config);

    Frame read();

    void close();
};

struct WriterConfig {
    std::vector<synnax::Authority> authorities;
    std::vector<channel::Key> channels;
    synnax::Subject subject;
    synnax::TimeStamp start;

    void to_proto(api::v1::FrameWriterConfig *f) const;
};


class Writer {
private:
    WriterStream *stream{};
public:
    Writer(WriterStream *s, const WriterConfig &config);

    /// @brief Sends one frame to the given target.
    bool write(Frame fr);

    std::pair<synnax::TimeStamp, bool> commit();

//    std::exception error();

    void close();

private:
};

class Client {
private:
    IteratorClient *iterator_client;
    StreamerClient *streamer_client;
    WriterClient *writer_client;
public:
    Client(IteratorClient *iterator_client, StreamerClient *streamer_client, WriterClient *writer_client) :
            iterator_client(iterator_client),
            streamer_client(streamer_client),
            writer_client(writer_client) {}


    std::pair<Iterator, freighter::Error> openIterator(const IteratorConfig &config);

    std::pair<Writer, freighter::Error> openWriter(const WriterConfig &config);

    std::pair<Streamer, freighter::Error> openStreamer(const StreamerConfig &config);
};
}
