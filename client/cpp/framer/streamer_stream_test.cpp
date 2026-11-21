// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <memory>
#include <string>
#include <utility>

#include "gtest/gtest.h"

#include "client/cpp/framer/framer.h"
#include "freighter/cpp/freighter.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/testutil/testutil.h"

namespace synnax::framer {
namespace {
/// @brief a streamer stream that answers the open, sends the given number of frames,
/// and then ends with the given status.
class FramesThenEndStream final : public StreamerStream {
    x::errors::Error status;
    int frames;
    bool answered = false;

public:
    FramesThenEndStream(const int frames, const x::errors::Error &status):
        status(status), frames(frames) {}

    std::pair<grpc::framer::StreamerResponse, x::errors::Error> receive() override {
        if (!this->answered) {
            this->answered = true;
            return {grpc::framer::StreamerResponse(), x::errors::NIL};
        }
        if (this->frames > 0) {
            --this->frames;
            return {grpc::framer::StreamerResponse(), x::errors::NIL};
        }
        return {grpc::framer::StreamerResponse(), this->status};
    }

    x::errors::Error send(grpc::framer::StreamerRequest &) const override {
        return x::errors::NIL;
    }

    void close_send() override {}
};

/// @brief a streamer transport that opens FramesThenEndStream streams.
class FramesThenEndClient final : public StreamerClient {
    int frames;
    x::errors::Error status;

public:
    FramesThenEndClient(const int frames, const x::errors::Error &status):
        frames(frames), status(status) {}

    void use(std::shared_ptr<freighter::Middleware>) override {}

    std::pair<std::unique_ptr<StreamerStream>, x::errors::Error>
    stream(const std::string &) override {
        return {
            std::make_unique<FramesThenEndStream>(this->frames, this->status),
            x::errors::NIL
        };
    }
};

Client frames_then_end_client(const int frames, const x::errors::Error &status) {
    return {
        std::make_unique<FramesThenEndClient>(frames, status),
        nullptr,
        channel::Client()
    };
}

/// @brief answers the open, then sends a missed marker and one frame. Records whether
/// the open request asked for missed markers.
class MissedThenFrameStream final : public StreamerStream {
    int received = 0;
    std::shared_ptr<bool> requested;

public:
    explicit MissedThenFrameStream(std::shared_ptr<bool> requested):
        requested(std::move(requested)) {}

    std::pair<grpc::framer::StreamerResponse, x::errors::Error> receive() override {
        grpc::framer::StreamerResponse res;
        if (this->received++ == 1) res.set_missed(true);
        return {res, x::errors::NIL};
    }

    x::errors::Error send(grpc::framer::StreamerRequest &req) const override {
        *this->requested = req.missed_requested();
        return x::errors::NIL;
    }

    void close_send() override {}
};

class MissedThenFrameClient final : public StreamerClient {
    std::shared_ptr<bool> requested;

public:
    explicit MissedThenFrameClient(std::shared_ptr<bool> requested):
        requested(std::move(requested)) {}

    void use(std::shared_ptr<freighter::Middleware>) override {}

    std::pair<std::unique_ptr<StreamerStream>, x::errors::Error>
    stream(const std::string &) override {
        return {
            std::make_unique<MissedThenFrameStream>(this->requested),
            x::errors::NIL
        };
    }
};
}

/// @brief it should return the error the stream ends with when frames are still in
/// flight at close.
TEST(StreamerStream, CloseReturnsErrorAfterInFlightFrames) {
    const auto client = frames_then_end_client(2, x::errors::VALIDATION);
    const auto streamer = ASSERT_NIL_P(client.open_streamer(StreamerConfig{}));
    ASSERT_OCCURRED_AS(streamer.close(), x::errors::VALIDATION);
}

/// @brief it should return no error when the stream ends with EOF after in-flight
/// frames.
TEST(StreamerStream, CloseIgnoresEOFAfterInFlightFrames) {
    const auto client = frames_then_end_client(2, freighter::ERR_EOF);
    const auto streamer = ASSERT_NIL_P(client.open_streamer(StreamerConfig{}));
    ASSERT_NIL(streamer.close());
}

/// @brief it should call on_missed for a missed marker and return an empty frame.
TEST(StreamerStream, ReadCallsOnMissed) {
    const auto requested = std::make_shared<bool>(false);
    const Client client{
        std::make_unique<MissedThenFrameClient>(requested),
        nullptr,
        channel::Client()
    };
    int missed = 0;
    auto streamer = ASSERT_NIL_P(
        client.open_streamer(StreamerConfig{.on_missed = [&missed] { ++missed; }})
    );
    const auto frame = ASSERT_NIL_P(streamer.read());
    ASSERT_TRUE(*requested);
    ASSERT_TRUE(frame.empty());
    ASSERT_EQ(missed, 1);
}
}
