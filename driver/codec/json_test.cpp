// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <string>
#include <type_traits>
#include <utility>
#include <vector>

#include "gtest/gtest.h"

#include "client/cpp/icd/types.gen.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/testutil/testutil.h"
#include "x/cpp/uuid/uuid.h"

#include "driver/codec/errors.h"
#include "driver/codec/json.h"

namespace driver::codec {
namespace {
namespace icd = synnax::icd;

/// @brief returns a message named reading with one field v of type F at /v, bound to a
/// channel of dt.
template<typename F>
JSONMessage reading(const x::telem::DataType &dt) {
    F f;
    f.key = x::uuid::create();
    f.name = "v";
    f.pointer = "/v";
    if constexpr (std::is_same_v<F, icd::TimestampJSONField>) f.time_format = "iso8601";
    return {
        .name = "reading",
        .payload = {.fields = {f}},
        .bindings = {{.field = f.key, .data_type = dt}},
    };
}

/// @brief returns the error that creating a decoder for one field of type F bound to a
/// channel of dt gives.
template<typename F>
x::errors::Error create_error(const x::telem::DataType &dt) {
    return JSON::create({reading<F>(dt)}).second;
}
}

/// @brief it should write each field kind to the data types it converts to.
TEST(JSON, ConvertsKindsToDataTypes) {
    ASSERT_NIL(create_error<icd::StringJSONField>(x::telem::STRING_T));
    ASSERT_NIL(create_error<icd::IntegerJSONField>(x::telem::FLOAT32_T));
    ASSERT_NIL(create_error<icd::UnsignedJSONField>(x::telem::FLOAT64_T));
    ASSERT_NIL(create_error<icd::BooleanJSONField>(x::telem::FLOAT64_T));
    ASSERT_NIL(create_error<icd::UnsignedJSONField>(x::telem::INT64_T));
    ASSERT_NIL(create_error<icd::BooleanJSONField>(x::telem::INT8_T));
    ASSERT_NIL(create_error<icd::BooleanJSONField>(x::telem::UINT8_T));
}

/// @brief it should reject a field kind that does not convert to its data type.
TEST(JSON, RejectsKindsThatDoNotConvert) {
    const auto check = [](const x::errors::Error &err, const std::string &data) {
        ASSERT_OCCURRED_AS(err, CONFIG_ERROR);
        ASSERT_EQ(err.data, "message reading: field v: " + data);
    };
    check(
        create_error<icd::FloatJSONField>(x::telem::INT32_T),
        "cannot write float values to int32 channels"
    );
    check(
        create_error<icd::IntegerJSONField>(x::telem::UINT32_T),
        "cannot write integer values to uint32 channels"
    );
    check(
        create_error<icd::TimestampJSONField>(x::telem::INT64_T),
        "cannot write timestamp values to int64 channels"
    );
    check(
        create_error<icd::StringJSONField>(x::telem::FLOAT64_T),
        "cannot write string values to float64 channels"
    );
    check(
        create_error<icd::FloatJSONField>(x::telem::DataType("complex128")),
        "cannot write float values to complex128 channels"
    );
}

/// @brief it should replace the samples of a message on each decode.
TEST(JSON, ReplacesSamplesOnEachDecode) {
    auto dec = ASSERT_NIL_P(
        JSON::create({reading<icd::FloatJSONField>(x::telem::FLOAT64_T)})
    );
    ASSERT_EQ(ASSERT_NIL_P(dec.decode(R"({"v": 1})")), 0);
    ASSERT_EQ(ASSERT_NIL_P(dec.decode(R"({"v": 2})")), 0);
    ASSERT_EQ(dec.samples(0)[0].values<double>(), std::vector<double>{2});
}

/// @brief it should reject a document that is not JSON.
TEST(JSON, RejectsDocumentThatIsNotJSON) {
    auto dec = ASSERT_NIL_P(
        JSON::create({reading<icd::FloatJSONField>(x::telem::FLOAT64_T)})
    );
    const auto [message, err] = dec.decode("not json");
    ASSERT_OCCURRED_AS(err, x::errors::VALIDATION);
    ASSERT_EQ(err.data, "invalid character 'o' after offset 1");
}
}
