// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { PagerDuty } from "@/feature/pagerduty";
import { createWithHook } from "@/platform/task/testutil";

describe("PagerDuty.Task create", () => {
  it("should create an alert task with defaults when the config is not valid", async () => {
    const tsk = await createWithHook(
      PagerDuty.Task.useCreateAlert,
      PagerDuty.Task.ALERT_SCHEMAS,
      { config: { alerts: "none" } },
    );
    expect(tsk.config.alerts).toEqual([]);
  });
});
