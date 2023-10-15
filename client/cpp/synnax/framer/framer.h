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


using namespace Synnax;

namespace Synnax::Framer {
typedef Freighter::Stream<
        api::v1::FrameIteratorResponse,
        api::v1::FrameIteratorRequest,
        grpc::Status> IteratorStream;

typedef Freighter::StreamClient<
        api::v1::FrameIteratorResponse,
        api::v1::FrameIteratorRequest,
        grpc::Status> IteratorClient;

typedef Freighter::Stream<
        api::v1::FrameStreamerResponse,
        api::v1::FrameStreamerRequest,
        grpc::Status> StreamerStream;

typedef Freighter::StreamClient<
        api::v1::FrameStreamerResponse,
        api::v1::FrameStreamerRequest,
        grpc::Status> StreamerClient;

typedef Freighter::Stream<
        api::v1::FrameWriterResponse,
        api::v1::FrameWriterRequest,
        grpc::Status> WriterStream;

typedef Freighter::StreamClient<
        api::v1::FrameWriterResponse,
        api::v1::FrameWriterRequest,
        grpc::Status> WriterClient;


/// @brief Frame type.
class Frame {
    std::vector<Channel::Key> *columns;
    std::vector<Telem::Series> *series;

public:
    Frame(std::vector<Channel::Key> *channels, std::vector<Telem::Series> *series);

    Frame(size_t size);

    explicit Frame(const api::v1::Frame &f);

    void to_proto(api::v1::Frame *f) const;

    void push_back(Channel::Key col, Telem::Series ser);

    size_t size() const { return series->size(); }

    std::pair<Channel::Key, Telem::Series> operator[](size_t i) const {
        return std::make_pair((*columns)[i], (*series)[i]);
    }
};

struct IteratorConfig {
    std::vector<Channel::Key> channels;
    Telem::TimeRange bounds;
};

class Iterator {
private:
public:
    Iterator(IteratorStream *stream, const IteratorConfig &config);

    bool next(Telem::TimeSpan span);

    bool prev(Telem::TimeSpan span);

    bool seekFirst();

    bool seekLast();

    bool seekLT(Telem::TimeStamp ts);

    bool seekGE(Telem::TimeStamp ts);

    bool valid();

    void close();
};

struct StreamerConfig {
    Telem::TimeStamp start;
    std::vector<Channel::Key> channels;
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
    std::vector<Telem::Authority> authorities;
    std::vector<Channel::Key> channels;
    Telem::Subject subject;
    Telem::TimeStamp start;

    void to_proto(api::v1::FrameWriterConfig *f) const;
};


class Writer {
private:
    WriterStream *stream{};
public:
    Writer(WriterStream *s, const WriterConfig &config);

    /// @brief Sends one frame to the given target.
    bool write(Frame fr);

    std::pair<Telem::TimeStamp, bool> commit();

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


    Iterator openIterator(const IteratorConfig &config);

    Writer openWriter(const WriterConfig &config);

    Streamer openStreamer(const StreamerConfig &config);
};
}
