// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { sleep } from "@/sleep";
import { TimeSpan } from "@/telem";
import { timer } from "@/timer";

describe("timer", () => {
  describe("start", () => {
    it("should expire after the span", async () => {
      const span = TimeSpan.milliseconds(20);
      const started = performance.now();
      await timer.start(span).expired;
      // Node can fire a timer up to a millisecond early.
      expect(performance.now() - started).toBeGreaterThanOrEqual(span.milliseconds - 1);
    });

    it("should never expire after cancel", async () => {
      const t = timer.start(TimeSpan.milliseconds(5));
      let expired = false;
      void t.expired.then(() => {
        expired = true;
      });
      t.cancel();
      await sleep.sleep(TimeSpan.milliseconds(30));
      expect(expired).toBe(false);
    });

    it("should allow cancel more than once", () => {
      const t = timer.start(TimeSpan.milliseconds(5));
      t.cancel();
      expect(() => t.cancel()).not.toThrow();
    });
  });
});
