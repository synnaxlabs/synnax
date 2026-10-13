// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <vector>

#include "gtest/gtest.h"

#include "client/cpp/framer/framer.h"
#include "client/cpp/testutil/testutil.h"
#include "x/cpp/test/test.h"

namespace synnax::framer {
/// @brief it should allow the caller to dynamically update the keys fo the codec.
TEST(CodecTests, DynamicCodecUpdate) {
    auto client = new_test_client();

    auto [idx_ch, data_ch] = create_indexed_pair(client);
    Codec codec(client.channels);

    codec.update(std::vector{idx_ch.key});

    auto frame = x::telem::Frame(
        idx_ch.key,
        x::telem::Series(x::telem::TimeStamp(x::telem::SECOND))
    );

    std::vector<uint8_t> encoded;
    ASSERT_NIL(codec.encode(frame, encoded));
    auto decoded_frame = ASSERT_NIL_P(codec.decode(encoded));
    assert_frames_equal(frame, decoded_frame);

    codec.update(std::vector{data_ch.key});
    auto frame2 = x::telem::Frame(data_ch.key, x::telem::Series(1.0f));
    ASSERT_NIL(codec.encode(frame2, encoded));
    auto decoded_frame2 = ASSERT_NIL_P(codec.decode(encoded));
    assert_frames_equal(frame2, decoded_frame2);
}

/// @brief it should correctly encode/decode values when the codec are out of sync
TEST(CodecTests, UninitializedCodec) {
    auto client = new_test_client();
    Codec codec(client.channels);

    auto [idx_ch, _] = create_indexed_pair(client);
    auto frame = x::telem::Frame(
        idx_ch.key,
        x::telem::Series(x::telem::TimeStamp(x::telem::SECOND))
    );

    std::vector<uint8_t> encoded;
    ASSERT_THROW(codec.encode(frame, encoded), std::runtime_error);
}

/// @brief it should correctly manage the lifecycle of codecs that are temporarily
/// out of sync by using historical states.
TEST(CodecTests, OutOfSyncCodecs) {
    auto client = new_test_client();
    auto [idx_ch, data_ch] = create_indexed_pair(client);

    Codec encoder(client.channels);
    Codec decoder(client.channels);

    // Initial state - both in sync
    ASSERT_NIL(encoder.update(std::vector{idx_ch.key}));
    ASSERT_NIL(decoder.update(std::vector{idx_ch.key}));

    auto frame = x::telem::Frame(
        idx_ch.key,
        x::telem::Series(x::telem::TimeStamp(x::telem::SECOND))
    );

    std::vector<uint8_t> encoded;
    ASSERT_NIL(encoder.encode(frame, encoded));
    auto decoded_frame = ASSERT_NIL_P(decoder.decode(encoded));
    assert_frames_equal(frame, decoded_frame);

    // Decoder updates but encoder doesn't - should still work with old format
    ASSERT_NIL(decoder.update(std::vector{data_ch.key}));
    ASSERT_NIL(encoder.encode(frame, encoded));
    auto decoded_frame2 = ASSERT_NIL_P(decoder.decode(encoded));
    assert_frames_equal(frame, decoded_frame2);

    // Encoder updates - old frame should now fail
    ASSERT_NIL(encoder.update(std::vector{data_ch.key}));
    ASSERT_OCCURRED_AS(encoder.encode(frame, encoded), x::errors::VALIDATION);

    // New frame with updated channel should work
    auto frame2 = x::telem::Frame(data_ch.key, x::telem::Series(1.0f));
    ASSERT_NIL(encoder.encode(frame2, encoded));
    auto decoded_frame3 = ASSERT_NIL_P(decoder.decode(encoded));
    assert_frames_equal(frame2, decoded_frame3);
}
}
