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
#include <utility>
#include <vector>

#include "absl/base/log_severity.h"
#include "absl/log/globals.h"
#include "absl/log/log_entry.h"
#include "absl/log/log_sink.h"
#include "absl/log/log_sink_registry.h"
#include "gtest/gtest.h"

#include "x/cpp/json/json.h"
#include "x/cpp/test/test.h"

#include "driver/ni/scan_task.h"
#include "driver/ni/syscfg/api.h"
#include "driver/ni/syscfg/sugared.h"

namespace driver::ni {
namespace {
/// @brief syscfg::API that enumerates a single physical device.
class SingleDeviceAPI final : public syscfg::API {
    std::string serial;
    std::string product_name;
    bool enumerated = false;

    static NISysCfgStatus copy(void *value, const std::string &str) {
        std::strcpy(static_cast<char *>(value), str.c_str());
        return NISysCfg_OK;
    }

public:
    SingleDeviceAPI(std::string serial, std::string product_name):
        serial(std::move(serial)), product_name(std::move(product_name)) {}

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
        return NISysCfg_OK;
    }

    NISYSCFGCFUNC NextResource(
        NISysCfgSessionHandle,
        NISysCfgEnumResourceHandle,
        NISysCfgResourceHandle *
    ) override {
        if (this->enumerated) return NISysCfg_EndOfEnum;
        this->enumerated = true;
        return NISysCfg_OK;
    }

    NISYSCFGCFUNC GetResourceProperty(
        NISysCfgResourceHandle,
        const NISysCfgResourceProperty propertyID,
        void *value
    ) override {
        switch (propertyID) {
            case NISysCfgResourcePropertyIsSimulated:
            case NISysCfgResourcePropertyIsChassis:
                *static_cast<NISysCfgBool *>(value) = NISysCfgBoolFalse;
                return NISysCfg_OK;
            case NISysCfgResourcePropertySerialNumber:
                return copy(value, this->serial);
            case NISysCfgResourcePropertyProductName:
                return copy(value, this->product_name);
            default:
                return copy(value, "");
        }
    }

    NISYSCFGCFUNC GetResourceIndexedProperty(
        NISysCfgResourceHandle,
        const NISysCfgIndexedProperty propertyID,
        unsigned int,
        void *value
    ) override {
        if (propertyID == NISysCfgIndexedPropertyExpertResourceName)
            return copy(value, "\"Mod1\"");
        return copy(value, "Mod1");
    }

    NISYSCFGCFUNC
    GetStatusDescriptionW(NISysCfgSessionHandle, NISysCfgStatus, wchar_t **) override {
        return NISysCfg_OK;
    }

    NISYSCFGCFUNC
    SetFilterPropertyV(NISysCfgFilterHandle, NISysCfgFilterProperty, va_list) override {
        return NISysCfg_OK;
    }

    NISYSCFGCFUNC FreeDetailedStringW(wchar_t[]) override { return NISysCfg_OK; }
};

/// @brief records every log entry at VLOG level 1 for the lifetime of the capture.
class LogCapture final : public absl::LogSink {
    int prev_vlog_level;
    std::vector<std::pair<absl::LogSeverity, std::string>> entries;

public:
    LogCapture(): prev_vlog_level(absl::SetGlobalVLogLevel(1)) {
        absl::AddLogSink(this);
    }

    ~LogCapture() override {
        absl::RemoveLogSink(this);
        absl::SetGlobalVLogLevel(this->prev_vlog_level);
    }

    void Send(const absl::LogEntry &entry) override {
        this->entries.emplace_back(entry.log_severity(), entry.text_message());
    }

    /// @brief returns the severity of every entry whose text equals msg.
    [[nodiscard]] std::vector<absl::LogSeverity>
    severities(const std::string &msg) const {
        std::vector<absl::LogSeverity> out;
        for (const auto &[severity, text]: this->entries)
            if (text == msg) out.push_back(severity);
        return out;
    }
};

/// @brief runs one scan over a single device and expects it to be filtered.
void scan_filtered(const std::string &serial, const std::string &product_name) {
    const auto api = std::make_shared<syscfg::SugaredAPI>(
        std::make_shared<SingleDeviceAPI>(serial, product_name)
    );
    x::json::json j = x::json::json::object();
    auto p = x::json::Parser(j);
    Scanner scanner(api, ScanTaskConfig(p), synnax::task::Task{});
    ASSERT_NIL(scanner.start());
    const auto [devices, err] = scanner.scan(common::ScannerContext{});
    ASSERT_NIL(err);
    EXPECT_TRUE(devices.empty());
    ASSERT_NIL(scanner.stop());
}
}

/// @brief it should log a filtered cRIO only at VLOG(1).
TEST(ScanTaskTest, testFilteredCRIOLogsVerbose) {
    const LogCapture logs;
    scan_filtered("022B5288", "NI cRIO-9040");
    EXPECT_EQ(
        logs.severities(
            "[ni.scan_task] device ignored by filter: 022B5288 (model: cRIO-9040)"
        ),
        std::vector{absl::LogSeverity::kInfo}
    );
}

/// @brief it should log a filtered non-cRIO device as a warning.
TEST(ScanTaskTest, testFilteredUnknownLogsWarning) {
    const LogCapture logs;
    scan_filtered("01ABCDEF", "Unknown");
    EXPECT_EQ(
        logs.severities(
            "[ni.scan_task] device ignored by filter: 01ABCDEF (model: nown)"
        ),
        std::vector{absl::LogSeverity::kWarning}
    );
}
/// @brief it should parse scan task configuration with defaults and custom values.
TEST(ScanTaskTest, testConfigParse) {
    // Test default configuration
    x::json::json j = {{"disabled", false}};
    auto p = x::json::Parser(j);
    ScanTaskConfig cfg(p);

    EXPECT_FALSE(cfg.disabled);
    EXPECT_EQ(cfg.rate.hz(), common::DEFAULT_SCAN_RATE.hz());
    EXPECT_EQ(cfg.ignored_models.size(), DEFAULT_IGNORED_MODELS.size());

    // Test custom configuration
    x::json::json j2 = {
        {"disabled", true},
        {"rate", 10.0},
        {"ignored_models", x::json::json::array({"^Test.*", "^Mock.*"})}
    };
    auto p2 = x::json::Parser(j2);
    ScanTaskConfig cfg2(p2);

    EXPECT_TRUE(cfg2.disabled);
    EXPECT_EQ(cfg2.rate.hz(), 10.0);
    EXPECT_EQ(cfg2.ignored_models.size(), 2);
}

/// @brief it should correctly identify models to ignore based on regex patterns.
TEST(ScanTaskTest, testConfigShouldIgnore) {
    x::json::json j = {
        {"ignored_models", x::json::json::array({"^Test.*", "^Mock.*", "PXI-.*"})}
    };
    auto p = x::json::Parser(j);
    ScanTaskConfig cfg(p);

    // Should ignore models matching the patterns
    EXPECT_TRUE(cfg.should_ignore("TestDevice"));
    EXPECT_TRUE(cfg.should_ignore("MockDevice123"));
    EXPECT_TRUE(cfg.should_ignore("PXI-6255"));

    // Should not ignore models not matching the patterns
    EXPECT_FALSE(cfg.should_ignore("NI-DAQ"));
    EXPECT_FALSE(cfg.should_ignore("Regular-Device"));
    EXPECT_FALSE(cfg.should_ignore("cDAQ-9178"));

    // Test with default configuration
    x::json::json j2 = {};
    auto p2 = x::json::Parser(j2);
    ScanTaskConfig cfg2(p2);

    // Should ignore models matching default patterns
    EXPECT_TRUE(cfg2.should_ignore("cRIO-9068"));
    EXPECT_TRUE(cfg2.should_ignore("nownDevice"));

    // Should not ignore models not matching default patterns
    EXPECT_FALSE(cfg2.should_ignore("PXI-6255"));
    EXPECT_FALSE(cfg2.should_ignore("NI-DAQ"));
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
}
