// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cstring>
#include <memory>
#include <string>
#include <unordered_map>
#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/testutil/testutil.h"

#include "driver/ni/daqmx/mock.h"
#include "driver/ni/daqmx/sugared.h"

namespace driver::ni::daqmx {
namespace {
const std::string ERROR_MESSAGE = "Specified property is not supported.";

/// @brief a DAQmx API that holds device attributes and answers a call with no buffer
/// with the size the value needs, as DAQmx does.
class FakeAPI final : public MockAPI {
public:
    std::unordered_map<int32, std::string> strings;
    std::unordered_map<int32, std::vector<float64>> doubles;
    /// @brief the buffer size of each attribute read, in call order.
    std::vector<uInt32> sizes;
    /// @brief the status that StartTask returns.
    int32 start_status = 0;

    int32 GetDeviceAttributeString(
        const char[],
        const int32 attribute,
        char value[],
        const uInt32 size
    ) override {
        this->sizes.push_back(size);
        const auto it = this->strings.find(attribute);
        if (it == this->strings.end()) return DAQmxErrorAttrNotSupported;
        if (size == 0) return static_cast<int32>(it->second.size() + 1);
        std::strncpy(value, it->second.c_str(), size);
        return 0;
    }

    int32 GetDeviceAttributeDoubleArray(
        const char[],
        const int32 attribute,
        float64 value[],
        const uInt32 size
    ) override {
        this->sizes.push_back(size);
        const auto it = this->doubles.find(attribute);
        if (it == this->doubles.end()) return DAQmxErrorAttrNotSupported;
        if (size == 0) return static_cast<int32>(it->second.size());
        std::ranges::copy(it->second, value);
        return 0;
    }

    int32 StartTask(TaskHandle) override { return this->start_status; }

    int32 GetExtendedErrorInfo(char errorString[], const uInt32 bufferSize) override {
        if (errorString == nullptr) return static_cast<int32>(ERROR_MESSAGE.size() + 1);
        std::strncpy(errorString, ERROR_MESSAGE.c_str(), bufferSize);
        return 0;
    }
};
}

/// @brief it should keep every digit of a positive status in the error type.
TEST(SugaredAPITest, testParsesPositiveStatus) {
    auto api = std::make_shared<FakeAPI>();
    api->start_status = 200123;
    SugaredAPI dmx(api);
    const auto err = dmx.StartTask(nullptr);
    EXPECT_EQ(err.type, CRITICAL_ERROR.sub("200123").type);
    EXPECT_EQ(err.data, ERROR_MESSAGE);
}

/// @brief it should keep every digit of a negative status in the error type.
TEST(SugaredAPITest, testParsesNegativeStatus) {
    auto api = std::make_shared<FakeAPI>();
    api->start_status = DAQmxErrorAttrNotSupported;
    SugaredAPI dmx(api);
    const auto err = dmx.StartTask(nullptr);
    EXPECT_EQ(err.type, ATTRIBUTE_NOT_SUPPORTED.type);
    EXPECT_EQ(err.data, ERROR_MESSAGE);
}

/// @brief it should ask for the size of a string first, so a long value is not cut.
TEST(SugaredAPITest, testReadsLongDeviceString) {
    auto api = std::make_shared<FakeAPI>();
    std::string channels;
    for (int i = 0; i < 80; i++)
        channels += (i == 0 ? "" : ", ") + std::string("Dev1/ai") + std::to_string(i);
    api->strings[DAQmx_Dev_AI_PhysicalChans] = channels;
    SugaredAPI dmx(api);
    EXPECT_EQ(
        ASSERT_NIL_P(dmx.GetDeviceAttributeString("Dev1", DAQmx_Dev_AI_PhysicalChans)),
        channels
    );
    EXPECT_EQ(
        api->sizes,
        std::vector<uInt32>({0, static_cast<uInt32>(channels.size() + 1)})
    );
}

/// @brief it should read an empty string as empty.
TEST(SugaredAPITest, testReadsEmptyDeviceString) {
    auto api = std::make_shared<FakeAPI>();
    api->strings[DAQmx_Dev_Terminals] = "";
    SugaredAPI dmx(api);
    EXPECT_EQ(
        ASSERT_NIL_P(dmx.GetDeviceAttributeString("Dev1", DAQmx_Dev_Terminals)),
        ""
    );
}

/// @brief it should return the error of the size query without a second read.
TEST(SugaredAPITest, testReturnsSizeQueryError) {
    auto api = std::make_shared<FakeAPI>();
    SugaredAPI dmx(api);
    const auto [value, err] = dmx.GetDeviceAttributeString(
        "Dev1",
        DAQmx_Dev_ProductType
    );
    EXPECT_EQ(value, "");
    EXPECT_EQ(err.type, ATTRIBUTE_NOT_SUPPORTED.type);
    EXPECT_EQ(api->sizes, std::vector<uInt32>({0}));
}

/// @brief it should read a float64 array of the size that DAQmx gives.
TEST(SugaredAPITest, testReadsDeviceDoubleArray) {
    auto api = std::make_shared<FakeAPI>();
    api->doubles[DAQmx_Dev_AI_VoltageRngs] = {-10, 10, -5, 5, -0.2, 0.2};
    SugaredAPI dmx(api);
    EXPECT_EQ(
        ASSERT_NIL_P(
            dmx.GetDeviceAttributeDoubleArray("Dev1", DAQmx_Dev_AI_VoltageRngs)
        ),
        std::vector<float64>({-10, 10, -5, 5, -0.2, 0.2})
    );
    EXPECT_EQ(api->sizes, std::vector<uInt32>({0, 6}));
}
}
