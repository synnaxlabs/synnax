// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { Telem } from "@/telem";

const NOW = new TimeStamp(new Date(2026, 7, 23, 14, 5, 0, 0));

describe("describeDay", () => {
  it("should name nearby days and fall back to month and day", () => {
    expect(Telem.Text.describeDay(NOW, NOW)).toBe("Today");
    expect(Telem.Text.describeDay(NOW.sub(TimeSpan.DAY), NOW)).toBe("Yesterday");
    expect(Telem.Text.describeDay(NOW.add(TimeSpan.DAY), NOW)).toBe("Tomorrow");
    expect(Telem.Text.describeDay(NOW.sub(TimeSpan.days(3)), NOW)).toBe("Thursday");
    expect(Telem.Text.describeDay(NOW.sub(TimeSpan.days(30)), NOW)).toBe("Jul 24");
    expect(Telem.Text.describeDay(NOW.sub(TimeSpan.days(400)), NOW)).toBe(
      "Jul 19, 2025",
    );
  });
});

describe("formatTime", () => {
  it("should drop zero seconds and keep set precision", () => {
    expect(Telem.Text.formatTime(NOW)).toBe("14:05");
    expect(Telem.Text.formatTime(NOW.add(TimeSpan.seconds(32)))).toBe("14:05:32");
    expect(Telem.Text.formatTime(NOW.add(TimeSpan.milliseconds(250)))).toBe(
      "14:05:00.250",
    );
  });

  it("should drop label digits below the resolution", () => {
    const ts = NOW.add(TimeSpan.seconds(32)).add(TimeSpan.milliseconds(250));
    expect(Telem.Text.formatTime(ts, TimeSpan.MINUTE)).toBe("14:05");
    expect(Telem.Text.formatTime(ts, TimeSpan.SECOND)).toBe("14:05:32");
    expect(Telem.Text.formatTime(ts)).toBe("14:05:32.250");
  });
});
