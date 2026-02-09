// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <sstream>

#include "gtest/gtest.h"

#include "x/cpp/telem/telem.h"
#include "x/cpp/test/test.h"
#include "x/cpp/thread/rt/rt.h"

namespace x::thread::rt {
TEST(RTConfigTest, DefaultConfig) {
    const Config cfg;
    EXPECT_FALSE(cfg.enabled);
    EXPECT_EQ(cfg.priority, DEFAULT_PRIORITY);
    EXPECT_EQ(cfg.cpu_affinity, CPU_AFFINITY_NONE);
    EXPECT_FALSE(cfg.lock_memory);
    EXPECT_EQ(cfg.period, telem::TimeSpan::ZERO());
    EXPECT_EQ(cfg.computation, telem::TimeSpan::ZERO());
    EXPECT_EQ(cfg.deadline, telem::TimeSpan::ZERO());
    EXPECT_FALSE(cfg.prefer_deadline_scheduler);
    EXPECT_FALSE(cfg.use_mmcss);
}

TEST(RTConfigTest, HasTiming) {
    Config cfg;
    EXPECT_FALSE(cfg.has_timing());
    cfg.period = telem::MILLISECOND;
    EXPECT_TRUE(cfg.has_timing());
}

TEST(RTConfigTest, WithTimingDefaults) {
    Config cfg;
    EXPECT_FALSE(cfg.has_timing());
    auto with_defaults = cfg.with_timing_defaults();
    EXPECT_TRUE(with_defaults.has_timing());
    EXPECT_EQ(with_defaults.period, DEFAULT_PERIOD);
    EXPECT_EQ(with_defaults.computation, DEFAULT_COMPUTATION);
    EXPECT_EQ(with_defaults.deadline, DEFAULT_DEADLINE);
}

TEST(RTConfigTest, WithTimingDefaultsPreservesExisting) {
    Config cfg;
    cfg.period = telem::MILLISECOND * 2;
    cfg.computation = telem::MICROSECOND * 400;
    cfg.deadline = telem::MICROSECOND * 800;
    auto with_defaults = cfg.with_timing_defaults();
    EXPECT_EQ(with_defaults.period, telem::MILLISECOND * 2);
    EXPECT_EQ(with_defaults.computation, telem::MICROSECOND * 400);
    EXPECT_EQ(with_defaults.deadline, telem::MICROSECOND * 800);
}

TEST(RTConfigTest, ApplyEmptyConfig) {
    Config cfg;
    ASSERT_NIL(apply_config(cfg));
}

TEST(RTConfigTest, ApplyWithRTEnabled) {
    Config cfg;
    cfg.enabled = true;
    cfg.priority = 50;
    ASSERT_NIL(apply_config(cfg));
}

TEST(RTConfigTest, ApplyWithTiming) {
    Config cfg;
    cfg.enabled = true;
    cfg.period = telem::MILLISECOND;
    cfg.computation = telem::MICROSECOND * 200;
    cfg.deadline = telem::MICROSECOND * 500;
    ASSERT_NIL(apply_config(cfg));
}

TEST(RTConfigTest, ApplyWithDeadlineScheduler) {
    Config cfg;
    cfg.enabled = true;
    cfg.period = telem::MILLISECOND;
    cfg.computation = telem::MICROSECOND * 200;
    cfg.deadline = telem::MICROSECOND * 500;
    cfg.prefer_deadline_scheduler = true;
    ASSERT_NIL(apply_config(cfg));
}

TEST(RTConfigTest, ApplyWithMMCSS) {
    Config cfg;
    cfg.enabled = true;
    cfg.use_mmcss = true;
    ASSERT_NIL(apply_config(cfg));
}

TEST(RTConfigTest, HasRTSupportReturns) {
    [[maybe_unused]] bool supported = has_support();
}

TEST(RTCapabilitiesTest, GetCapabilities) {
    auto caps = get_capabilities();
    (void) caps.priority_scheduling;
    (void) caps.deadline_scheduling;
    (void) caps.time_constraint;
    (void) caps.mmcss;
    (void) caps.cpu_affinity;
    (void) caps.memory_locking;
}

TEST(CapabilityTest, DefaultState) {
    Capability cap;
    EXPECT_FALSE(cap.supported);
    EXPECT_FALSE(cap.permitted);
    EXPECT_FALSE(cap.ok());
    EXPECT_FALSE(cap);
}

TEST(CapabilityTest, SupportedOnly) {
    Capability cap{true, false};
    EXPECT_TRUE(cap.supported);
    EXPECT_FALSE(cap.permitted);
    EXPECT_FALSE(cap.ok());
    EXPECT_FALSE(cap);
    EXPECT_TRUE(cap.missing_permissions());
}

TEST(CapabilityTest, FullyEnabled) {
    Capability cap{true, true};
    EXPECT_TRUE(cap.supported);
    EXPECT_TRUE(cap.permitted);
    EXPECT_TRUE(cap.ok());
    EXPECT_TRUE(cap);
    EXPECT_FALSE(cap.missing_permissions());
}

TEST(RTCapabilitiesTest, Any) {
    Capabilities caps;
    EXPECT_FALSE(caps.any());
    caps.priority_scheduling = {true, true};
    EXPECT_TRUE(caps.any());
}

TEST(RTCapabilitiesTest, TimingAware) {
    Capabilities caps;
    EXPECT_FALSE(caps.timing_aware());
    caps.deadline_scheduling = {true, true};
    EXPECT_TRUE(caps.timing_aware());
    caps.deadline_scheduling = {false, false};
    caps.time_constraint = {true, true};
    EXPECT_TRUE(caps.timing_aware());
}

TEST(RTCapabilitiesTest, HasPermissionIssues) {
    Capabilities caps;
    EXPECT_FALSE(caps.has_permission_issues());
    caps.priority_scheduling = {true, false};
    EXPECT_TRUE(caps.has_permission_issues());
}

TEST(RTCapabilitiesTest, OstreamOperator) {
    Capabilities caps;
    caps.priority_scheduling = {true, true};
    caps.cpu_affinity = {true, true};
    std::ostringstream oss;
    oss << caps;
    EXPECT_NE(oss.str().find("priority scheduling"), std::string::npos);
    EXPECT_NE(oss.str().find("cpu affinity"), std::string::npos);
}

TEST(RTCapabilitiesTest, OstreamShowsMissingPermissions) {
    Capabilities caps;
    caps.priority_scheduling = {true, false};
    std::ostringstream oss;
    oss << caps;
    EXPECT_NE(oss.str().find("missing permissions"), std::string::npos);
}

TEST(RTConfigTest, OstreamOperator) {
    Config cfg;
    cfg.enabled = true;
    cfg.period = telem::MILLISECOND;
    cfg.computation = telem::MICROSECOND * 200;
    cfg.deadline = telem::MICROSECOND * 500;
    std::ostringstream oss;
    oss << cfg;
    EXPECT_NE(oss.str().find("enabled"), std::string::npos);
    EXPECT_NE(oss.str().find("period"), std::string::npos);
}
}
