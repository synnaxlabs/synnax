// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { resolutionFor } from "@/ranger/Timeline";

describe("resolutionFor", () => {
  it("should coarsen with the span", () => {
    expect(resolutionFor(TimeSpan.days(2)).equals(TimeSpan.MINUTE)).toBe(true);
    expect(resolutionFor(TimeSpan.hours(2)).equals(TimeSpan.SECOND)).toBe(true);
    expect(resolutionFor(TimeSpan.minutes(5)).equals(TimeSpan.SECOND)).toBe(true);
    expect(resolutionFor(TimeSpan.seconds(15)).equals(TimeSpan.MILLISECOND)).toBe(true);
    expect(resolutionFor(TimeSpan.milliseconds(3)).equals(TimeSpan.MICROSECOND)).toBe(
      true,
    );
  });
});
