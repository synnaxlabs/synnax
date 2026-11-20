// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Ranger } from "@synnaxlabs/pluto";
import { TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { pin } from "@/feature/range/explorer/time";

const AT = TimeStamp.seconds(1000);
const DAY = TimeSpan.days(1);

describe("pin", () => {
  it("should read a recent span back from the instant", () => {
    expect(pin({ recent: DAY }, AT)).toEqual({
      and: [{ overlaps: new TimeRange(AT.sub(DAY), AT) }],
    });
  });

  it("should keep the other fields beside the pinned span", () => {
    expect(pin({ labeled: true, recent: DAY }, AT)).toEqual({
      and: [{ labeled: true }, { overlaps: new TimeRange(AT.sub(DAY), AT) }],
    });
  });

  it("should pin the stages and spans inside and, or, and not", () => {
    const inProgress = { and: [{ or: [Ranger.stageWhere("in_progress", AT)] }] };
    const recent = { and: [{ overlaps: new TimeRange(AT.sub(DAY), AT) }] };
    expect(
      pin(
        {
          and: [{ stages: ["in_progress"] }, { labeled: true }],
          or: [{ recent: DAY }],
          not: { stages: ["in_progress"] },
        },
        AT,
      ),
    ).toEqual({
      and: [inProgress, { labeled: true }],
      or: [recent],
      not: inProgress,
    });
  });

  it("should return a filter that reads no time as it is", () => {
    expect(pin({ labeled: true }, AT)).toEqual({ labeled: true });
  });
});
