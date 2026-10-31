// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { filter, TimeRange, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { type channel } from "@/channel";
import { ranger } from "@/ranger";

describe("builder", () => {
  it("should build a request that the request schema accepts", () => {
    const key = "00000000-0000-4000-8000-000000000001";
    const label = "00000000-0000-4000-8000-000000000002";
    const window = new TimeRange(TimeStamp.seconds(1), TimeStamp.seconds(2));
    const request = filter.toRequest([
      ranger.or(
        ranger.name.startsWith("hot", { caseInsensitive: true }),
        ranger.labels.has(label),
      ),
      ranger.not(ranger.key.in([key])),
      ranger.overlaps(window),
      ranger.limit(5),
      ranger.includeLabels(),
    ]);
    expect(ranger.retrieveRequestZ.parse(request)).toEqual({
      where: {
        and: [
          {
            or: [
              { name: { startsWith: "hot", caseInsensitive: true } },
              { labels: [label] },
            ],
          },
          { not: { key: { in: [key] } } },
          { overlaps: window },
        ],
      },
      limit: 5,
      includeLabels: true,
    });
  });

  it("should reject a condition of another entity", () => {
    // @ts-expect-error a range condition is not a channel condition
    const condition: channel.Condition = ranger.name.eq("Hotfire");
    expect(condition.tree).toEqual({ name: { eq: "Hotfire" } });
  });
});
