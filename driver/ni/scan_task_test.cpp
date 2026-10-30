// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <atomic>
#include <cstdint>
#include <cstring>
#include <mutex>
#include <string>
#include <vector>

#include "absl/log/globals.h"
#include "absl/log/log_entry.h"
#include "absl/log/log_sink.h"
#include "absl/log/log_sink_registry.h"
#include "gtest/gtest.h"

#include "client/cpp/testutil/testutil.h"
#include "x/cpp/json/json.h"
#include "x/cpp/testutil/testutil.h"

#include "driver/ni/ni.h"
#include "driver/ni/scan_task.h"
#include "driver/ni/syscfg/api.h"

namespace driver::ni {
/// @brief it should parse scan task configuration with defaults and custom values.
TEST(ScanTaskTest, testConfigParse) {
    // Test default configuration
    x::json::json j = {{"disabled", false}};
    auto p = x::json::Parser(j);
    ScanTaskConfig cfg(p);

    EXPECT_FALSE(cfg.disabled);
    EXPECT_EQ(cfg.rate.hz(), common::DEFAULT_SCAN_RATE.hz());

    // Test custom configuration
    x::json::json j2 = {{"disabled", true}, {"rate", 10.0}};
    auto p2 = x::json::Parser(j2);
    ScanTaskConfig cfg2(p2);

    EXPECT_TRUE(cfg2.disabled);
    EXPECT_EQ(cfg2.rate.hz(), 10.0);
}

/// @brief to_synnax should include is_chassis in properties JSON.
TEST(NiDeviceTests, testToSynnaxIncludesIsChassisInProperties) {
    ni::Device chassis;
    chassis.key = "chassis-1";
    chassis.name = "NI cDAQ-9178";
    chassis.rack = 1;
    chassis.location = "cDAQ1";
    chassis.make = "NI";
    chassis.model = "cDAQ-9178";
    chassis.is_chassis = true;
    chassis.is_simulated = false;
    chassis.resource_name = "cDAQ1";

    auto synnax_dev = chassis.to_synnax();
    x::json::json props(synnax_dev.properties);
    EXPECT_TRUE(props.contains("is_chassis"));
    EXPECT_TRUE(props["is_chassis"].get<bool>());
    EXPECT_TRUE(props.contains("is_simulated"));
    EXPECT_FALSE(props["is_simulated"].get<bool>());
    EXPECT_EQ(props["resource_name"].get<std::string>(), "cDAQ1");
}

/// @brief to_synnax should set is_chassis to false for modules.
TEST(NiDeviceTests, testToSynnaxModuleIsChassisIsFalse) {
    ni::Device module;
    module.key = "module-2";
    module.name = "NI 9205";
    module.rack = 1;
    module.location = "Slot2";
    module.make = "NI";
    module.model = "9205";
    module.is_chassis = false;

    auto synnax_dev = module.to_synnax();
    x::json::json props(synnax_dev.properties);
    EXPECT_FALSE(props["is_chassis"].get<bool>());
}

/// @brief to_synnax should preserve status through conversion.
TEST(NiDeviceTests, testToSynnaxPreservesStatus) {
    ni::Device dev;
    dev.key = "status-dev";
    dev.name = "NI 9205";
    dev.rack = 1;
    dev.location = "Slot1";
    dev.make = "NI";
    dev.model = "9205";
    dev.status = synnax::device::Status{};
    dev.status->variant = synnax::status::VARIANT_SUCCESS;
    dev.status->message = "Device present";

    auto synnax_dev = dev.to_synnax();
    ASSERT_TRUE(synnax_dev.status.has_value());
    EXPECT_EQ(synnax_dev.status.value().variant, synnax::status::VARIANT_SUCCESS);
    EXPECT_EQ(synnax_dev.status.value().message, "Device present");
}

/// @brief an in-memory NI System Configuration API that enumerates physical devices.
class FakeAPI final : public syscfg::API {
public:
    struct Resource {
        std::string serial;
        std::string product;
    };

private:
    std::vector<Resource> resources;
    std::size_t next = 0;

    [[nodiscard]] const Resource &at(NISysCfgResourceHandle handle) const {
        return this->resources[reinterpret_cast<std::uintptr_t>(handle) - 1];
    }

public:
    explicit FakeAPI(std::vector<Resource> resources):
        resources(std::move(resources)) {}

    NISYSCFGCFUNC InitializeSession(
        const char *,
        const char *,
        const char *,
        NISysCfgLocale,
        NISysCfgBool,
        unsigned int,
        NISysCfgEnumExpertHandle *,
        NISysCfgSessionHandle *
    ) override {
        return NISysCfg_OK;
    }

    NISYSCFGCFUNC CreateFilter(NISysCfgSessionHandle, NISysCfgFilterHandle *) override {
        return NISysCfg_OK;
    }

    NISYSCFGCDECL
    SetFilterProperty(NISysCfgFilterHandle, NISysCfgFilterProperty, ...) override {
        return NISysCfg_OK;
    }

    NISYSCFGCFUNC CloseHandle(void *) override { return NISysCfg_OK; }

    NISYSCFGCFUNC FindHardware(
        NISysCfgSessionHandle,
        NISysCfgFilterMode,
        NISysCfgFilterHandle,
        const char *,
        NISysCfgEnumResourceHandle *
    ) override {
        this->next = 0;
        return NISysCfg_OK;
    }

    NISYSCFGCFUNC NextResource(
        NISysCfgSessionHandle,
        NISysCfgEnumResourceHandle,
        NISysCfgResourceHandle *handle
    ) override {
        if (this->next == this->resources.size()) return NISysCfg_EndOfEnum;
        *handle = reinterpret_cast<NISysCfgResourceHandle>(++this->next);
        return NISysCfg_OK;
    }

    NISYSCFGCFUNC GetResourceProperty(
        NISysCfgResourceHandle handle,
        NISysCfgResourceProperty property,
        void *value
    ) override {
        const auto &res = this->at(handle);
        switch (property) {
            case NISysCfgResourcePropertyIsSimulated:
            case NISysCfgResourcePropertyIsChassis:
                *static_cast<NISysCfgBool *>(value) = NISysCfgBoolFalse;
                return NISysCfg_OK;
            case NISysCfgResourcePropertySerialNumber:
                std::strcpy(static_cast<char *>(value), res.serial.c_str());
                return NISysCfg_OK;
            case NISysCfgResourcePropertyProductName:
                std::strcpy(static_cast<char *>(value), res.product.c_str());
                return NISysCfg_OK;
            default:
                return NISysCfg_PropDoesNotExist;
        }
    }

    NISYSCFGCFUNC GetResourceIndexedProperty(
        NISysCfgResourceHandle,
        NISysCfgIndexedProperty,
        unsigned int,
        void *value
    ) override {
        std::strcpy(static_cast<char *>(value), "\"Dev1\"");
        return NISysCfg_OK;
    }

    NISYSCFGCFUNC
    GetStatusDescriptionW(NISysCfgSessionHandle, NISysCfgStatus, wchar_t **) override {
        return NISysCfg_NotImplemented;
    }

    NISYSCFGCFUNC
    SetFilterPropertyV(NISysCfgFilterHandle, NISysCfgFilterProperty, va_list) override {
        return NISysCfg_OK;
    }

    NISYSCFGCFUNC FreeDetailedStringW(wchar_t[]) override { return NISysCfg_OK; }
};

/// @brief returns a System Configuration library that loads api on first use.
std::shared_ptr<x::lib::Lazy<syscfg::SugaredAPI>>
loaded(const std::shared_ptr<syscfg::API> &api) {
    return std::make_shared<x::lib::Lazy<syscfg::SugaredAPI>>([api] {
        return std::pair{std::make_shared<syscfg::SugaredAPI>(api), x::errors::NIL};
    });
}

/// @brief enables VLOG(1) and captures the text of every VLOG(1) line while in scope.
class CaptureSink final : public absl::LogSink {
    int prev_vlog_level;
    std::mutex mu;
    std::vector<std::string> lines;

public:
    CaptureSink(): prev_vlog_level(absl::SetGlobalVLogLevel(1)) {
        absl::AddLogSink(this);
    }

    ~CaptureSink() override {
        absl::RemoveLogSink(this);
        absl::SetGlobalVLogLevel(this->prev_vlog_level);
    }

    void Send(const absl::LogEntry &entry) override {
        if (entry.verbosity() != 1) return;
        std::lock_guard lock(this->mu);
        this->lines.emplace_back(entry.text_message());
    }

    [[nodiscard]] std::size_t count(const std::string &substr) {
        std::lock_guard lock(this->mu);
        return std::ranges::count_if(this->lines, [&](const std::string &line) {
            return line.contains(substr);
        });
    }
};

/// @brief the scanner should log each device skipped by the model filter once, at
/// VLOG(1).
TEST(ScannerTest, testLogsIgnoredDeviceOnce) {
    auto api = std::make_shared<FakeAPI>(std::vector<FakeAPI::Resource>{
        {"022B5288", "NI cRIO-9040"},
        {"022B5299", "NI cRIO-9045"},
        {"01A2B3C4", "NI 9485"},
    });
    x::json::json j = {{"disabled", false}};
    auto p = x::json::Parser(j);
    Scanner scanner(loaded(api), ScanTaskConfig(p), synnax::task::Task{});
    CaptureSink sink;
    for (std::size_t i = 0; i < 3; i++) {
        auto [devices, err] = scanner.scan(common::ScannerContext{.count = i});
        ASSERT_FALSE(err) << err;
        ASSERT_EQ(devices.size(), 1);
        EXPECT_EQ(devices[0].key, "01A2B3C4");
    }
    EXPECT_EQ(sink.count("skipping cRIO-9040 (022B5288)"), 1);
    EXPECT_EQ(sink.count("skipping cRIO-9045 (022B5299)"), 1);
}

/// @brief returns the devices found by one scan of the given resources.
std::vector<synnax::device::Device> scan_once(
    std::vector<FakeAPI::Resource> resources,
    x::json::json j = {{"disabled", false}}
) {
    auto api = std::make_shared<FakeAPI>(std::move(resources));
    auto p = x::json::Parser(std::move(j));
    Scanner scanner(loaded(api), ScanTaskConfig(p), synnax::task::Task{});
    auto [devices, err] = scanner.scan(common::ScannerContext{});
    EXPECT_FALSE(err) << err;
    return devices;
}

/// @brief the scanner should skip cRIO and unknown devices whether or not their product
/// name starts with the make.
TEST(ScannerTest, testSkipsIgnoredModels) {
    const auto devices = scan_once({
        {"022B5288", "NI cRIO-9040"},
        {"022B5288", "cRIO-9040"},
        {"022B5299", "NI cRIO-9068"},
        {"01ABCDEF", "Unknown"},
        {"01ABCDF0", "UnknownDevice"},
        {"01A2B3C4", "NI 9485"},
    });
    ASSERT_EQ(devices.size(), 1);
    EXPECT_EQ(devices[0].key, "01A2B3C4");
    EXPECT_EQ(devices[0].model, "9485");
    EXPECT_EQ(devices[0].name, "NI 9485");
}

/// @brief the scanner should keep devices whose model does not match an ignored
/// pattern.
TEST(ScannerTest, testKeepsOtherModels) {
    const auto devices = scan_once({
        {"01000001", "NI PXI-6255"},
        {"01000002", "NI cDAQ-9178"},
        {"01000003", "NI NI-DAQ"},
        {"01000004", "Regular-Device"},
    });
    std::vector<std::string> models;
    models.reserve(devices.size());
    for (const auto &dev: devices)
        models.push_back(dev.model);
    EXPECT_EQ(
        models,
        std::vector<std::string>({"PXI-6255", "cDAQ-9178", "NI-DAQ", "Regular-Device"})
    );
}

/// @brief the scanner should ignore the ignored_models patterns of a saved config.
TEST(ScannerTest, testIgnoresSavedPatterns) {
    const auto devices = scan_once(
        {
            {"01000005", "NI TestDevice"},
            {"01000006", "NI MockDevice123"},
            {"01000001", "NI PXI-6255"},
            {"01ABCDEF", "Unknown"},
        },
        {{"ignored_models", x::json::json::array({"^Test.*", "^Mock.*", "PXI-.*"})}}
    );
    std::vector<std::string> models;
    models.reserve(devices.size());
    for (const auto &dev: devices)
        models.push_back(dev.model);
    EXPECT_EQ(
        models,
        std::vector<std::string>({"TestDevice", "MockDevice123", "PXI-6255"})
    );
}

/// @brief it should start the scanner without the System Configuration library, warn
/// while the library does not load, and clear the warning on the first scan after it
/// loads.
TEST(ScannerTest, testWarnsUntilLibraryLoads) {
    std::atomic<int> loads = 0;
    Factory factory(
        []() -> std::pair<std::shared_ptr<daqmx::SugaredAPI>, x::errors::Error> {
            return {nullptr, driver::errors::missing_lib(daqmx::LIBRARY_INFO)};
        },
        [&loads]() -> std::pair<std::shared_ptr<syscfg::SugaredAPI>, x::errors::Error> {
            if (loads++ == 0)
                return {nullptr, driver::errors::missing_lib(syscfg::LIBRARY_INFO)};
            return {
                std::make_shared<syscfg::SugaredAPI>(
                    std::make_shared<FakeAPI>(std::vector<FakeAPI::Resource>{})
                ),
                x::errors::NIL
            };
        }
    );
    const auto ctx = std::make_shared<task::MockContext>(
        std::make_shared<synnax::Synnax>(new_test_client())
    );
    auto [scan_task, handled] = factory.configure_task(
        ctx,
        synnax::task::Task{
            .name = "NI Scanner",
            .type = SCAN_TASK_TYPE,
            .config = {{"rate", 1}},
        },
        task::NO_COMMAND
    );
    ASSERT_TRUE(handled);
    ASSERT_NE(scan_task, nullptr);
    ASSERT_EVENTUALLY_GE_WITH_TIMEOUT(
        ctx->snapshot().size(),
        3,
        std::chrono::seconds(5),
        std::chrono::milliseconds(10)
    );
    scan_task->stop(false);
    const auto statuses = ctx->snapshot();
    EXPECT_EQ(statuses[0].variant, synnax::status::VARIANT_SUCCESS);
    EXPECT_EQ(statuses[0].message, "Started scan task");
    EXPECT_EQ(statuses[1].variant, synnax::status::VARIANT_WARNING);
    EXPECT_EQ(statuses[1].message, "Failed to scan for devices");
    EXPECT_EQ(
        statuses[1].description,
        "National Instruments System Configuration library is not installed. Download "
        "here: https://www.ni.com/en/support/downloads/drivers/"
        "download.system-configuration.html."
    );
    EXPECT_EQ(statuses[2].variant, synnax::status::VARIANT_SUCCESS);
    EXPECT_EQ(statuses[2].message, "Scan task running");
}
}
