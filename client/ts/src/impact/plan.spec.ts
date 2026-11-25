// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { errors } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

// The channel module loads first, before errors and impact. Decoding the error below
// must not need impact's schema, which reads the channel module as it loads.
import { channel } from "@/channel";
import { impact } from "@/impact";
import { PlanChangedError } from "@/index";

describe("planOf", () => {
  it("should parse the plan that a decoded plan-changed error carries", () => {
    const keys = [channel.keyZ.parse(1048577)];
    const sent = {
      entries: [
        {
          record: { type: "lineplot", key: "8c1b8c4e-6c1e-4a5e-9a39-4f1a3e0c2b11" },
          effect: "detach",
          choices: [],
        },
      ],
      kept: 0,
      hidden: { restrict: 0, cascade: 0, detach: 1 },
      token: "0123456789abcdef",
      change: { type: "delete_channels", keys },
    };
    const decoded = errors.decode({
      type: "sy.impact.plan_changed",
      data: JSON.stringify(sent),
    });
    expect(decoded).toSatisfy(PlanChangedError.matches);
    const changed = decoded as PlanChangedError;
    expect(changed.message).toEqual("the records that the change touches changed");
    expect(impact.planOf(changed)).toEqual(impact.planZ.parse(sent));
  });
});
