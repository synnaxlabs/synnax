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
#include <unordered_map>
#include <utility>
#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/lib/lib.h"
#include "x/cpp/test/test.h"

#include "driver/bus/scan.h"
#include "driver/errors/errors.h"

namespace driver::bus {
namespace {
struct Info {
    std::uint16_t card = 0;
    std::uint16_t channel = 0;
    std::string name;
};

/// @brief a backend that lists fixed channels or fails with a fixed error.
struct Backend {
    std::vector<Info> infos;
    x::errors::Error err;

    [[nodiscard]] std::pair<std::vector<Info>, x::errors::Error> list() const {
        return {this->infos, this->err};
    }
};

using Backends = std::unordered_map<std::string, std::shared_ptr<Backend>>;

Scanner<Backend> create_scanner(Backends backends) {
    return {
        synnax::task::Task{.rack = 7, .name = "scan"},
        "arinc429",
        "ARINC 429",
        std::move(backends),
    };
}
}

TEST(Scanner, ReportsADeviceForEachListedChannel) {
    auto scanner = create_scanner({
        {"simulated",
         std::make_shared<Backend>(Backend{.infos = {{0, 1, "Simulated channel"}}})},
    });
    const auto devs = ASSERT_NIL_P(scanner.scan({}));
    ASSERT_EQ(devs.size(), 1);
    EXPECT_EQ(devs[0].key, "arinc429_7_simulated_0_1");
    EXPECT_EQ(devs[0].make, "ARINC 429");
    EXPECT_EQ(devs[0].model, "simulated");
    EXPECT_EQ(devs[0].name, "Simulated channel");
    EXPECT_EQ(devs[0].rack, 7);
    EXPECT_EQ(devs[0].properties.at("backend"), "simulated");
    EXPECT_EQ(devs[0].properties.at("card"), 0);
    EXPECT_EQ(devs[0].properties.at("channel"), 1);
    ASSERT_TRUE(devs[0].status.has_value());
    EXPECT_EQ(devs[0].status->variant, synnax::status::VARIANT_SUCCESS);
}

TEST(Scanner, KeepsATrackedDeviceOnTheSameChannel) {
    auto scanner = create_scanner({
        {"simulated", std::make_shared<Backend>(Backend{.infos = {{0, 0, "sim"}}})},
    });
    const std::unordered_map<std::string, synnax::device::Device> tracked{
        {"mine",
         synnax::device::Device{
             .key = "mine",
             .name = "My channel",
             .properties = {
                 {"backend", "simulated"},
                 {"card", 0},
                 {"channel", 0},
                 {"speed", "low"},
             },
         }},
    };
    const auto devs = ASSERT_NIL_P(scanner.scan({.devices = &tracked}));
    ASSERT_EQ(devs.size(), 1);
    EXPECT_EQ(devs[0].key, "mine");
    EXPECT_EQ(devs[0].properties.at("speed"), "low");
}

TEST(Scanner, SkipsABackendWhoseLibraryIsMissingOrUnsupported) {
    auto scanner = create_scanner({
        {"ddc",
         std::make_shared<Backend>(
             Backend{.err = x::errors::Error(x::lib::LOAD_ERROR)}
         )},
        {"ballard",
         std::make_shared<Backend>(Backend{.err = vendor::unsupported("Ballard")})},
    });
    EXPECT_TRUE(ASSERT_NIL_P(scanner.scan({})).empty());
}

TEST(Scanner, ReturnsAnyOtherErrorOfABackend) {
    auto scanner = create_scanner({
        {"ddc",
         std::make_shared<Backend>(Backend{
             .err = x::errors::Error(errors::CRITICAL_HARDWARE_ERROR, "card fault")
         })},
    });
    ASSERT_OCCURRED_AS_P(scanner.scan({}), errors::CRITICAL_HARDWARE_ERROR);
}
}
